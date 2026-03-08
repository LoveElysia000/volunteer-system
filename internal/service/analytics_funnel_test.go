package service

import (
	"context"
	"math"
	"testing"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/repository"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAnalyticsServiceForTest(t *testing.T) *AnalyticsService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	schema := []string{
		`CREATE TABLE volunteers (id INTEGER PRIMARY KEY, account_id INTEGER, created_at DATETIME)`,
		`CREATE TABLE org_members (id INTEGER PRIMARY KEY, org_id INTEGER, volunteer_id INTEGER, status INTEGER, created_at DATETIME)`,
		`CREATE TABLE activities (id INTEGER PRIMARY KEY, org_id INTEGER)`,
		`CREATE TABLE activity_signups (id INTEGER PRIMARY KEY, activity_id INTEGER, volunteer_id INTEGER, status INTEGER, check_in_status INTEGER, work_hour_status INTEGER, signup_time DATETIME)`,
		`CREATE TABLE rbac_roles (id INTEGER PRIMARY KEY, role_code TEXT, role_name TEXT, status INTEGER)`,
		`CREATE TABLE rbac_permissions (id INTEGER PRIMARY KEY, resource TEXT, action TEXT, status INTEGER)`,
		`CREATE TABLE rbac_role_permissions (id INTEGER PRIMARY KEY, role_id INTEGER, permission_id INTEGER)`,
		`CREATE TABLE rbac_account_roles (id INTEGER PRIMARY KEY, account_id INTEGER, role_id INTEGER, scope_type TEXT, scope_id INTEGER, status INTEGER, expires_at DATETIME NULL)`,
	}
	for _, stmt := range schema {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create schema failed: %v", err)
		}
	}

	seed := []string{
		`INSERT INTO activities(id, org_id) VALUES (5001, 1001)`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (1, 'org_manager', 'Org Manager', 1)`,
		`INSERT INTO rbac_permissions(id, resource, action, status) VALUES (1, 'analytics', 'org.read', 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (1, 1, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (1, 200, 1, 'org', 1001, 1)`,
		`INSERT INTO activity_signups(id, activity_id, volunteer_id, status, check_in_status, work_hour_status, signup_time) VALUES (1, 5001, 1, 2, 1, 1, '2026-03-10 10:00:00')`,
		`INSERT INTO activity_signups(id, activity_id, volunteer_id, status, check_in_status, work_hour_status, signup_time) VALUES (2, 5001, 2, 2, 1, 1, '2026-03-10 10:00:00')`,
		`INSERT INTO activity_signups(id, activity_id, volunteer_id, status, check_in_status, work_hour_status, signup_time) VALUES (3, 5001, 3, 2, 1, 0, '2026-03-10 10:00:00')`,
		`INSERT INTO activity_signups(id, activity_id, volunteer_id, status, check_in_status, work_hour_status, signup_time) VALUES (4, 5001, 4, 1, 0, 0, '2026-03-10 10:00:00')`,
		`INSERT INTO org_members(id, org_id, volunteer_id, status, created_at) VALUES (1, 1001, 1, 2, '2026-03-09 00:00:00')`,
		`INSERT INTO org_members(id, org_id, volunteer_id, status, created_at) VALUES (2, 1001, 2, 2, '2026-03-09 00:00:00')`,
		`INSERT INTO org_members(id, org_id, volunteer_id, status, created_at) VALUES (3, 1001, 3, 2, '2026-03-09 00:00:00')`,
		`INSERT INTO org_members(id, org_id, volunteer_id, status, created_at) VALUES (4, 1001, 4, 2, '2026-03-09 00:00:00')`,
		`INSERT INTO org_members(id, org_id, volunteer_id, status, created_at) VALUES (5, 1001, 5, 2, '2026-03-09 00:00:00')`,
	}
	for _, stmt := range seed {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}
	for i := 1; i <= 10; i++ {
		stmt := `INSERT INTO volunteers(id, account_id, created_at) VALUES (?, ?, '2026-03-08 00:00:00')`
		if err := db.Exec(stmt, i, i).Error; err != nil {
			t.Fatalf("seed volunteer failed: %v", err)
		}
	}

	repo := &repository.Repository{DB: db}
	ctx := context.Background()
	repo.SetContext(&ctx)
	c := &app.RequestContext{}
	c.Set(middleware.UserIDKey, "200")

	return &AnalyticsService{Service: Service{ctx: ctx, c: c, repo: repo}}
}

func TestOrgFunnelSummary_ComputesConversionRates(t *testing.T) {
	svc := newAnalyticsServiceForTest(t)

	resp, err := svc.OrgFunnelSummary(&api.OrgFunnelSummaryRequest{
		OrgId: 1001,
		Start: "2026-03-01 00:00:00",
		End:   "2026-03-31 23:59:59",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.RegistrationCount != 10 || resp.MembershipCount != 5 || resp.SignupCount != 4 || resp.AttendanceCount != 3 || resp.WorkhourCount != 2 {
		t.Fatalf("unexpected counts: %#v", resp)
	}
	if math.Abs(resp.RegistrationToMembershipRate-50) > 0.01 ||
		math.Abs(resp.MembershipToSignupRate-80) > 0.01 ||
		math.Abs(resp.SignupToAttendanceRate-75) > 0.01 ||
		math.Abs(resp.AttendanceToWorkhourRate-66.67) > 0.01 {
		t.Fatalf("unexpected rates: %#v", resp)
	}
}
