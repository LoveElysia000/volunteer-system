package repository

import (
	"context"
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRBACRepositoryTestDB(t *testing.T) *Repository {
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
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (1, 'org_manager', 'Org Manager', 1)`,
		`INSERT INTO rbac_permissions(id, resource, action, status) VALUES (1, 'organization', 'manage', 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (1, 1, 1)`,
	}
	for _, stmt := range seed {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	for i := 1; i <= 501; i++ {
		stmt := fmt.Sprintf(
			"INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (%d, 100, 1, 'org', %d, 1)",
			i,
			i,
		)
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("seed scopes failed at %d: %v", i, err)
		}
	}

	repo := &Repository{DB: db}
	ctx := context.Background()
	repo.SetContext(&ctx)
	return repo
}

func TestListOrgScopeIDsByPermission(t *testing.T) {
	repo := newRBACRepositoryTestDB(t)

	t.Run("no limit returns all scopes", func(t *testing.T) {
		ids, err := repo.ListOrgScopeIDsByPermission(repo.DB, 100, "organization", "manage", 0)
		if err != nil {
			t.Fatalf("list org scopes failed: %v", err)
		}
		if len(ids) != 501 {
			t.Fatalf("expected 501 scopes, got %d", len(ids))
		}
		if ids[0] != 1 || ids[len(ids)-1] != 501 {
			t.Fatalf("unexpected scopes boundary: first=%d last=%d", ids[0], ids[len(ids)-1])
		}
	})

	t.Run("positive limit still works", func(t *testing.T) {
		ids, err := repo.ListOrgScopeIDsByPermission(repo.DB, 100, "organization", "manage", 2)
		if err != nil {
			t.Fatalf("list org scopes failed: %v", err)
		}
		if len(ids) != 2 || ids[0] != 1 || ids[1] != 2 {
			t.Fatalf("unexpected limited scopes: %v", ids)
		}
	})
}

func TestHasActiveRBACBindingByRoleCodeAndScope(t *testing.T) {
	repo := newRBACRepositoryTestDB(t)

	t.Run("active binding exists", func(t *testing.T) {
		ok, err := repo.HasActiveRBACBindingByRoleCodeAndScope(repo.DB, 100, "org_manager", "org", 1)
		if err != nil {
			t.Fatalf("check binding failed: %v", err)
		}
		if !ok {
			t.Fatalf("expected binding exists")
		}
	})

	t.Run("role mismatch returns false", func(t *testing.T) {
		ok, err := repo.HasActiveRBACBindingByRoleCodeAndScope(repo.DB, 100, "org_owner", "org", 1)
		if err != nil {
			t.Fatalf("check binding failed: %v", err)
		}
		if ok {
			t.Fatalf("expected binding not exists")
		}
	})

	t.Run("inactive binding returns false", func(t *testing.T) {
		if err := repo.DB.Exec("UPDATE rbac_account_roles SET status = 0 WHERE account_id = ? AND scope_type = ? AND scope_id = ?", 100, "org", 1).Error; err != nil {
			t.Fatalf("disable binding failed: %v", err)
		}
		ok, err := repo.HasActiveRBACBindingByRoleCodeAndScope(repo.DB, 100, "org_manager", "org", 1)
		if err != nil {
			t.Fatalf("check binding failed: %v", err)
		}
		if ok {
			t.Fatalf("expected inactive binding to be filtered")
		}
	})
}
