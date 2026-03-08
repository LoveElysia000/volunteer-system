package service

import (
	"volunteer-system/internal/api"
)

const (
	assistantStreamMaxDeltaChunkRunes = 80
)

type assistantStreamEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type AssistantStreamEvent struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

func BuildAssistantStreamEvents(sessionID int64, resp *api.AssistantChatResponse) []AssistantStreamEvent {
	internalEvents := buildAssistantStreamEvents(sessionID, resp)
	events := make([]AssistantStreamEvent, 0, len(internalEvents))
	for _, event := range internalEvents {
		events = append(events, AssistantStreamEvent{
			Event: event.Event,
			Data:  event.Data,
		})
	}
	return events
}

func buildAssistantStreamEvents(sessionID int64, resp *api.AssistantChatResponse) []assistantStreamEvent {
	events := make([]assistantStreamEvent, 0)
	events = append(events, assistantStreamEvent{
		Event: "start",
		Data: map[string]any{
			"session_id": sessionID,
		},
	})

	for _, delta := range splitTextByRunes(resp.GetReply(), assistantStreamMaxDeltaChunkRunes) {
		events = append(events, assistantStreamEvent{
			Event: "delta",
			Data:  map[string]any{"text": delta},
		})
	}

	for _, tool := range resp.GetToolCalls() {
		if tool == nil {
			continue
		}
		events = append(events, assistantStreamEvent{Event: "tool", Data: tool})
	}
	if usage := resp.GetUsage(); usage != nil {
		events = append(events, assistantStreamEvent{Event: "usage", Data: usage})
	}
	events = append(events, assistantStreamEvent{
		Event: "done",
		Data:  map[string]any{"finish_reason": "stop"},
	})
	return events
}

func splitTextByRunes(text string, chunkSize int) []string {
	if chunkSize <= 0 {
		chunkSize = assistantStreamMaxDeltaChunkRunes
	}
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}
	result := make([]string, 0, (len(runes)+chunkSize-1)/chunkSize)
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		result = append(result, string(runes[i:end]))
	}
	return result
}
