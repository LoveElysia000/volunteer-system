package service

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"volunteer-system/config"
	"volunteer-system/pkg/ai"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func (s *AssistantService) runEinoRuntime(in *runtimeChatInput) (*runtimeChatOutput, error) {
	if in == nil {
		return nil, errors.New("runtime input cannot be nil")
	}

	cfg := s.getAIConfig()
	if cfg == nil || !cfg.Enabled {
		return nil, ai.ErrAIUnavailable
	}

	apiKey := resolveEinoAPIKey(cfg)
	if strings.TrimSpace(apiKey) == "" {
		return nil, ai.ErrAPIKeyMissing
	}

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

	collector := &einoToolCollector{}
	tools, err := s.buildEinoTools(in.UserID, collector)
	if err != nil {
		return nil, err
	}

	agent, err := adk.NewChatModelAgent(s.ctx, &adk.ChatModelAgentConfig{
		Name:          "volunteer_assistant",
		Description:   "volunteer system AI assistant",
		Instruction:   s.buildSystemPrompt(in.Scene),
		Model:         chatModel,
		ToolsConfig:   adk.ToolsConfig{ToolsNodeConfig: compose.ToolsNodeConfig{Tools: tools}},
		MaxIterations: resolveEinoMaxSteps(cfg),
	})
	if err != nil {
		return nil, err
	}

	runner := adk.NewRunner(s.ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: false,
		CheckPointStore: s.newEinoCheckpointStore(cfg),
	})

	checkpointID := buildEinoCheckpointID(in.SessionID, in.RequestID)
	inputMessages := buildEinoInputMessages(in.History)
	if len(inputMessages) == 0 {
		inputMessages = []*schema.Message{schema.UserMessage(in.Message)}
	}

	startedAt := time.Now()
	iter := runner.Run(s.ctx, inputMessages, adk.WithCheckPointID(checkpointID))

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
			return &runtimeChatOutput{
				Reply:        reply,
				Model:        modelName,
				FinishReason: finishReason,
				TokenIn:      tokenIn,
				TokenOut:     tokenOut,
				LatencyMS:    int32(time.Since(startedAt).Milliseconds()),
				Success:      false,
				ToolCalls:    collector.Snapshot(),
			}, event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		msg, _, msgErr := adk.GetMessage(event)
		if msgErr != nil {
			return &runtimeChatOutput{
				Reply:        reply,
				Model:        modelName,
				FinishReason: finishReason,
				TokenIn:      tokenIn,
				TokenOut:     tokenOut,
				LatencyMS:    int32(time.Since(startedAt).Milliseconds()),
				Success:      false,
				ToolCalls:    collector.Snapshot(),
			}, msgErr
		}
		if msg == nil {
			continue
		}
		if event.Output.MessageOutput.Role != schema.Assistant {
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
				tokenIn = clampInt32(msg.ResponseMeta.Usage.PromptTokens)
				tokenOut = clampInt32(msg.ResponseMeta.Usage.CompletionTokens)
			}
		}
	}

	if strings.TrimSpace(reply) == "" {
		return &runtimeChatOutput{
			Reply:        "",
			Model:        modelName,
			FinishReason: finishReason,
			TokenIn:      tokenIn,
			TokenOut:     tokenOut,
			LatencyMS:    int32(time.Since(startedAt).Milliseconds()),
			Success:      false,
			ToolCalls:    collector.Snapshot(),
		}, errors.New("empty assistant reply from eino runtime")
	}

	if finishReason == "" {
		finishReason = "stop"
	}

	return &runtimeChatOutput{
		Reply:        reply,
		Model:        modelName,
		FinishReason: finishReason,
		TokenIn:      tokenIn,
		TokenOut:     tokenOut,
		LatencyMS:    int32(time.Since(startedAt).Milliseconds()),
		Success:      true,
		ToolCalls:    collector.Snapshot(),
	}, nil
}

func resolveEinoMaxSteps(cfg *config.AIConfig) int {
	if cfg == nil || cfg.Eino.MaxSteps <= 0 {
		return 4
	}
	if cfg.Eino.MaxSteps > 32 {
		return 32
	}
	return cfg.Eino.MaxSteps
}

func resolveEinoModelTimeout(cfg *config.AIConfig) time.Duration {
	if cfg == nil || cfg.RequestTimeoutMS <= 0 {
		return 15 * time.Second
	}
	return time.Duration(cfg.RequestTimeoutMS) * time.Millisecond
}

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

func resolveEinoAPIKey(cfg *config.AIConfig) string {
	if cfg == nil {
		return ""
	}

	v := strings.TrimSpace(cfg.APIKey)
	if v != "" {
		if resolved, ok := expandConfigEnvExpression(v); ok {
			if strings.TrimSpace(resolved) != "" {
				return strings.TrimSpace(resolved)
			}
		} else {
			return v
		}
	}

	for _, envName := range aiKeyEnvCandidates(normalizeAIProvider(cfg)) {
		if env := strings.TrimSpace(os.Getenv(envName)); env != "" {
			return env
		}
	}
	return ""
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

func clampInt32(v int) int32 {
	if v <= 0 {
		return 0
	}
	if v > int(^uint32(0)>>1) {
		return int32(^uint32(0) >> 1)
	}
	return int32(v)
}

func buildEinoCheckpointID(sessionID int64, requestID string) string {
	if strings.TrimSpace(requestID) == "" {
		requestID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("assistant:%d:%s", sessionID, strings.TrimSpace(requestID))
}

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
			messages = append(messages, schema.SystemMessage("工具结果(JSON):\n"+content))
		}
	}
	return messages
}
