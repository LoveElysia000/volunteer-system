package service

import (
	"testing"
	"volunteer-system/internal/api"
)

func TestAssistantChatStream_EmitsDeltaAndDoneEvents(t *testing.T) {
	events := BuildAssistantStreamEvents(101, &api.AssistantChatResponse{
		Reply: "hello stream",
		Usage: &api.AssistantUsage{Model: "x", TokenIn: 1, TokenOut: 2, LatencyMs: 3},
	})
	if len(events) < 4 {
		t.Fatalf("expected at least 4 events, got %d", len(events))
	}
	if events[0].Event != "start" {
		t.Fatalf("expected first event start, got %s", events[0].Event)
	}
	last := events[len(events)-1]
	if last.Event != "done" {
		t.Fatalf("expected last event done, got %s", last.Event)
	}

	hasDelta := false
	hasUsage := false
	for _, event := range events {
		if event.Event == "delta" {
			hasDelta = true
		}
		if event.Event == "usage" {
			hasUsage = true
		}
	}
	if !hasDelta || !hasUsage {
		t.Fatalf("expected delta and usage events, got %#v", events)
	}
}
