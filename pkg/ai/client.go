package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
	"volunteer-system/config"
)

// client.go 提供 AI 对话客户端实现，面向 OpenAI Chat Completions 兼容协议。
//
// 该文件的核心职责：
// 1. 统一多 Provider 调用入口（deepseek / openai / compatible），并归一化 provider 名称。
// 2. 解析请求所需参数：API Key、BaseURL、Model，支持“请求级 > 配置级 > 默认值”的回退顺序。
// 3. 支持 API Key 的多来源解析：显式配置、${ENV_VAR:default} 表达式、环境变量候选列表。
// 4. 负责 HTTP 调用与错误处理：超时控制、状态码判定、可重试错误识别、指数退避重试。
// 5. 解析并归一化响应结构，兼容不同 provider 的 content 形态，输出统一 ChatResponse。
// 6. 在错误信息中对响应体做截断，避免超长内容影响日志可读性。

var (
	// ErrAIUnavailable 表示 AI 功能不可用
	ErrAIUnavailable = errors.New("AI 助手未启用")
	// ErrAPIKeyMissing 表示 API Key 缺失
	ErrAPIKeyMissing = errors.New("AI API Key 未配置")
)

// Client 兼容 OpenAI 协议的多 Provider 客户端
type Client struct {
	cfg        *config.AIConfig
	httpClient *http.Client
}

// NewClient 创建 AI 客户端
func NewClient(cfg *config.AIConfig) *Client {
	timeout := 15 * time.Second
	if cfg != nil && cfg.RequestTimeoutMS > 0 {
		timeout = time.Duration(cfg.RequestTimeoutMS) * time.Millisecond
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Chat 按 provider 调用 chat completions 接口（deepseek/openai/compatible）
func (c *Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 入口兜底：AI 未启用或请求消息为空时直接失败，避免无意义下游调用。
	if c.cfg == nil || !c.cfg.Enabled {
		return nil, ErrAIUnavailable
	}
	if req == nil || len(req.Messages) == 0 {
		return nil, errors.New("AI 请求参数为空")
	}

	// 先解析 provider，再据此确定 key/baseURL/model 的默认值和回退顺序。
	provider := normalizeProvider(c.cfg.Provider)

	apiKey := resolveAPIKey(c.cfg.APIKey, provider)
	if apiKey == "" {
		return nil, ErrAPIKeyMissing
	}

	baseURL := resolveBaseURL(c.cfg.BaseURL, provider)

	model := resolveModel(req.Model, c.cfg.ChatModel, provider)

	// 构造与 OpenAI Chat Completions 兼容的最小请求体。
	payload := map[string]any{
		"model":    model,
		"messages": req.Messages,
	}
	// temperature=0 在多数 provider 中等价于未传，这里仅在显式 >0 时附带。
	if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
	maxRetries := normalizeMaxRetries(c.cfg)
	attempts := maxRetries + 1

	var lastErr error
	// 仅对可重试错误（网络异常、429、5xx）执行退避重试。
	for attempt := 0; attempt < attempts; attempt++ {
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)

		httpResp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = err
			// 网络层错误仅在可重试且仍有剩余次数时退避后重试。
			if attempt < maxRetries && shouldRetryTransportError(err) {
				if backoffErr := waitRetryBackoff(ctx, attempt); backoffErr != nil {
					return nil, backoffErr
				}
				continue
			}
			return nil, err
		}

		respBody, readErr := io.ReadAll(httpResp.Body)
		_ = httpResp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			// 读取失败通常是瞬时网络问题，保守按重试策略处理。
			if attempt < maxRetries {
				if backoffErr := waitRetryBackoff(ctx, attempt); backoffErr != nil {
					return nil, backoffErr
				}
				continue
			}
			return nil, readErr
		}
		if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
			lastErr = fmt.Errorf("AI 接口调用失败: status=%d body=%s", httpResp.StatusCode, truncate(string(respBody), 200))
			// 仅对限流/5xx 进行重试，避免对确定性客户端错误反复请求。
			if attempt < maxRetries && shouldRetryStatusCode(httpResp.StatusCode) {
				if backoffErr := waitRetryBackoff(ctx, attempt); backoffErr != nil {
					return nil, backoffErr
				}
				continue
			}
			return nil, lastErr
		}

		var wire wireChatResponse
		if err := json.Unmarshal(respBody, &wire); err != nil {
			lastErr = err
			return nil, err
		}
		// 协议上 choices 应至少包含一个候选回复。
		if len(wire.Choices) == 0 {
			lastErr = errors.New("AI 接口返回空结果")
			return nil, lastErr
		}

		content := extractContent(wire.Choices[0].Message.Content)
		return &ChatResponse{
			Model:        wire.Model,
			Content:      content,
			FinishReason: wire.Choices[0].FinishReason,
			Usage: Usage{
				PromptTokens:     wire.Usage.PromptTokens,
				CompletionTokens: wire.Usage.CompletionTokens,
			},
		}, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("AI 调用失败")
}

// wireChatResponse 是与上游 provider 对齐的响应解码结构。
type wireChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content any `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int32 `json:"prompt_tokens"`
		CompletionTokens int32 `json:"completion_tokens"`
	} `json:"usage"`
}

func resolveAPIKey(raw, provider string) string {
	// 优先使用显式配置（支持 ${ENV:default}），其次按 provider 约定环境变量回退。
	v := strings.TrimSpace(raw)
	if v != "" {
		if resolved, ok := expandEnvExpression(v); ok {
			if resolved != "" {
				return resolved
			}
		} else {
			return v
		}
	}

	for _, envName := range apiKeyEnvCandidates(provider) {
		if env := strings.TrimSpace(os.Getenv(envName)); env != "" {
			return env
		}
	}
	return ""
}

func extractContent(v any) string {
	// 兼容不同 provider 的 content 结构：string 或多段内容数组。
	switch val := v.(type) {
	case string:
		return val
	case []any:
		parts := make([]string, 0, len(val))
		for _, item := range val {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := m["text"]; ok {
				parts = append(parts, fmt.Sprintf("%v", text))
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%v", v)
	}
}

// truncate 仅用于日志/错误展示，避免把超长响应体直接拼接到错误信息中。
func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

// normalizeMaxRetries 将非法重试次数归零，统一由调用方按 "重试+首调" 计算总尝试次数。
func normalizeMaxRetries(cfg *config.AIConfig) int {
	if cfg == nil || cfg.MaxRetries < 0 {
		return 0
	}
	return cfg.MaxRetries
}

// normalizeProvider 归一化 provider 名称；未知值按 compatible 处理。
func normalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "deepseek":
		return "deepseek"
	case "openai":
		return "openai"
	case "compatible":
		return "compatible"
	default:
		return "compatible"
	}
}

// resolveBaseURL 支持显式配置优先，未配置时按 provider 选择默认网关。
func resolveBaseURL(raw, provider string) string {
	baseURL := strings.TrimSpace(raw)
	if baseURL != "" {
		return baseURL
	}

	switch provider {
	case "deepseek":
		return "https://api.deepseek.com/v1"
	case "openai":
		return "https://api.openai.com/v1"
	default:
		return "https://api.openai.com/v1"
	}
}

// resolveModel 支持请求级覆盖，其次走配置，最后按 provider 选默认模型。
func resolveModel(requestModel, configModel, provider string) string {
	model := strings.TrimSpace(requestModel)
	if model != "" {
		return model
	}
	model = strings.TrimSpace(configModel)
	if model != "" {
		return model
	}

	switch provider {
	case "deepseek":
		return "deepseek-chat"
	case "openai":
		return "gpt-4o-mini"
	default:
		return "gpt-4o-mini"
	}
}

// expandEnvExpression 解析 ${ENV_VAR:default} 形式配置。
func expandEnvExpression(v string) (string, bool) {
	// 支持 ${ENV_VAR:default} 形式
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

// apiKeyEnvCandidates 返回不同 provider 的环境变量优先级。
func apiKeyEnvCandidates(provider string) []string {
	switch provider {
	case "deepseek":
		return []string{"DEEPSEEK_API_KEY", "AI_API_KEY", "VOLUNTEER_AI_API_KEY"}
	case "openai":
		return []string{"OPENAI_API_KEY", "AI_API_KEY", "VOLUNTEER_AI_API_KEY"}
	default:
		return []string{"AI_API_KEY", "VOLUNTEER_AI_API_KEY", "OPENAI_API_KEY", "DEEPSEEK_API_KEY"}
	}
}

// shouldRetryStatusCode 定义可重试状态码策略（限流/服务端错误）。
func shouldRetryStatusCode(statusCode int) bool {
	if statusCode == http.StatusTooManyRequests {
		return true
	}
	return statusCode >= 500 && statusCode <= 599
}

// shouldRetryTransportError 对网络层异常进行保守重试，context cancel 不重试。
func shouldRetryTransportError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return true
}

// waitRetryBackoff 指数退避（200ms 起步，上限 2s），并响应调用方 context 取消。
func waitRetryBackoff(ctx context.Context, attempt int) error {
	delay := 200 * time.Millisecond
	for i := 0; i < attempt; i++ {
		delay *= 2
		if delay > 2*time.Second {
			delay = 2 * time.Second
			break
		}
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
