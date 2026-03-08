package service

import (
	"context"
	"testing"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/repository"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newOrganizationServiceWithRBAC(t *testing.T, userID string) *OrganizationService {
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
			status INTEGER NOT NULL DEFAULT 1
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
		`INSERT INTO organizations(id, account_id, org_name, status) VALUES (1001, 9001, 'Org A', 1)`,
		`INSERT INTO organizations(id, account_id, org_name, status) VALUES (1002, 9002, 'Org B', 1)`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (1, 'super_admin', 'Super Admin', 1)`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (2, 'org_manager', 'Org Manager', 1)`,
		`INSERT INTO rbac_permissions(id, resource, action, status) VALUES (1, 'organization', 'manage', 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (1, 1, 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (2, 2, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (1, 100, 1, 'global', 0, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (2, 200, 2, 'org', 1001, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (3, 300, 2, 'org', 1002, 1)`,
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

	return &OrganizationService{Service: Service{ctx: ctx, c: c, repo: repo}}
}

func TestOrganizationWriteOps_ByRBAC(t *testing.T) {
	t.Run("super_admin allowed globally", func(t *testing.T) {
		svc := newOrganizationServiceWithRBAC(t, "100")
		if _, err := svc.requireOrganizationManagePermission(1002); err != nil {
			t.Fatalf("expected allow, got err: %v", err)
		}
	})

	t.Run("org_owner_or_manager allowed in own org", func(t *testing.T) {
		svc := newOrganizationServiceWithRBAC(t, "200")
		if _, err := svc.requireOrganizationManagePermission(1001); err != nil {
			t.Fatalf("expected allow, got err: %v", err)
		}
	})

	t.Run("volunteer denied", func(t *testing.T) {
		svc := newOrganizationServiceWithRBAC(t, "300")
		if _, err := svc.requireOrganizationManagePermission(1001); err == nil {
			t.Fatalf("expected deny, got nil")
		}
	})
}
