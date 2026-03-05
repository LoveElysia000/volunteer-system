package service

func (s *AssistantService) runRuntime(in *runtimeChatInput) (*runtimeChatOutput, error) {
	return s.runEinoRuntime(in)
}

func (s *AssistantService) buildRuntimeFallbackOutput(partial *runtimeChatOutput, cause error) *runtimeChatOutput {
	toolCalls := make([]runtimeToolCall, 0)
	if partial != nil {
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
