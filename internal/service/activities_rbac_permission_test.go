package service

import (
	"context"
	"testing"
	"volunteer-system/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newActivityServiceWithPermissionTestDB(t *testing.T) *ActivityService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	schema := []string{
		`CREATE TABLE activities (
			id INTEGER PRIMARY KEY,
			org_id INTEGER NOT NULL
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
		`INSERT INTO activities(id, org_id) VALUES (1, 1001)`,
		`INSERT INTO activities(id, org_id) VALUES (2, 1002)`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (1, 'super_admin', 'Super Admin', 1)`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (2, 'org_manager', 'Org Manager', 1)`,
		`INSERT INTO rbac_permissions(id, resource, action, status) VALUES (1, 'organization', 'manage', 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (1, 1, 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (2, 2, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (1, 100, 1, 'global', 0, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (2, 200, 2, 'org', 1001, 1)`,
	}
	for _, stmt := range seed {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	repo := &repository.Repository{DB: db}
	ctx := context.Background()
	repo.SetContext(&ctx)
	return &ActivityService{
		Service: Service{
			ctx:  ctx,
			repo: repo,
		},
	}
}

func TestEnsureActivityOperableByCurrentOrg_UsesRBAC(t *testing.T) {
	svc := newActivityServiceWithPermissionTestDB(t)

	t.Run("org scoped manager can operate own org activity", func(t *testing.T) {
		if _, err := svc.ensureActivityOperableByCurrentOrg(1, 200); err != nil {
			t.Fatalf("expected allow, got err: %v", err)
		}
	})

	t.Run("org scoped manager cannot operate other org activity", func(t *testing.T) {
		if _, err := svc.ensureActivityOperableByCurrentOrg(2, 200); err == nil {
			t.Fatalf("expected deny, got nil")
		}
	})

	t.Run("global super admin can operate any org activity", func(t *testing.T) {
		if _, err := svc.ensureActivityOperableByCurrentOrg(2, 100); err != nil {
			t.Fatalf("expected allow, got err: %v", err)
		}
	})
}
