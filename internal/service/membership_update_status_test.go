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

func newMembershipServiceForUpdateStatusTest(t *testing.T) *MembershipService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	schema := []string{
		`CREATE TABLE organizations (
			id INTEGER PRIMARY KEY,
			account_id INTEGER NOT NULL,
			org_name TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE org_members (
			id INTEGER PRIMARY KEY,
			org_id INTEGER NOT NULL,
			volunteer_id INTEGER NOT NULL,
			status INTEGER NOT NULL,
			role INTEGER NOT NULL DEFAULT 1,
			applied_at DATETIME,
			joined_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE volunteers (
			id INTEGER PRIMARY KEY,
			account_id INTEGER NOT NULL,
			real_name TEXT,
			id_card TEXT,
			created_at DATETIME,
			updated_at DATETIME
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
		`INSERT INTO organizations(id, account_id, org_name) VALUES (1001, 500, 'Org A')`,
		`INSERT INTO volunteers(id, account_id, real_name, id_card) VALUES (3001, 7001, 'V', 'ID')`,
		`INSERT INTO org_members(id, org_id, volunteer_id, status, role) VALUES (9001, 1001, 3001, 1, 1)`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (3, 'org_manager', 'Org Manager', 1)`,
		`INSERT INTO rbac_permissions(id, resource, action, status) VALUES (2, 'membership', 'manage', 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (2, 3, 2)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (2, 200, 3, 'org', 1001, 1)`,
	}
	for _, stmt := range seed {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("seed data failed: %v", err)
		}
	}

	repo := &repository.Repository{DB: db}
	ctx := context.Background()
	repo.SetContext(&ctx)

	c := &app.RequestContext{}
	c.Set(middleware.UserIDKey, "200")

	return &MembershipService{
		Service: Service{
			ctx:  ctx,
			c:    c,
			repo: repo,
		},
	}
}

func TestUpdateMemberStatus_UsesTokenUserNotBodyAccountID(t *testing.T) {
	svc := newMembershipServiceForUpdateStatusTest(t)

	req := &api.MemberStatusUpdateRequest{
		MembershipId: 9001,
		Status:       model.MemberStatusRejected,
	}

	_, err := svc.UpdateMemberStatus(req)
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}

	member, err := svc.repo.GetMembershipByID(svc.repo.DB, 9001)
	if err != nil {
		t.Fatalf("query member failed: %v", err)
	}
	if member.Status != model.MemberStatusRejected {
		t.Fatalf("expected status %d, got %d", model.MemberStatusRejected, member.Status)
	}
}
