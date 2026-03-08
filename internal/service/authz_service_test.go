package service

import (
	"context"
	"strings"
	"testing"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/repository"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAuthzServiceForRuleTest(t *testing.T, userID string) *AuthzService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	schema := []string{
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
			description TEXT,
			status INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE rbac_role_permissions (
			id INTEGER PRIMARY KEY,
			role_id INTEGER NOT NULL,
			permission_id INTEGER NOT NULL,
			UNIQUE(role_id, permission_id)
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
		`CREATE TABLE rbac_change_logs (
			id INTEGER PRIMARY KEY,
			operator_id INTEGER NOT NULL,
			target_account_id INTEGER NOT NULL DEFAULT 0,
			target_role_id INTEGER NOT NULL DEFAULT 0,
			scope_type TEXT NOT NULL DEFAULT '',
			scope_id INTEGER NOT NULL DEFAULT 0,
			change_type TEXT NOT NULL,
			before_value TEXT,
			after_value TEXT,
			remark TEXT,
			created_at DATETIME
		)`,
	}
	for _, stmt := range schema {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create schema failed: %v", err)
		}
	}

	seed := []string{
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (1, 'super_admin', 'Super Admin', 1)`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (2, 'org_manager', 'Org Manager', 1)`,
		`INSERT INTO rbac_permissions(id, resource, action, description, status) VALUES (1, 'organization', 'manage', 'organization.manage', 1)`,
		`INSERT INTO rbac_permissions(id, resource, action, description, status) VALUES (2, 'rbac', 'manage', 'rbac.manage', 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (1, 1, 2)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (2, 2, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (1, 1, 1, 'global', 0, 1)`,
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

	return &AuthzService{
		Service: Service{
			ctx:  ctx,
			c:    c,
			repo: repo,
		},
	}
}

func TestSetRolePermissions_RBACManageGuardrails(t *testing.T) {
	svc := newAuthzServiceForRuleTest(t, "1")

	t.Run("non super admin role cannot own rbac.manage", func(t *testing.T) {
		_, err := svc.SetRolePermissions(&api.RolePermissionsSetRequest{
			RoleId:        2,
			PermissionIds: []int64{1, 2},
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "仅 super_admin 角色可拥有 rbac.manage") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("super admin role must keep rbac.manage", func(t *testing.T) {
		_, err := svc.SetRolePermissions(&api.RolePermissionsSetRequest{
			RoleId:        1,
			PermissionIds: []int64{1},
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "super_admin 角色必须保留 rbac.manage") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("super admin role cannot own business permissions", func(t *testing.T) {
		_, err := svc.SetRolePermissions(&api.RolePermissionsSetRequest{
			RoleId:        1,
			PermissionIds: []int64{1, 2},
		})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "super_admin 角色仅允许保留 rbac.manage") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
