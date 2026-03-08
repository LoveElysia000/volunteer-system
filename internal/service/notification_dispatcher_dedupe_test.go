package service

import (
	"context"
	"testing"
	"volunteer-system/config"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newNotificationServiceForDedupeTest(t *testing.T) *NotificationService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	schema := []string{
		`CREATE TABLE notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_type TEXT NOT NULL,
			biz_type TEXT NOT NULL,
			biz_id INTEGER NOT NULL,
			source_org_id INTEGER NOT NULL,
			sender_id INTEGER NOT NULL,
			dedupe_key TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE notification_inbox (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			notification_id INTEGER NOT NULL,
			receiver_id INTEGER NOT NULL,
			source_org_id INTEGER NOT NULL,
			read_status INTEGER NOT NULL,
			read_at DATETIME,
			inbox_status INTEGER NOT NULL,
			archived_reason TEXT NOT NULL DEFAULT '',
			archived_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE sys_accounts (
			id INTEGER PRIMARY KEY,
			email TEXT,
			status INTEGER,
			deleted_at DATETIME
		)`,
		`INSERT INTO sys_accounts(id, email, status) VALUES (1, 'volunteer@example.com', 1)`,
	}

	for _, stmt := range schema {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("init schema failed: %v", err)
		}
	}

	repo := &repository.Repository{DB: db}
	ctx := context.Background()
	repo.SetContext(&ctx)

	return &NotificationService{Service: Service{ctx: ctx, repo: repo}}
}

func TestHandleEventAndDispatchEmail_DedupeSkipsSecondEmail(t *testing.T) {
	svc := newNotificationServiceForDedupeTest(t)

	cfgPtr := config.GetConfig()
	originCfg := *cfgPtr
	defer func() { *cfgPtr = originCfg }()
	cfgPtr.Email = &config.EmailConfig{Enabled: true}

	originSend := sendEmailFn
	defer func() { sendEmailFn = originSend }()
	emailCalls := 0
	sendEmailFn = func(cfg *config.EmailConfig, to, subject, body string) error {
		emailCalls++
		return nil
	}

	evt := NotificationEvent{
		EventType:   model.NotificationEventMemberJoinApproved,
		BizType:     model.NotificationBizTypeMembership,
		BizID:       9001,
		SourceOrgID: 1001,
		ActorID:     2,
		Payload: map[string]any{
			"receiverAccountID": int64(1),
		},
		DedupeKey: "member.join.approved:9001",
	}

	if err := svc.handleEventAndDispatchEmail(evt); err != nil {
		t.Fatalf("first dispatch failed: %v", err)
	}
	if err := svc.handleEventAndDispatchEmail(evt); err != nil {
		t.Fatalf("second dispatch failed: %v", err)
	}

	if emailCalls != 1 {
		t.Fatalf("expected 1 email call, got %d", emailCalls)
	}

	var notifyCount int64
	if err := svc.repo.DB.Table("notifications").Count(&notifyCount).Error; err != nil {
		t.Fatalf("count notifications failed: %v", err)
	}
	if notifyCount != 1 {
		t.Fatalf("expected 1 notification row, got %d", notifyCount)
	}

	var inboxCount int64
	if err := svc.repo.DB.Table("notification_inbox").Count(&inboxCount).Error; err != nil {
		t.Fatalf("count inbox failed: %v", err)
	}
	if inboxCount != 1 {
		t.Fatalf("expected 1 inbox row, got %d", inboxCount)
	}
}
