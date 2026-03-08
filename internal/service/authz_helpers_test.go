package service

import (
	"context"
	"testing"
	"volunteer-system/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRBACServiceForTest(t *testing.T) *Service {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	schema := []string{
		`CREATE TABLE rbac_roles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			role_code TEXT NOT NULL,
			role_name TEXT NOT NULL,
			status INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE rbac_permissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			resource TEXT NOT NULL,
			action TEXT NOT NULL,
			status INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE rbac_role_permissions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			role_id INTEGER NOT NULL,
			permission_id INTEGER NOT NULL
		)`,
		`CREATE TABLE rbac_account_roles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
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
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (1, 'super_admin', 'Super Admin', 1)`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (2, 'org_owner', 'Org Owner', 1)`,
		`INSERT INTO rbac_permissions(id, resource, action, status) VALUES (1, 'organization', 'manage', 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (1, 1, 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (2, 2, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (1, 100, 1, 'global', 0, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (2, 200, 2, 'org', 1001, 1)`,
	}

	for _, stmt := range seed {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("seed data failed: %v", err)
		}
	}

	repo := &repository.Repository{DB: db}
	ctx := context.Background()
	repo.SetContext(&ctx)

	return &Service{
		ctx:  ctx,
		repo: repo,
	}
}

func TestHasPermissionByScope(t *testing.T) {
	svc := newRBACServiceForTest(t)

	t.Run("super admin global allow", func(t *testing.T) {
		err := svc.requireOrgPermission(100, 2002, "organization", "manage")
		if err != nil {
			t.Fatalf("expected allow, got err: %v", err)
		}
	})

	t.Run("org role allow in same org", func(t *testing.T) {
		err := svc.requireOrgPermission(200, 1001, "organization", "manage")
		if err != nil {
			t.Fatalf("expected allow, got err: %v", err)
		}
	})

	t.Run("org role deny in other org", func(t *testing.T) {
		err := svc.requireOrgPermission(200, 9999, "organization", "manage")
		if err == nil {
			t.Fatalf("expected deny, got nil")
		}
	})
}
