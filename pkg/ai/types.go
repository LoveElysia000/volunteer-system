package ai

// types.go 定义 AI 客户端与上层服务之间的通用数据结构。
//
// 该文件的定位：
// 1. 抽象对话消息结构（Message），统一角色、内容与工具调用关联字段。
// 2. 约定请求输入（ChatRequest）与响应输出（ChatResponse）的最小字段集合。
// 3. 对 token 用量做独立建模（Usage），便于统计、计费和观测。
// 4. 与具体 Provider 的 wire 协议解耦，保证 service 层仅依赖稳定的内部类型。

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
