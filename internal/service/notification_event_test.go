package service

import (
	"strings"
	"testing"
	"volunteer-system/internal/model"
)

func TestRenderNotificationMessage_SignupRejected(t *testing.T) {
	title, content := renderNotificationMessage(NotificationEvent{
		EventType: model.NotificationEventSignupRejected,
		Payload: map[string]any{
			"activityTitle": "City Cleanup",
			"reason":        "quota reached",
		},
	})

	if !strings.Contains(title, "未通过") {
		t.Fatalf("unexpected title: %s", title)
	}
	if !strings.Contains(content, "City Cleanup") || !strings.Contains(content, "quota reached") {
		t.Fatalf("unexpected content: %s", content)
	}
}
