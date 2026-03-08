package service

import (
	"context"
	"fmt"
	"testing"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/repository"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newVolunteerServiceForRBACListTest(t *testing.T, userID string, seed []string) *VolunteerService {
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
			created_at DATETIME
		)`,
		`CREATE TABLE volunteers (
			id INTEGER PRIMARY KEY,
			account_id INTEGER NOT NULL,
			real_name TEXT NOT NULL,
			gender INTEGER NOT NULL,
			avatar_url TEXT,
			total_hours REAL NOT NULL DEFAULT 0,
			service_count INTEGER NOT NULL DEFAULT 0,
			credit_score INTEGER NOT NULL DEFAULT 100,
			audit_status INTEGER NOT NULL DEFAULT 0,
			status INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE org_members (
			id INTEGER PRIMARY KEY,
			org_id INTEGER NOT NULL,
			volunteer_id INTEGER NOT NULL,
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

	return &VolunteerService{
		Service: Service{
			ctx:  ctx,
			c:    c,
			repo: repo,
		},
	}
}

func TestVolunteerList_UsesRBACOrgScope(t *testing.T) {
	seed := []string{
		`INSERT INTO organizations(id, account_id, org_name, created_at) VALUES (1001, 9001, 'Org A', '2026-03-01 00:00:00')`,
		`INSERT INTO organizations(id, account_id, org_name, created_at) VALUES (1002, 9002, 'Org B', '2026-03-01 00:00:00')`,
		`INSERT INTO volunteers(id, account_id, real_name, gender, status, created_at, updated_at) VALUES (1, 501, 'Alice', 2, 1, '2026-03-01 00:00:00', '2026-03-01 00:00:00')`,
		`INSERT INTO volunteers(id, account_id, real_name, gender, status, created_at, updated_at) VALUES (2, 502, 'Bob', 1, 1, '2026-03-01 00:00:00', '2026-03-01 00:00:00')`,
		`INSERT INTO org_members(id, org_id, volunteer_id, status) VALUES (1, 1001, 1, 2)`,
		`INSERT INTO org_members(id, org_id, volunteer_id, status) VALUES (2, 1002, 2, 2)`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (1, 'super_admin', 'Super Admin', 1)`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (2, 'org_manager', 'Org Manager', 1)`,
		`INSERT INTO rbac_permissions(id, resource, action, status) VALUES (1, 'organization', 'manage', 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (1, 1, 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (2, 2, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (1, 100, 1, 'global', 0, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (2, 200, 2, 'org', 1001, 1)`,
	}

	t.Run("org scoped manager only sees volunteers in managed org", func(t *testing.T) {
		svc := newVolunteerServiceForRBACListTest(t, "200", seed)
		resp, err := svc.VolunteerList(&api.VolunteerListRequest{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("volunteer list failed: %v", err)
		}
		if resp.Total != 1 || len(resp.List) != 1 || resp.List[0].Id != 1 {
			t.Fatalf("unexpected result: total=%d list=%v", resp.Total, resp.List)
		}
	})

	t.Run("global manager sees volunteers across orgs", func(t *testing.T) {
		svc := newVolunteerServiceForRBACListTest(t, "100", seed)
		resp, err := svc.VolunteerList(&api.VolunteerListRequest{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("volunteer list failed: %v", err)
		}
		if resp.Total != 2 || len(resp.List) != 2 {
			t.Fatalf("unexpected result: total=%d list=%v", resp.Total, resp.List)
		}
	})

	t.Run("org scopes are not truncated when more than 500", func(t *testing.T) {
		largeSeed := []string{
			`INSERT INTO volunteers(id, account_id, real_name, gender, status, created_at, updated_at) VALUES (999, 5999, 'Carol', 2, 1, '2026-03-01 00:00:00', '2026-03-01 00:00:00')`,
			`INSERT INTO org_members(id, org_id, volunteer_id, status) VALUES (999, 501, 999, 2)`,
			`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (2, 'org_manager', 'Org Manager', 1)`,
			`INSERT INTO rbac_permissions(id, resource, action, status) VALUES (1, 'organization', 'manage', 1)`,
			`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (1, 2, 1)`,
		}
		for i := 1; i <= 501; i++ {
			largeSeed = append(largeSeed, fmt.Sprintf(
				"INSERT INTO organizations(id, account_id, org_name, created_at) VALUES (%d, %d, 'Org %d', '2026-03-01 00:00:00')",
				i, 9000+i, i,
			))
			largeSeed = append(largeSeed, fmt.Sprintf(
				"INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (%d, 300, 2, 'org', %d, 1)",
				i, i,
			))
		}

		svc := newVolunteerServiceForRBACListTest(t, "300", largeSeed)
		resp, err := svc.VolunteerList(&api.VolunteerListRequest{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("volunteer list failed: %v", err)
		}
		if resp.Total != 1 || len(resp.List) != 1 || resp.List[0].Id != 999 {
			t.Fatalf("unexpected result: total=%d list=%v", resp.Total, resp.List)
		}
	})
}
