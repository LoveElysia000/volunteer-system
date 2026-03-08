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

func newWorkHourServiceForRBACListTest(t *testing.T, userID string, seed []string) *WorkHourService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	schema := []string{
		`CREATE TABLE sys_accounts (
			id INTEGER PRIMARY KEY,
			identity_type INTEGER NOT NULL,
			status INTEGER NOT NULL DEFAULT 1,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE activities (
			id INTEGER PRIMARY KEY,
			org_id INTEGER NOT NULL
		)`,
		`CREATE TABLE work_hour_logs (
			id INTEGER PRIMARY KEY,
			volunteer_id INTEGER,
			activity_id INTEGER,
			signup_id INTEGER,
			operation_type INTEGER,
			hours_delta REAL,
			service_count_delta INTEGER,
			before_total_hours REAL,
			after_total_hours REAL,
			before_service_count INTEGER,
			after_service_count INTEGER,
			work_hour_version INTEGER,
			ref_log_id INTEGER,
			reason TEXT,
			operator_id INTEGER,
			idempotency_key TEXT,
			created_at DATETIME
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

	return &WorkHourService{
		Service: Service{
			ctx:  ctx,
			c:    c,
			repo: repo,
		},
	}
}

func TestWorkHourLogList_UsesRBACOrgScope(t *testing.T) {
	seed := []string{
		`INSERT INTO sys_accounts(id, identity_type, status) VALUES (100, 2, 1)`,
		`INSERT INTO sys_accounts(id, identity_type, status) VALUES (200, 2, 1)`,
		`INSERT INTO activities(id, org_id) VALUES (1, 1001)`,
		`INSERT INTO activities(id, org_id) VALUES (2, 1002)`,
		`INSERT INTO work_hour_logs(id, volunteer_id, activity_id, signup_id, operation_type, created_at) VALUES (1, 1, 1, 1, 1, '2026-03-01 00:00:00')`,
		`INSERT INTO work_hour_logs(id, volunteer_id, activity_id, signup_id, operation_type, created_at) VALUES (2, 2, 2, 2, 1, '2026-03-01 00:00:00')`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (1, 'super_admin', 'Super Admin', 1)`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (2, 'org_manager', 'Org Manager', 1)`,
		`INSERT INTO rbac_permissions(id, resource, action, status) VALUES (1, 'organization', 'manage', 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (1, 1, 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (2, 2, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (1, 100, 1, 'global', 0, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (2, 200, 2, 'org', 1001, 1)`,
	}

	t.Run("org scoped manager only sees managed org logs", func(t *testing.T) {
		svc := newWorkHourServiceForRBACListTest(t, "200", seed)
		resp, err := svc.WorkHourLogList(&api.WorkHourLogListRequest{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("work hour log list failed: %v", err)
		}
		if resp.Total != 1 || len(resp.List) != 1 || resp.List[0].ActivityId != 1 {
			t.Fatalf("unexpected result: total=%d list=%v", resp.Total, resp.List)
		}
	})

	t.Run("global manager sees all org logs", func(t *testing.T) {
		svc := newWorkHourServiceForRBACListTest(t, "100", seed)
		resp, err := svc.WorkHourLogList(&api.WorkHourLogListRequest{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("work hour log list failed: %v", err)
		}
		if resp.Total != 2 || len(resp.List) != 2 {
			t.Fatalf("unexpected result: total=%d list=%v", resp.Total, resp.List)
		}
	})

	t.Run("org scopes are not truncated when more than 500", func(t *testing.T) {
		largeSeed := []string{
			`INSERT INTO sys_accounts(id, identity_type, status) VALUES (300, 2, 1)`,
			`INSERT INTO activities(id, org_id) VALUES (999, 501)`,
			`INSERT INTO work_hour_logs(id, volunteer_id, activity_id, signup_id, operation_type, created_at) VALUES (999, 9, 999, 9, 1, '2026-03-01 00:00:00')`,
			`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (2, 'org_manager', 'Org Manager', 1)`,
			`INSERT INTO rbac_permissions(id, resource, action, status) VALUES (1, 'organization', 'manage', 1)`,
			`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (1, 2, 1)`,
		}
		for i := 1; i <= 501; i++ {
			largeSeed = append(largeSeed, fmt.Sprintf(
				"INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (%d, 300, 2, 'org', %d, 1)",
				i, i,
			))
		}

		svc := newWorkHourServiceForRBACListTest(t, "300", largeSeed)
		resp, err := svc.WorkHourLogList(&api.WorkHourLogListRequest{Page: 1, PageSize: 20})
		if err != nil {
			t.Fatalf("work hour log list failed: %v", err)
		}
		if resp.Total != 1 || len(resp.List) != 1 || resp.List[0].Id != 999 {
			t.Fatalf("unexpected result: total=%d list=%v", resp.Total, resp.List)
		}
	})
}
