package service

import (
	"context"
	"testing"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAuditServiceForBatchTest(t *testing.T) *AuditService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	schema := []string{
		`CREATE TABLE sys_accounts (
			id INTEGER PRIMARY KEY,
			identity_type INTEGER NOT NULL,
			status INTEGER NOT NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE audit_records (
			id INTEGER PRIMARY KEY,
			target_type INTEGER NOT NULL,
			target_id INTEGER NOT NULL,
			creator_id INTEGER NOT NULL,
			auditor_id INTEGER NOT NULL,
			old_content TEXT NOT NULL,
			new_content TEXT NOT NULL,
			audit_result INTEGER NOT NULL,
			reject_reason TEXT NOT NULL,
			audit_time DATETIME,
			created_at DATETIME,
			operation_type INTEGER NOT NULL,
			status INTEGER NOT NULL
		)`,
		`CREATE TABLE rbac_roles (
			id INTEGER PRIMARY KEY,
			role_code TEXT NOT NULL,
			role_name TEXT NOT NULL,
			status INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE rbac_permissions (
			id INTEGER PRIMARY KEY,
			resource TEXT NOT NULL,
			action TEXT NOT NULL,
			status INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE rbac_role_permissions (
			id INTEGER PRIMARY KEY,
			role_id INTEGER NOT NULL,
			permission_id INTEGER NOT NULL
		)`,
		`CREATE TABLE rbac_account_roles (
			id INTEGER PRIMARY KEY,
			account_id INTEGER NOT NULL,
			role_id INTEGER NOT NULL,
			scope_type TEXT NOT NULL,
			scope_id INTEGER NOT NULL,
			status INTEGER NOT NULL DEFAULT 1,
			expires_at DATETIME NULL
		)`,
	}
	for _, stmt := range schema {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create schema failed: %v", err)
		}
	}

	seed := []string{
		`INSERT INTO sys_accounts(id, identity_type, status) VALUES (200, 2, 1)`,
		`INSERT INTO audit_records(id, target_type, target_id, creator_id, auditor_id, old_content, new_content, audit_result, reject_reason, operation_type, status) VALUES (1, 2, 1001, 1, 0, '{}', '{}', 0, '', 2, 1)`,
		`INSERT INTO audit_records(id, target_type, target_id, creator_id, auditor_id, old_content, new_content, audit_result, reject_reason, operation_type, status) VALUES (2, 2, 1001, 1, 0, '{}', '{}', 1, 'done', 2, 2)`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (1, 'org_manager', 'Org Manager', 1)`,
		`INSERT INTO rbac_permissions(id, resource, action, status) VALUES (1, 'audit', 'review', 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (1, 1, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (1, 200, 1, 'org', 1001, 1)`,
	}
	for _, stmt := range seed {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	repo := &repository.Repository{DB: db}
	ctx := context.Background()
	repo.SetContext(&ctx)
	c := &app.RequestContext{}
	c.Set(middleware.UserIDKey, "200")

	return &AuditService{Service: Service{ctx: ctx, c: c, repo: repo}}
}

func TestAuditBatchDecision_PartialSuccess(t *testing.T) {
	svc := newAuditServiceForBatchTest(t)

	resp, err := svc.AuditBatchDecision(&api.AuditBatchDecisionRequest{
		Ids:    []int64{1, 2},
		Action: 2,
		Reason: "batch reject",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp.SuccessCount != 1 {
		t.Fatalf("expected success_count=1, got %d", resp.SuccessCount)
	}
	if len(resp.FailedIds) != 1 || resp.FailedIds[0] != 2 {
		t.Fatalf("expected failed_ids=[2], got %#v", resp.FailedIds)
	}

	record, err := svc.repo.GetAuditRecordByID(svc.repo.DB, 1)
	if err != nil {
		t.Fatalf("query audit record failed: %v", err)
	}
	if record.Status != model.AuditStatusRejected {
		t.Fatalf("expected status rejected, got %d", record.Status)
	}
}
