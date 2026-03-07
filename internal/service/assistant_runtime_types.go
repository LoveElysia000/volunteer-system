package service

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
