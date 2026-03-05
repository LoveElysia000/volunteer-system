package service

// runtimeChatInput describes the data required by runtime executors.
type runtimeChatInput struct {
	UserID    int64
	SessionID int64
	Scene     string
	Message   string
	History   []*aiMessageSnapshot
	RequestID string
}

// aiMessageSnapshot is a lightweight history message used by runtime executors.
type aiMessageSnapshot struct {
	Role    int32
	Content string
}

// runtimeToolCall is the normalized tool call result returned by a runtime.
type runtimeToolCall struct {
	ToolName   string
	InputJSON  string
	OutputJSON string
	Success    bool
	ErrorCode  string
	ErrorMsg   string
	LatencyMS  int32
}

// runtimeChatOutput is the normalized output produced by a runtime.
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

func runtimeToolCallsToAssistantResults(calls []runtimeToolCall) []*assistantToolResult {
	results := make([]*assistantToolResult, 0, len(calls))
	for _, call := range calls {
		c := call
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
