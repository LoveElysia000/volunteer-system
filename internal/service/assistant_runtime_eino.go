package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"volunteer-system/config"
	"volunteer-system/pkg/ai"
	"volunteer-system/pkg/util"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// assistant_runtime_eino.go 负责 Eino 运行时接入与配置映射。
//
// 设计边界：
// 1. 仅处理“模型 + 工具 + checkpoint + 重试”编排，不处理业务持久化。
// 2. 输入/输出统一走 runtimeChatInput/runtimeChatOutput，便于上层替换运行时实现。
// 3. 配置项在此文件集中收敛，避免业务层散落 provider 判断。

// runtimeChatInput 是运行时执行器的统一输入。
type runtimeChatInput struct {
	UserID    int64
	SessionID int64
	Scene     string
	Message   string
	History   []*aiMessageSnapshot
	RequestID string
}

// aiMessageSnapshot 是用于运行时的轻量历史消息快照。
type aiMessageSnapshot struct {
	Role    int32
	Content string
}

// runtimeToolCall 表示运行时返回的标准化工具调用结果。
type runtimeToolCall struct {
	ToolName   string
	InputJSON  string
	OutputJSON string
	Success    bool
	ErrorCode  string
	ErrorMsg   string
	LatencyMS  int32
}

// runtimeChatOutput 是运行时输出的统一结构。
// 业务层只依赖该结构，不感知底层框架细节（Eino/其他）。
type runtimeChatOutput struct {
	Reply        string
	Model        string
	FinishReason string
	TokenIn      int32
	TokenOut     int32
	LatencyMS    int32
	Success      bool
	ToolCalls    []runtimeToolCall
}

// runtimeToolCallsToAssistantResults 将 runtime 工具结果转换为业务层结果结构。
func runtimeToolCallsToAssistantResults(calls []runtimeToolCall) []*assistantToolResult {
	results := make([]*assistantToolResult, 0, len(calls))
	for _, call := range calls {
		c := call
		// 做值拷贝，避免循环变量地址复用导致结果串值。
		results = append(results, &assistantToolResult{
			ToolName:   c.ToolName,
			InputJSON:  nonEmptyJSON(c.InputJSON),
			OutputJSON: nonEmptyJSON(c.OutputJSON),
			Success:    c.Success,
			ErrorCode:  c.ErrorCode,
			ErrorMsg:   c.ErrorMsg,
			LatencyMS:  c.LatencyMS,
		})
	}
	return results
}

// runRuntime 作为运行时路由入口。
// 当前仅接入 Eino，后续若引入其他 runtime，可在此统一分流。
func (s *AssistantService) runRuntime(in *runtimeChatInput) (*runtimeChatOutput, error) {
	return s.runEinoRuntime(in)
}

// buildRuntimeFallbackOutput 在运行时失败时构建降级输出。
// 若部分工具已执行完成，会保留其结果用于用户可见回复。
func (s *AssistantService) buildRuntimeFallbackOutput(partial *runtimeChatOutput, cause error) *runtimeChatOutput {
	toolCalls := make([]runtimeToolCall, 0)
	if partial != nil {
		// 部分成功场景会携带已执行工具结果，用于拼接降级回复。
		toolCalls = partial.ToolCalls
	}

	reply := s.buildFallbackReply(runtimeToolCallsToAssistantResults(toolCalls), cause)
	return &runtimeChatOutput{
		Reply:        reply,
		Model:        "fallback",
		FinishReason: "fallback",
		TokenIn:      0,
		TokenOut:     0,
		LatencyMS:    0,
		Success:      false,
		ToolCalls:    toolCalls,
	}
}

// runEinoRuntime 做运行前兜底校验（配置与密钥），并进入单次执行流程。
func (s *AssistantService) runEinoRuntime(in *runtimeChatInput) (*runtimeChatOutput, error) {
	if in == nil {
		return nil, errors.New("runtime input cannot be nil")
	}

	// 读取 AI 开关和 provider 配置，未启用直接返回统一错误。
	cfg := s.getAIConfig()
	if cfg == nil || !cfg.Enabled {
		return nil, ai.ErrAIUnavailable
	}

	// API Key 统一从配置/环境变量解析，避免调用层自行判断来源。
	apiKey := resolveEinoAPIKey(cfg)
	if strings.TrimSpace(apiKey) == "" {
		return nil, ai.ErrAPIKeyMissing
	}

	return s.runEinoRuntimeOnce(in, cfg, apiKey)
}

// runEinoRuntimeOnce 负责一次完整 Agent 执行：
// - 构建模型与工具；
// - 拉起 Runner 并消费事件流；
// - 汇总 assistant 文本、finish reason、token 用量；
// - 返回统一 runtime 输出。
func (s *AssistantService) runEinoRuntimeOnce(in *runtimeChatInput, cfg *config.AIConfig, apiKey string) (*runtimeChatOutput, error) {
	// 先组装模型连接参数。
	modelName := resolveEinoModel(cfg)
	baseURL := resolveEinoBaseURL(cfg)
	chatModel, err := einoopenai.NewChatModel(s.ctx, &einoopenai.ChatModelConfig{
		APIKey:  apiKey,
		Model:   modelName,
		BaseURL: baseURL,
		Timeout: resolveEinoModelTimeout(cfg),
	})
	if err != nil {
		return nil, err
	}

	// 绑定工具并创建采集器，后续无论成功失败都可回传工具执行痕迹。
	collector := &einoToolCollector{}
	tools, err := s.buildEinoTools(in.UserID, collector)
	if err != nil {
		return nil, err
	}

	// Agent 聚合了系统提示、工具集合和模型重试策略。
	agent, err := adk.NewChatModelAgent(s.ctx, &adk.ChatModelAgentConfig{
		Name:             "volunteer_assistant",
		Description:      "volunteer system AI assistant",
		Instruction:      s.buildSystemPrompt(in.Scene),
		Model:            chatModel,
		ToolsConfig:      adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{Tools: tools}},
		MaxIterations:    resolveEinoMaxSteps(cfg),
		ModelRetryConfig: resolveEinoModelRetryConfig(cfg),
	})
	if err != nil {
		return nil, err
	}

	// Runner 承担事件流执行与 checkpoint 读写。
	runner := adk.NewRunner(s.ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: resolveEinoEnableStream(cfg),
		CheckPointStore: s.newEinoCheckpointStore(cfg),
	})

	// checkpointID 决定会话上下文是否可复用；history 转换为模型输入消息。
	checkpointID := buildEinoCheckpointID(in.SessionID, in.RequestID)
	inputMessages := buildEinoInputMessages(in.History)
	if len(inputMessages) == 0 {
		// 历史为空时至少注入本轮用户问题，避免空请求。
		inputMessages = []*schema.Message{schema.UserMessage(in.Message)}
	}

	startedAt := time.Now()
	iter := runner.Run(s.ctx, inputMessages, adk.WithCheckPointID(checkpointID))

	// 在事件流中逐步覆盖最终输出，始终保留最后一次 assistant 内容与 usage。
	reply := ""
	finishReason := ""
	tokenIn := int32(0)
	tokenOut := int32(0)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event == nil {
			continue
		}
		if event.Err != nil {
			if isEinoRetryNotice(event.Err) {
				// Model retry notice event is non-fatal and should not break the run loop.
				continue
			}
			// 运行失败时保留已收集的工具调用结果，供上层 fallback 使用。
			return buildEinoRuntimeOutput(modelName, reply, finishReason, tokenIn, tokenOut, startedAt, false, collector), event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		msg, _, msgErr := adk.GetMessage(event)
		if msgErr != nil {
			return buildEinoRuntimeOutput(modelName, reply, finishReason, tokenIn, tokenOut, startedAt, false, collector), msgErr
		}
		if msg == nil {
			continue
		}
		if event.Output.MessageOutput.Role != schema.Assistant {
			// 非 assistant 角色事件不作为最终回复内容。
			continue
		}

		if content := strings.TrimSpace(msg.Content); content != "" {
			reply = content
		}
		if msg.ResponseMeta != nil {
			if fr := strings.TrimSpace(msg.ResponseMeta.FinishReason); fr != "" {
				finishReason = fr
			}
			if msg.ResponseMeta.Usage != nil {
				// usage 采用最后一次有效 assistant 事件，保证与最终回复一致。
				tokenIn = util.ClampInt32(msg.ResponseMeta.Usage.PromptTokens)
				tokenOut = util.ClampInt32(msg.ResponseMeta.Usage.CompletionTokens)
			}
		}
	}

	if strings.TrimSpace(reply) == "" {
		// Eino 正常返回但未给出 assistant 内容时视作失败，交给上层 fallback。
		return buildEinoRuntimeOutput(modelName, "", finishReason, tokenIn, tokenOut, startedAt, false, collector), errors.New("empty assistant reply from eino runtime")
	}

	if finishReason == "" {
		// 供应商未返回 finish reason 时按 stop 兜底，便于上层枚举映射。
		finishReason = "stop"
	}

	return buildEinoRuntimeOutput(modelName, reply, finishReason, tokenIn, tokenOut, startedAt, true, collector), nil
}

// buildEinoRuntimeOutput 把事件流中累计状态收敛为统一输出结构。
func buildEinoRuntimeOutput(modelName, reply, finishReason string, tokenIn, tokenOut int32, startedAt time.Time, success bool, collector *einoToolCollector) *runtimeChatOutput {
	// 统一在此处填充延迟与工具快照，避免多个返回路径字段不一致。
	return &runtimeChatOutput{
		Reply:        reply,
		Model:        modelName,
		FinishReason: finishReason,
		TokenIn:      tokenIn,
		TokenOut:     tokenOut,
		LatencyMS:    int32(time.Since(startedAt).Milliseconds()),
		Success:      success,
		ToolCalls:    collector.Snapshot(),
	}
}

// isEinoRetryNotice 用于识别 Eino 的“将要重试”提示事件。
// 该事件不是终态错误，不应中断整个运行循环。
func isEinoRetryNotice(err error) bool {
	if err == nil {
		return false
	}
	var retryNotice *adk.WillRetryError
	return errors.As(err, &retryNotice)
}

// resolveEinoMaxSteps 限制 Agent 最大迭代次数，避免异常场景下无界循环。
func resolveEinoMaxSteps(cfg *config.AIConfig) int {
	if cfg == nil || cfg.Eino.MaxSteps <= 0 {
		return 4
	}
	if cfg.Eino.MaxSteps > 32 {
		return 32
	}
	return cfg.Eino.MaxSteps
}

// resolveEinoMaxRetries 映射全局重试配置到 Eino 模型重试次数。
func resolveEinoMaxRetries(cfg *config.AIConfig) int {
	if cfg == nil || cfg.MaxRetries <= 0 {
		return 0
	}
	if cfg.MaxRetries > 5 {
		return 5
	}
	return cfg.MaxRetries
}

// resolveEinoModelTimeout 解析模型请求超时。
func resolveEinoModelTimeout(cfg *config.AIConfig) time.Duration {
	if cfg == nil || cfg.RequestTimeoutMS <= 0 {
		return 15 * time.Second
	}
	return time.Duration(cfg.RequestTimeoutMS) * time.Millisecond
}

// resolveEinoEnableStream 控制 Runner 是否启用事件流。
// 说明：当前 API 仍是非流式返回，此开关仅影响运行时内部事件消费方式。
func resolveEinoEnableStream(cfg *config.AIConfig) bool {
	if cfg == nil {
		return false
	}
	return cfg.Eino.EnableStream
}

// resolveEinoModelRetryConfig 把项目级重试策略注入 Eino Agent。
func resolveEinoModelRetryConfig(cfg *config.AIConfig) *adk.ModelRetryConfig {
	maxRetries := resolveEinoMaxRetries(cfg)
	if maxRetries <= 0 {
		return nil
	}
	return &adk.ModelRetryConfig{
		MaxRetries: maxRetries,
		IsRetryAble: func(_ context.Context, err error) bool {
			// 复用统一错误分类函数，便于不同 provider 共用重试准则。
			return shouldRetryEinoRuntimeError(err)
		},
		BackoffFunc: func(_ context.Context, attempt int) time.Duration {
			return resolveEinoRetryBackoff(attempt)
		},
	}
}

// resolveEinoModel 根据配置/Provider 选择默认模型。
func resolveEinoModel(cfg *config.AIConfig) string {
	if cfg != nil && strings.TrimSpace(cfg.ChatModel) != "" {
		return strings.TrimSpace(cfg.ChatModel)
	}

	switch normalizeAIProvider(cfg) {
	case "deepseek":
		return "deepseek-chat"
	case "openai":
		return "gpt-4o-mini"
	default:
		return "gpt-4o-mini"
	}
}

// resolveEinoBaseURL 根据 Provider 选择默认网关地址。
func resolveEinoBaseURL(cfg *config.AIConfig) string {
	if cfg != nil && strings.TrimSpace(cfg.BaseURL) != "" {
		return strings.TrimSpace(cfg.BaseURL)
	}

	switch normalizeAIProvider(cfg) {
	case "deepseek":
		return "https://api.deepseek.com/v1"
	case "openai":
		return "https://api.openai.com/v1"
	default:
		return "https://api.openai.com/v1"
	}
}

// resolveEinoAPIKey 支持两种来源：
// 1) 配置文件直接值或 ${ENV:default}；
// 2) 按 provider 顺序回退的环境变量。
func resolveEinoAPIKey(cfg *config.AIConfig) string {
	if cfg == nil {
		return ""
	}

	v := strings.TrimSpace(cfg.APIKey)
	if v != "" {
		if resolved, ok := expandConfigEnvExpression(v); ok {
			if strings.TrimSpace(resolved) != "" {
				// 支持 ${ENV:default} 表达式。
				return strings.TrimSpace(resolved)
			}
		} else {
			return v
		}
	}

	// 配置为空时按 provider 约定顺序回退环境变量。
	for _, envName := range aiKeyEnvCandidates(normalizeAIProvider(cfg)) {
		if env := strings.TrimSpace(os.Getenv(envName)); env != "" {
			return env
		}
	}
	return ""
}

// shouldRetryEinoRuntimeError 统一模型错误重试判定：
// - 不重试：鉴权/参数/主动取消；
// - 可重试：超时、限流、5xx、典型网络瞬断。
func shouldRetryEinoRuntimeError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, ai.ErrAIUnavailable) || errors.Is(err, ai.ErrAPIKeyMissing) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	lower := strings.ToLower(strings.TrimSpace(err.Error()))
	// 已知业务类错误（权限/参数）快速失败，减少无效重试成本。
	if strings.HasPrefix(lower, "permission_denied:") || strings.HasPrefix(lower, "invalid_params:") {
		return false
	}
	if strings.Contains(lower, "status=429") ||
		strings.Contains(lower, "too many requests") ||
		strings.Contains(lower, "rate limit") {
		return true
	}
	if strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "deadline exceeded") ||
		strings.Contains(lower, "temporarily unavailable") ||
		strings.Contains(lower, "connection reset") ||
		strings.Contains(lower, "i/o timeout") ||
		strings.Contains(lower, "dial tcp") ||
		strings.Contains(lower, "tls handshake timeout") {
		return true
	}
	return strings.Contains(lower, "status=500") ||
		strings.Contains(lower, "status=502") ||
		strings.Contains(lower, "status=503") ||
		strings.Contains(lower, "status=504")
}

// resolveEinoRetryBackoff 使用指数退避并限制最大等待时间。
func resolveEinoRetryBackoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	delay := 200 * time.Millisecond
	for i := 0; i < attempt; i++ {
		delay *= 2
		if delay > 2*time.Second {
			// 上限封顶，避免重试等待过长拉高整体响应时间。
			delay = 2 * time.Second
			break
		}
	}
	return delay
}

func normalizeAIProvider(cfg *config.AIConfig) string {
	if cfg == nil {
		return "compatible"
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "deepseek":
		return "deepseek"
	case "openai":
		return "openai"
	default:
		return "compatible"
	}
}

func aiKeyEnvCandidates(provider string) []string {
	switch provider {
	case "deepseek":
		return []string{"DEEPSEEK_API_KEY", "AI_API_KEY", "VOLUNTEER_AI_API_KEY"}
	case "openai":
		return []string{"OPENAI_API_KEY", "AI_API_KEY", "VOLUNTEER_AI_API_KEY"}
	default:
		return []string{"AI_API_KEY", "VOLUNTEER_AI_API_KEY", "OPENAI_API_KEY", "DEEPSEEK_API_KEY"}
	}
}

func expandConfigEnvExpression(v string) (string, bool) {
	if !strings.HasPrefix(v, "${") || !strings.HasSuffix(v, "}") {
		return "", false
	}

	inner := strings.TrimSuffix(strings.TrimPrefix(v, "${"), "}")
	parts := strings.SplitN(inner, ":", 2)
	name := strings.TrimSpace(parts[0])
	fallback := ""
	if len(parts) == 2 {
		fallback = strings.TrimSpace(parts[1])
	}
	if name != "" {
		if env := strings.TrimSpace(os.Getenv(name)); env != "" {
			return env, true
		}
	}
	return fallback, true
}

// buildEinoCheckpointID 为会话级与无会话请求生成稳定 checkpoint key。
func buildEinoCheckpointID(sessionID int64, requestID string) string {
	rid := strings.TrimSpace(requestID)
	if rid == "" {
		rid = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if sessionID > 0 {
		return fmt.Sprintf("assistant:%d:%s", sessionID, rid)
	}
	return fmt.Sprintf("assistant:req:%s", rid)
}

// buildEinoInputMessages 把历史消息映射为 Eino schema 消息。
// 注意：工具消息统一降为 assistant 角色，避免工具输出污染 system 指令层。
func buildEinoInputMessages(history []*aiMessageSnapshot) []*schema.Message {
	messages := make([]*schema.Message, 0, len(history))
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
			messages = append(messages, schema.UserMessage(content))
		case assistantRoleAssistant:
			messages = append(messages, schema.AssistantMessage(content, nil))
		case assistantRoleTool:
			// Tool payload can contain user-generated text; keep it out of system role.
			messages = append(messages, schema.AssistantMessage("工具结果(JSON):\n"+content, nil))
		}
	}
	return messages
}
