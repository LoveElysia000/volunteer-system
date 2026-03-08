package service

import (
	"testing"
	"volunteer-system/config"
)

func TestEmailChannel_DisabledConfigSkipsSend(t *testing.T) {
	cfgPtr := config.GetConfig()
	origin := *cfgPtr
	defer func() {
		*cfgPtr = origin
	}()

	disabled := &config.EmailConfig{Enabled: false}
	cfgPtr.Email = disabled

	called := 0
	originSend := sendEmailFn
	sendEmailFn = func(cfg *config.EmailConfig, to, subject, body string) error {
		called++
		return nil
	}
	defer func() {
		sendEmailFn = originSend
	}()

	svc := &NotificationService{}
	if err := svc.sendEventEmail(NotificationEvent{}); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if called != 0 {
		t.Fatalf("expected sendEmail not called, got %d", called)
	}
}
