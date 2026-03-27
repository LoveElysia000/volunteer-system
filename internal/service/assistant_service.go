package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
	"volunteer-system/config"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/ai"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// assistant_service.go 实现 AI 助手核心编排服务。
//
// 主要职责：
// 1. 管理会话生命周期：创建会话、校验归属、更新最后活跃时间与标题。
// 2. 编排对话链路：写入用户消息 -> 规划/执行工具 -> 构建上下文 -> 调用 AI -> 落库回复。
// 3. 处理失败降级：当模型调用失败时，基于工具结果生成可用的 fallback 回复。
// 4. 维护消息一致性：通过 seq_no 重试机制降低并发写入冲突风险。
// 5. 记录观测数据：保存 token、延迟、finish_reason、工具调用日志与日聚合用量。
// 6. 提供场景化能力：通用问答、活动草案、运营分析等场景提示与快捷入口。

const (
	assistantSceneGeneral       = "general"
	assistantSceneActivityDraft = "activity_draft"
	assistantSceneOpsAdvisor    = "ops_advisor"

	assistantSessionStatusActive = 1

	assistantRoleSystem    = 1
	assistantRoleUser      = 2
	assistantRoleAssistant = 3
	assistantRoleTool      = 4

	assistantMaxSeqRetry         = 3
	assistantDefaultContextLimit = 20
	assistantMinContextLimit     = 4
	assistantMaxContextLimit     = 100
	assistantMaxUserMessageRunes = 2000
)

type assistantToolPlan struct {
	ToolName string
	Input    map[string]any
}

type assistantToolResult struct {
	LogID      int64
	ToolName   string
	InputJSON  string
	OutputJSON string
	Success    bool
	ErrorCode  string
	ErrorMsg   string
	LatencyMS  int32
}

// AssistantService AI 助手服务
type AssistantService struct {
	Service
	toolService *AssistantToolService
}

// NewAssistantService 创建 AI 助手服务实例
func NewAssistantService(ctx context.Context, c *app.RequestContext) *AssistantService {
	if ctx == nil {
		ctx = context.Background()
	}
	svc := &AssistantService{
		Service: Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
	svc.toolService = NewAssistantToolService(ctx, c)
	return svc
}

// CreateSession 创建会话
func (s *AssistantService) CreateSession(req *api.AssistantCreateSessionRequest) (*api.AssistantCreateSessionResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}

	accountID, err := s.currentAccountID()
	if err != nil {
		return nil, err
	}

	scene, err := normalizeAssistantScene(req.Scene)
	if err != nil {
		return nil, err
	}

	// 未传标题时按场景自动生成默认标题。
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = defaultSessionTitle(scene)
	}

	session, err := s.createSessionForUser(accountID, scene, title)
	if err != nil {
		return nil, err
	}

	return &api.AssistantCreateSessionResponse{SessionId: session.ID}, nil
}

// Chat AI 对话
func (s *AssistantService) Chat(req *api.AssistantChatRequest) (*api.AssistantChatResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}

	if req.Stream {
		return nil, errors.New("暂不支持流式输出，请将 stream 设为 false")
	}

	message, err := normalizeUserMessage(req.Message)
	if err != nil {
		return nil, err
	}

	accountID, err := s.currentAccountID()
	if err != nil {
		return nil, err
	}

	session, err := s.repo.GetAiSessionByIDAndUser(s.repo.DB, req.SessionId, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("会话不存在")
		}
		return nil, err
	}
	if session.Status != assistantSessionStatusActive {
		return nil, errors.New("会话已归档，无法继续对话")
	}

	cfg := s.getAIConfig()
	requestID := s.resolveRequestID()
	now := time.Now()
	// 先原子扣减请求配额，再进入后续流程，避免并发下超额放行。
	if err := s.checkDailyUserQuota(now, accountID, cfg); err != nil {
		return nil, err
	}
	usageOutcomeRecorded := false
	usageSuccess := false
	usageTokenIn := int32(0)
	usageTokenOut := int32(0)
	usageEstimatedCost := float64(0)
	// 任意中途 return 都会走该 defer：确保请求已扣配额时，最终结果指标至少被记录一次。
	defer func() {
		if usageOutcomeRecorded {
			return
		}
		if err := s.repo.AppendAiUsageOutcome(s.repo.DB, now, accountID, usageSuccess, usageTokenIn, usageTokenOut, usageEstimatedCost); err != nil {
			log.Error("补记 AI 失败用量失败: %v, account_id=%d", err, accountID)
		}
	}()

	userMessage := &model.AiMessage{
		SessionID: session.ID,
		Role:      assistantRoleUser,
		Content:   message,
		RequestID: requestID,
		CreatedAt: now,
	}
	if err := s.appendAiMessage(userMessage); err != nil {
		log.Error("写入用户消息失败: %v, session_id=%d account_id=%d", err, session.ID, accountID)
		return nil, err
	}

	contextLimit := resolveContextLimit(cfg)
	// 历史消息用于模型上下文，不需要全量拉取，避免 token 与延迟失控。
	historyRows, err := s.repo.ListRecentAiMessagesBySession(s.repo.DB, session.ID, contextLimit)
	if err != nil {
		log.Error("查询会话上下文失败: %v, session_id=%d", err, session.ID)
		return nil, err
	}

	// 反转为正序
	util.ReverseInPlace(historyRows)

	runtimeInput := &runtimeChatInput{
		UserID:    accountID,
		SessionID: session.ID,
		Scene:     session.Scene,
		Message:   message,
		History:   snapshotsFromMessages(historyRows),
		RequestID: requestID,
	}
	// 运行时失败时降级为 fallback 输出，尽量保持用户可用体验。
	runtimeOutput, runtimeErr := s.runRuntime(runtimeInput)
	if runtimeErr != nil {
		log.Warn("AI runtime execute failed, using fallback output: %v, session_id=%d account_id=%d", runtimeErr, session.ID, accountID)
		runtimeOutput = s.buildRuntimeFallbackOutput(runtimeOutput, runtimeErr)
	}
	if runtimeOutput == nil {
		return nil, errors.New("AI 运行时返回为空")
	}

	if strings.TrimSpace(runtimeOutput.Reply) == "" {
		runtimeOutput.Reply = "我已收到你的问题，但当前无法生成有效回复，请稍后重试。"
	}
	if strings.TrimSpace(runtimeOutput.Model) == "" {
		runtimeOutput.Model = s.getChatModel()
	}

	toolResults := runtimeToolCallsToAssistantResults(runtimeOutput.ToolCalls)
	// 工具结果先写消息与工具日志，再绑定到本次 assistant 消息。
	toolLogIDs := s.persistRuntimeToolResults(session.ID, accountID, requestID, toolResults)

	assistantMsg := &model.AiMessage{
		SessionID:    session.ID,
		Role:         assistantRoleAssistant,
		Content:      runtimeOutput.Reply,
		Model:        runtimeOutput.Model,
		FinishReason: mapFinishReason(runtimeOutput.FinishReason),
		TokenIn:      runtimeOutput.TokenIn,
		TokenOut:     runtimeOutput.TokenOut,
		LatencyMs:    runtimeOutput.LatencyMS,
		RequestID:    requestID,
		CreatedAt:    time.Now(),
	}
	if err := s.appendAiMessage(assistantMsg); err != nil {
		log.Error("写入助手消息失败: %v, session_id=%d account_id=%d", err, session.ID, accountID)
		return nil, err
	}

	if len(toolLogIDs) > 0 {
		if err := s.repo.UpdateAiToolCallMessageID(s.repo.DB, toolLogIDs, assistantMsg.ID); err != nil {
			log.Error("绑定工具日志消息ID失败: %v, session_id=%d message_id=%d", err, session.ID, assistantMsg.ID)
		}
	}

	title := ""
	if strings.TrimSpace(session.Title) == "" {
		title = s.generateSessionTitle(message, session.Scene)
	}
	if err := s.repo.UpdateAiSessionAfterMessage(s.repo.DB, session.ID, assistantMsg.CreatedAt, title); err != nil {
		log.Error("更新会话时间失败: %v, session_id=%d", err, session.ID)
	}

	runtimeSuccess := runtimeOutput.Success && runtimeErr == nil
	estimatedCost := s.estimateCost(runtimeOutput.Model, runtimeOutput.TokenIn, runtimeOutput.TokenOut)
	usageSuccess = runtimeSuccess
	usageTokenIn = runtimeOutput.TokenIn
	usageTokenOut = runtimeOutput.TokenOut
	usageEstimatedCost = estimatedCost
	if err := s.repo.AppendAiUsageOutcome(s.repo.DB, now, accountID, runtimeSuccess, runtimeOutput.TokenIn, runtimeOutput.TokenOut, estimatedCost); err != nil {
		log.Error("更新 AI 用量失败: %v, account_id=%d", err, accountID)
	} else {
		// 主路径写入成功后，关闭 defer 补记。
		usageOutcomeRecorded = true
	}

	return &api.AssistantChatResponse{
		Reply:     runtimeOutput.Reply,
		ToolCalls: toAPIToolCalls(toolResults),
		Usage: &api.AssistantUsage{
			Model:     runtimeOutput.Model,
			TokenIn:   runtimeOutput.TokenIn,
			TokenOut:  runtimeOutput.TokenOut,
			LatencyMs: runtimeOutput.LatencyMS,
		},
	}, nil
}
func (s *AssistantService) GetSessionMessages(req *api.AssistantSessionMessagesRequest) (*api.AssistantSessionMessagesResponse, error) {
	if req == nil || req.Id <= 0 {
		return nil, errors.New("会话ID不能为空")
	}

	accountID, err := s.currentAccountID()
	if err != nil {
		return nil, err
	}

	_, err = s.repo.GetAiSessionByIDAndUser(s.repo.DB, req.Id, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("会话不存在")
		}
		return nil, err
	}

	messages, err := s.repo.ListAiMessagesBySession(s.repo.DB, req.Id)
	if err != nil {
		return nil, err
	}

	items := make([]*api.AssistantMessageItem, 0, len(messages))
	for _, m := range messages {
		if m == nil {
			continue
		}
		// 统一做 API DTO 转换，避免直接暴露 model 结构。
		items = append(items, &api.AssistantMessageItem{
			Id:           m.ID,
			SessionId:    m.SessionID,
			SeqNo:        m.SeqNo,
			Role:         m.Role,
			Content:      m.Content,
			Model:        m.Model,
			FinishReason: m.FinishReason,
			TokenIn:      m.TokenIn,
			TokenOut:     m.TokenOut,
			LatencyMs:    m.LatencyMs,
			RequestId:    m.RequestID,
			CreatedAt:    m.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &api.AssistantSessionMessagesResponse{List: items}, nil
}

// ActivityDraftAction 活动草案快捷入口
func (s *AssistantService) ActivityDraftAction(req *api.AssistantActivityDraftActionRequest) (*api.AssistantActivityDraftActionResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if strings.TrimSpace(req.Topic) == "" {
		return nil, errors.New("主题不能为空")
	}

	accountID, err := s.currentAccountID()
	if err != nil {
		return nil, err
	}

	sessionID := req.SessionId
	if sessionID <= 0 {
		// 草案快捷入口允许无会话调用：内部自动创建会话承接上下文。
		session, err := s.createSessionForUser(accountID, assistantSceneActivityDraft, s.generateSessionTitle(req.Topic, assistantSceneActivityDraft))
		if err != nil {
			return nil, err
		}
		sessionID = session.ID
	} else {
		if _, err := s.repo.GetAiSessionByIDAndUser(s.repo.DB, sessionID, accountID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("会话不存在")
			}
			return nil, err
		}
	}

	message := fmt.Sprintf(
		"请生成活动草案，主题：%s。目标人群：%s。活动地点：%s。",
		strings.TrimSpace(req.Topic),
		strings.TrimSpace(req.TargetPeople),
		strings.TrimSpace(req.Location),
	)
	// 复用统一 Chat 链路，保证草案入口和普通对话的审计/记账行为一致。
	chatResp, err := s.Chat(&api.AssistantChatRequest{
		SessionId: sessionID,
		Message:   message,
		Stream:    false,
	})
	if err != nil {
		return nil, err
	}

	return &api.AssistantActivityDraftActionResponse{
		SessionId: sessionID,
		Result:    chatResp,
	}, nil
}

func (s *AssistantService) createSessionForUser(userID int64, scene, title string) (*model.AiSession, error) {
	session := &model.AiSession{
		UserID:        userID,
		Scene:         scene,
		Title:         util.TruncateText(strings.TrimSpace(title), 128),
		Status:        assistantSessionStatusActive,
		Summary:       "",
		LastMessageAt: nil,
	}
	if session.Title == "" {
		// 再次兜底，防止上游传入空白标题。
		session.Title = defaultSessionTitle(scene)
	}
	if err := s.repo.CreateAiSession(s.repo.DB, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *AssistantService) appendAiMessage(message *model.AiMessage) error {
	if message == nil {
		return errors.New("消息不能为空")
	}
	if message.SessionID <= 0 {
		return errors.New("会话ID不能为空")
	}
	if strings.TrimSpace(message.RequestID) == "" {
		message.RequestID = s.resolveRequestID()
	}

	// 通过唯一索引 (session_id, seq_no) + 重试，避免并发写入导致序号冲突。
	for i := 0; i < assistantMaxSeqRetry; i++ {
		// 每次重试都重新申请 seq_no，避免复用冲突序号。
		seqNo, err := s.repo.GetNextAiMessageSeqNo(s.repo.DB, message.SessionID)
		if err != nil {
			return err
		}
		message.SeqNo = seqNo
		if err := s.repo.CreateAiMessage(s.repo.DB, message); err != nil {
			if isDuplicateSeqError(err) {
				// 仅对并发冲突重试，其他错误直接返回。
				continue
			}
			return err
		}
		return nil
	}

	return errors.New("消息写入冲突，请重试")
}

// buildPromptMessages 将会话历史转换为 Chat Completions 上下文。
func (s *AssistantService) buildPromptMessages(scene string, history []*model.AiMessage) []ai.Message {
	messages := make([]ai.Message, 0, len(history)+1)
	// 第一条固定为系统提示，约束模型角色和行为边界。
	messages = append(messages, ai.Message{
		Role:    "system",
		Content: s.buildSystemPrompt(scene),
	})

	for _, item := range history {
		if item == nil {
			continue
		}
		content := strings.TrimSpace(item.Content)
		if content == "" {
			continue
		}

		switch item.Role {
		case assistantRoleUser:
			messages = append(messages, ai.Message{Role: "user", Content: content})
		case assistantRoleAssistant:
			messages = append(messages, ai.Message{Role: "assistant", Content: content})
		case assistantRoleTool:
			// 旧版 chat completion 链路中，工具结果通过 system 消息注入上下文。
			messages = append(messages, ai.Message{Role: "system", Content: "工具结果(JSON):\n" + content})
		}
	}

	return messages
}

func (s *AssistantService) buildSystemPrompt(scene string) string {
	base := "你是环保志愿者平台的 AI 助手。回答要准确、简洁、可执行。禁止编造不存在的数据。涉及权限或隐私时必须拒绝并解释原因。"
	switch scene {
	case assistantSceneActivityDraft:
		return base + "当前场景是活动草案生成，请优先给出活动标题、简介、执行流程和风险提示。"
	case assistantSceneOpsAdvisor:
		return base + "当前场景是组织运营分析，请优先基于活动统计结果给出可落地建议。"
	default:
		return base + "当前场景是通用问答。"
	}
}

func (s *AssistantService) buildFallbackReply(toolResults []*assistantToolResult, cause error) string {
	reason := resolveUserVisibleModelError(cause)
	if len(toolResults) == 0 {
		return "AI 服务暂时不可用（" + reason + "），请稍后重试。"
	}

	// 有工具结果时优先给出可执行信息，尽量降低模型失败的可用性损失。
	lines := make([]string, 0, len(toolResults)+2)
	lines = append(lines, "AI 服务暂时不可用，我先基于系统工具结果给你结论：")
	for _, item := range toolResults {
		if item == nil {
			continue
		}
		if item.Success {
			lines = append(lines, fmt.Sprintf("- %s: %s", item.ToolName, util.TruncateText(item.OutputJSON, 280)))
		} else {
			lines = append(lines, fmt.Sprintf("- %s 调用失败: %s", item.ToolName, item.ErrorMsg))
		}
	}
	lines = append(lines, "失败原因: "+reason)
	lines = append(lines, "你可以稍后重试，或缩小问题范围（例如指定活动关键词、组织ID）以提高成功率。")
	return strings.Join(lines, "\n")
}

func (s *AssistantService) checkDailyUserQuota(now time.Time, userID int64, cfg *config.AIConfig) error {
	dailyQuota := 0
	if cfg != nil && cfg.DailyUserQuota > 0 {
		dailyQuota = cfg.DailyUserQuota
	}
	// 原子配额消费：避免“先查后改”在高并发时被绕过。
	consumed, err := s.repo.ConsumeAiRequestQuota(s.repo.DB, now, userID, dailyQuota)
	if err != nil {
		log.Error("查询 AI 日配额失败: %v, user_id=%d", err, userID)
		return errors.New("系统繁忙，请稍后重试")
	}
	if !consumed {
		return errors.New("今日 AI 使用次数已达上限，请明日再试")
	}
	return nil
}

// persistRuntimeToolResults 同步落库工具结果：
// 1) 写入 tool 角色消息，参与后续上下文；
// 2) 写入 ai_tool_calls 结构化日志，便于审计与追踪。
func (s *AssistantService) persistRuntimeToolResults(sessionID, userID int64, requestID string, toolResults []*assistantToolResult) []int64 {
	toolLogIDs := make([]int64, 0, len(toolResults))
	for _, result := range toolResults {
		if result == nil {
			continue
		}

		toolMessage := &model.AiMessage{
			SessionID: sessionID,
			Role:      assistantRoleTool,
			Content:   nonEmptyJSON(result.OutputJSON),
			RequestID: requestID,
			CreatedAt: time.Now(),
		}
		if err := s.appendAiMessage(toolMessage); err != nil {
			log.Error("写入工具消息失败: %v, session_id=%d user_id=%d tool=%s", err, sessionID, userID, result.ToolName)
		}

		var output *string
		if strings.TrimSpace(result.OutputJSON) != "" {
			// 仅在非空时记录工具输出，数据库可区分“无输出”和“空 JSON”。
			value := result.OutputJSON
			output = &value
		}

		toolCall := &model.AiToolCall{
			SessionID:  sessionID,
			ToolName:   result.ToolName,
			ToolInput:  nonEmptyJSON(result.InputJSON),
			ToolOutput: output,
			Success:    boolToInt32(result.Success),
			ErrorCode:  util.TruncateText(result.ErrorCode, 64),
			ErrorMsg:   util.TruncateText(result.ErrorMsg, 255),
			LatencyMs:  result.LatencyMS,
		}
		if err := s.repo.CreateAiToolCall(s.repo.DB, toolCall); err != nil {
			log.Error("写入工具调用日志失败: %v, session_id=%d tool=%s", err, sessionID, result.ToolName)
			continue
		}

		// 回写 logID，后续会绑定到 assistant 消息形成闭环。
		result.LogID = toolCall.ID
		toolLogIDs = append(toolLogIDs, toolCall.ID)
	}
	return toolLogIDs
}

func (s *AssistantService) resolveRequestID() string {
	if s.c != nil {
		// 优先透传网关/上游注入的请求 ID，便于链路追踪。
		if rid := strings.TrimSpace(string(s.c.GetHeader("X-Request-Id"))); rid != "" {
			return rid
		}
		if rid := strings.TrimSpace(string(s.c.GetHeader("X-Request-ID"))); rid != "" {
			return rid
		}
	}
	return uuid.NewString()
}

func (s *AssistantService) getChatModel() string {
	cfg := s.getAIConfig()
	if cfg == nil || strings.TrimSpace(cfg.ChatModel) == "" {
		return "gpt-4o-mini"
	}
	return strings.TrimSpace(cfg.ChatModel)
}

func (s *AssistantService) getAIConfig() *config.AIConfig {
	cfg := config.GetConfig()
	if cfg == nil {
		return nil
	}
	return cfg.AI
}

func resolveContextLimit(cfg *config.AIConfig) int {
	limit := assistantDefaultContextLimit
	if cfg != nil && cfg.MaxContextMessages > 0 {
		limit = cfg.MaxContextMessages
	}
	if limit < assistantMinContextLimit {
		return assistantDefaultContextLimit
	}
	if limit > assistantMaxContextLimit {
		return assistantMaxContextLimit
	}
	return limit
}

func (s *AssistantService) estimateCost(modelName string, tokenIn, tokenOut int32) float64 {
	if tokenIn <= 0 && tokenOut <= 0 {
		return 0
	}

	// 这里是估算模型成本，不作为计费依据，仅用于运营观测。
	name := strings.ToLower(strings.TrimSpace(modelName))
	inputRate := 0.00015
	outputRate := 0.00060

	if strings.Contains(name, "deepseek") {
		inputRate = 0.00014
		outputRate = 0.00028
	} else if strings.Contains(name, "gpt-4o") && !strings.Contains(name, "mini") {
		inputRate = 0.005
		outputRate = 0.015
	}

	return (float64(tokenIn)/1000.0)*inputRate + (float64(tokenOut)/1000.0)*outputRate
}

func (s *AssistantService) generateSessionTitle(message, scene string) string {
	base := strings.TrimSpace(message)
	if base == "" {
		return defaultSessionTitle(scene)
	}
	// 标题去换行并截断，避免 UI 显示过长。
	base = strings.ReplaceAll(base, "\n", " ")
	base = strings.TrimSpace(base)
	if len(base) > 24 {
		base = base[:24]
	}
	return base
}

func normalizeAssistantScene(scene string) (string, error) {
	v := strings.TrimSpace(scene)
	if v == "" {
		return assistantSceneGeneral, nil
	}
	switch v {
	case assistantSceneGeneral, assistantSceneActivityDraft, assistantSceneOpsAdvisor:
		return v, nil
	default:
		return "", errors.New("不支持的会话场景")
	}
}

func defaultSessionTitle(scene string) string {
	switch scene {
	case assistantSceneActivityDraft:
		return "活动草案助手"
	case assistantSceneOpsAdvisor:
		return "组织运营助手"
	default:
		return "智能问答"
	}
}

func toAPIToolCalls(results []*assistantToolResult) []*api.AssistantToolCall {
	items := make([]*api.AssistantToolCall, 0, len(results))
	for _, item := range results {
		if item == nil {
			continue
		}
		items = append(items, &api.AssistantToolCall{
			ToolName:  item.ToolName,
			Success:   item.Success,
			ErrorCode: item.ErrorCode,
			ErrorMsg:  item.ErrorMsg,
			LatencyMs: item.LatencyMS,
			Input:     nonEmptyJSON(item.InputJSON),
			Output:    nonEmptyJSON(item.OutputJSON),
		})
	}
	return items
}

func snapshotsFromMessages(history []*model.AiMessage) []*aiMessageSnapshot {
	items := make([]*aiMessageSnapshot, 0, len(history))
	for _, m := range history {
		if m == nil {
			continue
		}
		items = append(items, &aiMessageSnapshot{
			Role:    m.Role,
			Content: m.Content,
		})
	}
	return items
}

func nonEmptyJSON(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "{}"
	}
	return raw
}

func boolToInt32(v bool) int32 {
	if v {
		return 1
	}
	return 0
}

// mapFinishReason 将 provider finish_reason 文本映射为内部枚举。
func mapFinishReason(reason string) int32 {
	switch strings.TrimSpace(strings.ToLower(reason)) {
	case "stop":
		return 1
	case "length":
		return 2
	case "content_filter":
		return 3
	case "tool_calls":
		return 4
	default:
		return 0
	}
}

func isDuplicateSeqError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "uk_ai_messages_session_seq") || strings.Contains(msg, "duplicate entry")
}

func normalizeUserMessage(raw string) (string, error) {
	message := strings.TrimSpace(raw)
	if message == "" {
		return "", errors.New("消息不能为空")
	}
	if utf8.RuneCountInString(message) > assistantMaxUserMessageRunes {
		return "", fmt.Errorf("消息过长，最大 %d 字符", assistantMaxUserMessageRunes)
	}
	return message, nil
}

func resolveUserVisibleModelError(err error) string {
	if err == nil {
		return "模型服务暂时不可用"
	}
	if errors.Is(err, ai.ErrAPIKeyMissing) {
		return "AI 配置异常"
	}
	if errors.Is(err, ai.ErrAIUnavailable) {
		return "AI 服务未启用"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "模型服务响应超时"
	}

	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "status=429") ||
		strings.Contains(lower, "too many requests") ||
		strings.Contains(lower, "rate limit") {
		return "模型服务限流"
	}
	if strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded") {
		return "模型服务响应超时"
	}

	return "模型服务暂时不可用"
}
