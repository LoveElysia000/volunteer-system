package service

import (
	"context"
	"testing"
	"volunteer-system/internal/api"
	"volunteer-system/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newRegisterServiceWithRBACTestDB(t *testing.T) *RegisterService {
	return newRegisterServiceWithRBACTestDBWithSeed(t, true)
}

func newRegisterServiceWithRBACTestDBWithoutSeed(t *testing.T) *RegisterService {
	return newRegisterServiceWithRBACTestDBWithSeed(t, false)
}

func newRegisterServiceWithRBACTestDBWithSeed(t *testing.T, seedDefaultRoles bool) *RegisterService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	schema := []string{
		`CREATE TABLE sys_accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			mobile TEXT NOT NULL,
			mobile_hash TEXT NOT NULL,
			email TEXT NOT NULL,
			password TEXT NOT NULL,
			identity_type INTEGER NOT NULL,
			status INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME,
			last_login_time DATETIME NULL,
			updated_at DATETIME,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE volunteers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id INTEGER NOT NULL,
			real_name TEXT NOT NULL,
			gender INTEGER NOT NULL,
			birthday DATETIME NULL,
			id_card TEXT NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			introduction TEXT NOT NULL DEFAULT '',
			total_hours REAL NOT NULL DEFAULT 0,
			total_points INTEGER NOT NULL DEFAULT 0,
			level_id INTEGER NOT NULL DEFAULT 1,
			service_count INTEGER NOT NULL DEFAULT 0,
			credit_score INTEGER NOT NULL DEFAULT 100,
			status INTEGER NOT NULL,
			audit_status INTEGER NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
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
			id INTEGER PRIMARY KEY AUTOINCREMENT,
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

	if seedDefaultRoles {
		seed := []string{
			`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (1, 'volunteer', 'Volunteer', 1)`,
			`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (2, 'org_owner', 'Org Owner', 1)`,
		}
		for _, stmt := range seed {
			if err := db.Exec(stmt).Error; err != nil {
				t.Fatalf("seed failed: %v", err)
			}
		}
	}

	repo := &repository.Repository{DB: db}
	ctx := context.Background()
	repo.SetContext(&ctx)
	return &RegisterService{
		Service: Service{
			ctx:  ctx,
			repo: repo,
		},
	}
}

func TestRegisterVolunteer_BindsDefaultRole(t *testing.T) {
	svc := newRegisterServiceWithRBACTestDB(t)

	_, err := svc.RegisterVolunteer(&api.VolunteerRegisterRequest{
		UserName: "volunteer_a",
		Name:     "Volunteer A",
		Phone:    "13812345678",
		Email:    "volunteer_a@example.com",
		Password: "Password123",
		Age:      24,
		Gender:   "男",
	})
	if err != nil {
		t.Fatalf("register volunteer failed: %v", err)
	}

	account, err := svc.repo.FindByEmail(svc.repo.DB, "volunteer_a@example.com")
	if err != nil {
		t.Fatalf("query account failed: %v", err)
	}

	var binding struct {
		RoleCode  string
		ScopeType string
		ScopeID   int64
	}
	err = svc.repo.DB.Table("rbac_account_roles ar").
		Select("r.role_code as role_code, ar.scope_type as scope_type, ar.scope_id as scope_id").
		Joins("JOIN rbac_roles r ON r.id = ar.role_id").
		Where("ar.account_id = ?", account.ID).
		Take(&binding).Error
	if err != nil {
		t.Fatalf("query role binding failed: %v", err)
	}
	if binding.RoleCode != "volunteer" || binding.ScopeType != "global" || binding.ScopeID != 0 {
		t.Fatalf("unexpected binding: %+v", binding)
	}
}

func TestRegisterOrganization_BindsDefaultRole(t *testing.T) {
	svc := newRegisterServiceWithRBACTestDB(t)

	_, err := svc.RegisterOrganization(&api.OrganizationRegisterRequest{
		Name:             "Owner A",
		Phone:            "13912345678",
		Email:            "org_a@example.com",
		Password:         "Password123",
		OrganizationName: "Org A",
		Code:             "ORG-CODE-001",
	})
	if err != nil {
		t.Fatalf("register organization failed: %v", err)
	}

	account, err := svc.repo.FindByEmail(svc.repo.DB, "org_a@example.com")
	if err != nil {
		t.Fatalf("query account failed: %v", err)
	}

	var orgID int64
	if err := svc.repo.DB.Table("organizations").Select("id").Where("account_id = ?", account.ID).Take(&orgID).Error; err != nil {
		t.Fatalf("query organization failed: %v", err)
	}

	var binding struct {
		RoleCode  string
		ScopeType string
		ScopeID   int64
	}
	err = svc.repo.DB.Table("rbac_account_roles ar").
		Select("r.role_code as role_code, ar.scope_type as scope_type, ar.scope_id as scope_id").
		Joins("JOIN rbac_roles r ON r.id = ar.role_id").
		Where("ar.account_id = ?", account.ID).
		Take(&binding).Error
	if err != nil {
		t.Fatalf("query role binding failed: %v", err)
	}
	if binding.RoleCode != "org_owner" || binding.ScopeType != "org" || binding.ScopeID != orgID {
		t.Fatalf("unexpected binding: %+v, org_id=%d", binding, orgID)
	}
}

func TestRegisterVolunteer_SucceedsWhenDefaultRoleMissing(t *testing.T) {
	svc := newRegisterServiceWithRBACTestDBWithoutSeed(t)

	_, err := svc.RegisterVolunteer(&api.VolunteerRegisterRequest{
		UserName: "volunteer_no_role",
		Name:     "Volunteer No Role",
		Phone:    "13800000001",
		Email:    "volunteer_no_role@example.com",
		Password: "Password123",
		Age:      30,
		Gender:   "男",
	})
	if err != nil {
		t.Fatalf("register volunteer should not fail when role seed missing, got: %v", err)
	}

	account, err := svc.repo.FindByEmail(svc.repo.DB, "volunteer_no_role@example.com")
	if err != nil {
		t.Fatalf("account should be created, query failed: %v", err)
	}
	if account.ID <= 0 {
		t.Fatalf("account id not generated")
	}

	var volunteerID int64
	if err := svc.repo.DB.Table("volunteers").Select("id").Where("account_id = ?", account.ID).Take(&volunteerID).Error; err != nil {
		t.Fatalf("volunteer profile should be created: %v", err)
	}

	var count int64
	if err := svc.repo.DB.Table("rbac_account_roles").Where("account_id = ?", account.ID).Count(&count).Error; err != nil {
		t.Fatalf("count rbac bindings failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no rbac binding without role seed, got %d", count)
	}
}

func TestRegisterOrganization_SucceedsWhenDefaultRoleMissing(t *testing.T) {
	svc := newRegisterServiceWithRBACTestDBWithoutSeed(t)

	_, err := svc.RegisterOrganization(&api.OrganizationRegisterRequest{
		Name:             "Owner No Role",
		Phone:            "13900000001",
		Email:            "org_no_role@example.com",
		Password:         "Password123",
		OrganizationName: "Org No Role",
		Code:             "ORG-NO-ROLE-001",
	})
	if err != nil {
		t.Fatalf("register organization should not fail when role seed missing, got: %v", err)
	}

	account, err := svc.repo.FindByEmail(svc.repo.DB, "org_no_role@example.com")
	if err != nil {
		t.Fatalf("account should be created, query failed: %v", err)
	}
	if account.ID <= 0 {
		t.Fatalf("account id not generated")
	}

	var orgID int64
	if err := svc.repo.DB.Table("organizations").Select("id").Where("account_id = ?", account.ID).Take(&orgID).Error; err != nil {
		t.Fatalf("organization should be created: %v", err)
	}

	var count int64
	if err := svc.repo.DB.Table("rbac_account_roles").Where("account_id = ?", account.ID).Count(&count).Error; err != nil {
		t.Fatalf("count rbac bindings failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no rbac binding without role seed, got %d", count)
	}
}
