package service

import (
	"errors"
	"strings"
	"testing"
)

func TestBuildRuntimeFallbackOutputWithToolCalls(t *testing.T) {
	svc := &AssistantService{}
	partial := &runtimeChatOutput{
		ToolCalls: []runtimeToolCall{
			{
				ToolName:   assistantToolActivitySearch,
				OutputJSON: `{"count":1}`,
				Success:    true,
			},
		},
	}

	out := svc.buildRuntimeFallbackOutput(partial, errors.New("status=429"))
	if out == nil {
		t.Fatal("fallback output should not be nil")
	}
	if out.Model != "fallback" || out.FinishReason != "fallback" {
		t.Fatalf("unexpected fallback metadata: model=%s reason=%s", out.Model, out.FinishReason)
	}
	if len(out.ToolCalls) != 1 {
		t.Fatalf("unexpected tool calls count: %d", len(out.ToolCalls))
	}
	if !strings.Contains(out.Reply, assistantToolActivitySearch) {
		t.Fatalf("fallback reply should include tool name, got: %s", out.Reply)
	}
	if !strings.Contains(out.Reply, "模型服务限流") {
		t.Fatalf("fallback reply should include user-visible reason, got: %s", out.Reply)
	}
}

func TestBuildRuntimeFallbackOutputWithoutToolCalls(t *testing.T) {
	svc := &AssistantService{}
	out := svc.buildRuntimeFallbackOutput(nil, errors.New("unknown"))
	if out == nil {
		t.Fatal("fallback output should not be nil")
	}
	if len(out.ToolCalls) != 0 {
		t.Fatalf("unexpected tool calls count: %d", len(out.ToolCalls))
	}
	if !strings.Contains(out.Reply, "AI 服务暂时不可用") {
		t.Fatalf("fallback reply mismatch: %s", out.Reply)
	}
}
