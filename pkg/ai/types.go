package ai

// Message 对话消息
type Message struct {
	Role       string `json:"role"`
	Content    string `json:"content,omitempty"`
	Name       string `json:"name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// ChatRequest 对话请求
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
}

// Usage token 用量
type Usage struct {
	PromptTokens     int32
	CompletionTokens int32
}

// ChatResponse 对话响应
type ChatResponse struct {
	Model        string
	Content      string
	FinishReason string
	Usage        Usage
}
