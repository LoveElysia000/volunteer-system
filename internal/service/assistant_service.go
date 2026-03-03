package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"volunteer-system/config"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/ai"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	assistantSceneGeneral       = "general"
	assistantSceneActivityDraft = "activity_draft"
	assistantSceneOpsAdvisor    = "ops_advisor"

	assistantSessionStatusActive = 1

	assistantRoleSystem    = 1
	assistantRoleUser      = 2
	assistantRoleAssistant = 3
	assistantRoleTool      = 4

	assistantMaxSeqRetry = 3
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
	aiClient    *ai.Client
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
	svc.aiClient = ai.NewClient(config.GetConfig().AI)
	svc.toolService = NewAssistantToolService(ctx, c)
	return svc
}

// CreateSession 创建会话
func (s *AssistantService) CreateSession(req *api.AssistantCreateSessionRequest) (*api.AssistantCreateSessionResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	scene, err := normalizeAssistantScene(req.Scene)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = defaultSessionTitle(scene)
	}

	session, err := s.createSessionForUser(userID, scene, title)
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

	message := strings.TrimSpace(req.Message)
	if message == "" {
		return nil, errors.New("消息不能为空")
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	session, err := s.repo.GetAiSessionByIDAndUser(s.repo.DB, req.SessionId, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("会话不存在")
		}
		return nil, err
	}

	cfg := s.getAIConfig()

	requestID := s.resolveRequestID()
	now := time.Now()

	userMessage := &model.AiMessage{
		SessionID: session.ID,
		Role:      assistantRoleUser,
		Content:   message,
		RequestID: requestID,
		CreatedAt: now,
	}
	if err := s.appendAiMessage(userMessage); err != nil {
		log.Error("写入用户消息失败: %v, session_id=%d user_id=%d", err, session.ID, userID)
		return nil, err
	}

	// 先做受控工具调用，再把结果以 tool 消息写入上下文，供后续模型回答引用。
	toolPlans := s.toolService.PlanTools(session.Scene, message)
	toolResults := make([]*assistantToolResult, 0, len(toolPlans))
	toolLogIDs := make([]int64, 0, len(toolPlans))
	for _, plan := range toolPlans {
		result := s.toolService.Execute(userID, plan)
		toolResults = append(toolResults, result)

		toolMessage := &model.AiMessage{
			SessionID: session.ID,
			Role:      assistantRoleTool,
			Content:   nonEmptyJSON(result.OutputJSON),
			RequestID: requestID,
			CreatedAt: time.Now(),
		}
		if err := s.appendAiMessage(toolMessage); err != nil {
			log.Error("写入工具消息失败: %v, session_id=%d user_id=%d tool=%s", err, session.ID, userID, result.ToolName)
		}

		var output *string
		if strings.TrimSpace(result.OutputJSON) != "" {
			value := result.OutputJSON
			output = &value
		}
		toolCall := &model.AiToolCall{
			SessionID:  session.ID,
			ToolName:   result.ToolName,
			ToolInput:  nonEmptyJSON(result.InputJSON),
			ToolOutput: output,
			Success:    boolToInt32(result.Success),
			ErrorCode:  truncateText(result.ErrorCode, 64),
			ErrorMsg:   truncateText(result.ErrorMsg, 255),
			LatencyMs:  result.LatencyMS,
		}
		if err := s.repo.CreateAiToolCall(s.repo.DB, toolCall); err != nil {
			log.Error("写入工具调用日志失败: %v, session_id=%d tool=%s", err, session.ID, result.ToolName)
		} else {
			result.LogID = toolCall.ID
			toolLogIDs = append(toolLogIDs, toolCall.ID)
		}
	}

	// 上下文窗口默认 20，可通过配置缩放，控制 token 成本与回复稳定性。
	contextLimit := 20
	if cfg != nil && cfg.MaxContextMessages > 0 {
		contextLimit = cfg.MaxContextMessages
	}

	history, err := s.repo.ListRecentAiMessagesBySession(s.repo.DB, session.ID, contextLimit)
	if err != nil {
		log.Error("查询会话上下文失败: %v, session_id=%d", err, session.ID)
		return nil, err
	}

	promptMessages := s.buildPromptMessages(session.Scene, history)
	modelStart := time.Now()
	chatResp, aiErr := s.aiClient.Chat(s.ctx, &ai.ChatRequest{
		Model:    s.getChatModel(),
		Messages: promptMessages,
	})
	latencyMS := int32(time.Since(modelStart).Milliseconds())

	var (
		reply        string
		modelName    string
		finishReason string
		tokenIn      int32
		tokenOut     int32
	)
	if aiErr != nil {
		// 模型不可用时降级为工具结果总结，尽量保证主链路可用。
		reply = s.buildFallbackReply(message, toolResults, aiErr)
		modelName = "fallback"
		finishReason = "fallback"
		log.Warn("AI 调用失败，使用降级回复: %v, session_id=%d user_id=%d", aiErr, session.ID, userID)
	} else {
		reply = strings.TrimSpace(chatResp.Content)
		modelName = strings.TrimSpace(chatResp.Model)
		finishReason = strings.TrimSpace(chatResp.FinishReason)
		tokenIn = chatResp.Usage.PromptTokens
		tokenOut = chatResp.Usage.CompletionTokens
	}
	if reply == "" {
		reply = "我已收到你的问题，但当前无法生成有效回复，请稍后重试。"
	}
	if modelName == "" {
		modelName = s.getChatModel()
	}

	assistantMsg := &model.AiMessage{
		SessionID:    session.ID,
		Role:         assistantRoleAssistant,
		Content:      reply,
		Model:        modelName,
		FinishReason: mapFinishReason(finishReason),
		TokenIn:      tokenIn,
		TokenOut:     tokenOut,
		LatencyMs:    latencyMS,
		RequestID:    requestID,
		CreatedAt:    time.Now(),
	}
	if err := s.appendAiMessage(assistantMsg); err != nil {
		log.Error("写入助手消息失败: %v, session_id=%d user_id=%d", err, session.ID, userID)
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
	if err := s.repo.UpdateAiSessionAfterMessage(s.repo.DB, session.ID, now, title); err != nil {
		log.Error("更新会话时间失败: %v, session_id=%d", err, session.ID)
	}

	// 仅保留每日用量聚合，便于观测与成本估算（当前不做日配额拦截）。
	estimatedCost := s.estimateCost(modelName, tokenIn, tokenOut)
	if err := s.repo.UpsertAiUsageDaily(s.repo.DB, now, userID, aiErr == nil, tokenIn, tokenOut, estimatedCost); err != nil {
		log.Error("更新 AI 用量失败: %v, user_id=%d", err, userID)
	}

	return &api.AssistantChatResponse{
		Reply:     reply,
		ToolCalls: toAPIToolCalls(toolResults),
		Usage: &api.AssistantUsage{
			Model:     modelName,
			TokenIn:   tokenIn,
			TokenOut:  tokenOut,
			LatencyMs: latencyMS,
		},
	}, nil
}

// GetSessionMessages 获取会话历史消息
func (s *AssistantService) GetSessionMessages(req *api.AssistantSessionMessagesRequest) (*api.AssistantSessionMessagesResponse, error) {
	if req == nil || req.Id <= 0 {
		return nil, errors.New("会话ID不能为空")
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	_, err = s.repo.GetAiSessionByIDAndUser(s.repo.DB, req.Id, userID)
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

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	sessionID := req.SessionId
	if sessionID <= 0 {
		session, err := s.createSessionForUser(userID, assistantSceneActivityDraft, s.generateSessionTitle(req.Topic, assistantSceneActivityDraft))
		if err != nil {
			return nil, err
		}
		sessionID = session.ID
	} else {
		if _, err := s.repo.GetAiSessionByIDAndUser(s.repo.DB, sessionID, userID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("会话不存在")
			}
			return nil, err
		}
	}

	message := fmt.Sprintf("请生成活动草案，主题：%s。目标人群：%s。活动地点：%s。", strings.TrimSpace(req.Topic), strings.TrimSpace(req.TargetPeople), strings.TrimSpace(req.Location))
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
		Title:         truncateText(strings.TrimSpace(title), 128),
		Status:        assistantSessionStatusActive,
		Summary:       "",
		LastMessageAt: nil,
	}
	if session.Title == "" {
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
		seqNo, err := s.repo.GetNextAiMessageSeqNo(s.repo.DB, message.SessionID)
		if err != nil {
			return err
		}
		message.SeqNo = seqNo
		if err := s.repo.CreateAiMessage(s.repo.DB, message); err != nil {
			if isDuplicateSeqError(err) {
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
			messages = append(messages, ai.Message{Role: "system", Content: "工具结果(JSON):\n" + content})
		}
	}

	return messages
}

func (s *AssistantService) buildSystemPrompt(scene string) string {
	base := "你是环保志愿者平台的 AI 助手。回答要求准确、简洁、可执行。禁止编造不存在的数据。涉及权限或隐私时必须拒绝并解释原因。"
	switch scene {
	case assistantSceneActivityDraft:
		return base + "当前场景是活动草案生成，请优先给出活动标题、简介、执行流程和风险提示。"
	case assistantSceneOpsAdvisor:
		return base + "当前场景是组织运营分析，请优先基于活动统计结果给出可落地建议。"
	default:
		return base + "当前场景是通用问答。"
	}
}

func (s *AssistantService) buildFallbackReply(userMessage string, toolResults []*assistantToolResult, cause error) string {
	if len(toolResults) == 0 {
		return "AI 服务暂时不可用，请稍后重试。"
	}

	lines := make([]string, 0, len(toolResults)+2)
	lines = append(lines, "AI 服务暂时不可用，我先基于系统工具结果给你结论：")
	for _, item := range toolResults {
		if item == nil {
			continue
		}
		if item.Success {
			lines = append(lines, fmt.Sprintf("- %s: %s", item.ToolName, truncateText(item.OutputJSON, 280)))
		} else {
			lines = append(lines, fmt.Sprintf("- %s 调用失败: %s", item.ToolName, item.ErrorMsg))
		}
	}
	if strings.TrimSpace(userMessage) != "" {
		lines = append(lines, "你可以稍后重试，或缩小问题范围（例如指定活动关键词/组织ID）以提高成功率。")
	}
	if cause != nil {
		lines = append(lines, "失败原因: "+truncateText(cause.Error(), 120))
	}
	return strings.Join(lines, "\n")
}

func (s *AssistantService) resolveRequestID() string {
	if s.c != nil {
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

func (s *AssistantService) estimateCost(modelName string, tokenIn, tokenOut int32) float64 {
	if tokenIn <= 0 && tokenOut <= 0 {
		return 0
	}

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

func truncateText(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
