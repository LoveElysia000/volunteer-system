package service

import (
	"context"
	"strings"
	"testing"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/repository"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newExportServiceForScopeTest(t *testing.T, userID string, seed []string) *ExportService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	schema := []string{
		`CREATE TABLE organizations (
			id INTEGER PRIMARY KEY,
			account_id INTEGER NOT NULL,
			org_name TEXT
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
	for _, stmt := range seed {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	repo := &repository.Repository{DB: db}
	ctx := context.Background()
	repo.SetContext(&ctx)
	c := &app.RequestContext{}
	c.Set(middleware.UserIDKey, userID)

	return &ExportService{
		Service: Service{
			ctx:  ctx,
			c:    c,
			repo: repo,
		},
	}
}

func TestGetCurrentOrgID_FallbackToRBACScope(t *testing.T) {
	seed := []string{
		`INSERT INTO organizations(id, account_id, org_name) VALUES (1001, 9001, 'Org A')`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (1, 'org_manager', 'Org Manager', 1)`,
		`INSERT INTO rbac_permissions(id, resource, action, status) VALUES (1, 'export', 'manage', 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (1, 1, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (1, 200, 1, 'org', 1001, 1)`,
	}
	svc := newExportServiceForScopeTest(t, "200", seed)

	orgID, err := svc.getCurrentOrgID()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if orgID != 1001 {
		t.Fatalf("expected org 1001, got %d", orgID)
	}
}

func TestGetCurrentOrgID_MultiScopeReturnsError(t *testing.T) {
	seed := []string{
		`INSERT INTO organizations(id, account_id, org_name) VALUES (1001, 9001, 'Org A')`,
		`INSERT INTO organizations(id, account_id, org_name) VALUES (1002, 9002, 'Org B')`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (1, 'org_manager', 'Org Manager', 1)`,
		`INSERT INTO rbac_permissions(id, resource, action, status) VALUES (1, 'export', 'manage', 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (1, 1, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (1, 201, 1, 'org', 1001, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (2, 201, 1, 'org', 1002, 1)`,
	}
	svc := newExportServiceForScopeTest(t, "201", seed)

	_, err := svc.getCurrentOrgID()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "多个组织作用域") {
		t.Fatalf("unexpected error: %v", err)
	}
}
