package service

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
