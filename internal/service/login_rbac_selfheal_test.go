package service

import (
	"context"
	"testing"
	"volunteer-system/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newLoginServiceWithRBACTestDB(t *testing.T) *LoginService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	schema := []string{
		`CREATE TABLE sys_accounts (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL,
			mobile TEXT NOT NULL,
			mobile_hash TEXT NOT NULL,
			email TEXT NOT NULL,
			password TEXT NOT NULL,
			identity_type INTEGER NOT NULL,
			status INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE organizations (
			id INTEGER PRIMARY KEY,
			account_id INTEGER NOT NULL,
			org_name TEXT NOT NULL,
			license_code TEXT NOT NULL,
			contact_person TEXT NOT NULL,
			contact_phone TEXT NOT NULL,
			address TEXT NOT NULL DEFAULT '',
			logo_url TEXT NOT NULL DEFAULT '',
			introduction TEXT NOT NULL DEFAULT '',
			status INTEGER NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE rbac_roles (
			id INTEGER PRIMARY KEY,
			role_code TEXT NOT NULL UNIQUE,
			role_name TEXT NOT NULL,
			status INTEGER NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE rbac_account_roles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id INTEGER NOT NULL,
			role_id INTEGER NOT NULL,
			scope_type TEXT NOT NULL,
			scope_id INTEGER NOT NULL,
			status INTEGER NOT NULL DEFAULT 1,
			granted_by INTEGER NOT NULL DEFAULT 0,
			expires_at DATETIME NULL,
			created_at DATETIME,
			updated_at DATETIME,
			UNIQUE(account_id, role_id, scope_type, scope_id)
		)`,
	}
	for _, stmt := range schema {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create schema failed: %v", err)
		}
	}

	seed := []string{
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (1, 'volunteer', 'Volunteer', 1)`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (2, 'org_owner', 'Org Owner', 1)`,
		`INSERT INTO sys_accounts(id, username, mobile, mobile_hash, email, password, identity_type, status) VALUES (11, 'vol', 'enc1', 'h1', 'vol@example.com', 'p', 1, 1)`,
		`INSERT INTO sys_accounts(id, username, mobile, mobile_hash, email, password, identity_type, status) VALUES (12, 'org', 'enc2', 'h2', 'org@example.com', 'p', 2, 1)`,
		`INSERT INTO organizations(id, account_id, org_name, license_code, contact_person, contact_phone, status) VALUES (1001, 12, 'Org A', 'C1', 'Owner', '13800000000', 1)`,
	}
	for _, stmt := range seed {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	repo := &repository.Repository{DB: db}
	ctx := context.Background()
	repo.SetContext(&ctx)

	return &LoginService{
		Service: Service{
			ctx:  ctx,
			repo: repo,
		},
	}
}

func TestEnsureDefaultRBACBinding_Volunteer(t *testing.T) {
	svc := newLoginServiceWithRBACTestDB(t)
	account, err := svc.repo.FindByID(svc.repo.DB, 11)
	if err != nil {
		t.Fatalf("query account failed: %v", err)
	}

	if err := svc.ensureDefaultRBACBinding(svc.repo.DB, account); err != nil {
		t.Fatalf("self-heal should succeed: %v", err)
	}

	var count int64
	err = svc.repo.DB.Table("rbac_account_roles ar").
		Joins("JOIN rbac_roles r ON r.id = ar.role_id").
		Where("ar.account_id = ? AND r.role_code = ? AND ar.scope_type = ? AND ar.scope_id = ?", 11, "volunteer", "global", 0).
		Count(&count).Error
	if err != nil {
		t.Fatalf("count binding failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 binding, got %d", count)
	}
}

func TestEnsureDefaultRBACBinding_Organization(t *testing.T) {
	svc := newLoginServiceWithRBACTestDB(t)
	account, err := svc.repo.FindByID(svc.repo.DB, 12)
	if err != nil {
		t.Fatalf("query account failed: %v", err)
	}

	if err := svc.ensureDefaultRBACBinding(svc.repo.DB, account); err != nil {
		t.Fatalf("self-heal should succeed: %v", err)
	}

	var count int64
	err = svc.repo.DB.Table("rbac_account_roles ar").
		Joins("JOIN rbac_roles r ON r.id = ar.role_id").
		Where("ar.account_id = ? AND r.role_code = ? AND ar.scope_type = ? AND ar.scope_id = ?", 12, "org_owner", "org", 1001).
		Count(&count).Error
	if err != nil {
		t.Fatalf("count binding failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 binding, got %d", count)
	}
}

func TestEnsureDefaultRBACBinding_Idempotent(t *testing.T) {
	svc := newLoginServiceWithRBACTestDB(t)
	account, err := svc.repo.FindByID(svc.repo.DB, 11)
	if err != nil {
		t.Fatalf("query account failed: %v", err)
	}

	if err := svc.ensureDefaultRBACBinding(svc.repo.DB, account); err != nil {
		t.Fatalf("first self-heal should succeed: %v", err)
	}
	if err := svc.ensureDefaultRBACBinding(svc.repo.DB, account); err != nil {
		t.Fatalf("second self-heal should succeed: %v", err)
	}

	var count int64
	err = svc.repo.DB.Table("rbac_account_roles ar").
		Joins("JOIN rbac_roles r ON r.id = ar.role_id").
		Where("ar.account_id = ? AND r.role_code = ? AND ar.scope_type = ? AND ar.scope_id = ?", 11, "volunteer", "global", 0).
		Count(&count).Error
	if err != nil {
		t.Fatalf("count binding failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 binding, got %d", count)
	}
}
