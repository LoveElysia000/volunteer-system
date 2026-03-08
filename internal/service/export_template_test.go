package service

import (
	"bytes"
	"context"
	"testing"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/repository"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newExportServiceForTemplateTest(t *testing.T) *ExportService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	schema := []string{
		`CREATE TABLE activities (id INTEGER PRIMARY KEY, org_id INTEGER, start_time DATETIME)`,
		`CREATE TABLE activity_signups (id INTEGER PRIMARY KEY, activity_id INTEGER, signup_time DATETIME, check_in_status INTEGER, work_hour_status INTEGER)`,
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
		`INSERT INTO activities(id, org_id, start_time) VALUES (1, 1001, '2026-03-10 10:00:00')`,
		`INSERT INTO activity_signups(id, activity_id, signup_time, check_in_status, work_hour_status) VALUES (1, 1, '2026-03-10 10:00:00', 1, 1)`,
		`INSERT INTO activity_signups(id, activity_id, signup_time, check_in_status, work_hour_status) VALUES (2, 1, '2026-03-10 10:00:00', 1, 0)`,
		`INSERT INTO rbac_roles(id, role_code, role_name, status) VALUES (1, 'org_manager', 'Org Manager', 1)`,
		`INSERT INTO rbac_permissions(id, resource, action, status) VALUES (1, 'export', 'manage', 1)`,
		`INSERT INTO rbac_role_permissions(id, role_id, permission_id) VALUES (1, 1, 1)`,
		`INSERT INTO rbac_account_roles(id, account_id, role_id, scope_type, scope_id, status) VALUES (1, 200, 1, 'org', 1001, 1)`,
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
	c.Set(middleware.UserIDKey, "200")
	return &ExportService{Service: Service{ctx: ctx, c: c, repo: repo}}
}

func TestExportOpsReport_MonthlyTemplateColumns(t *testing.T) {
	svc := newExportServiceForTemplateTest(t)

	file, err := svc.ExportOpsReport(&api.ExportOpsReportRequest{
		PeriodType: "monthly",
		OrgId:      1001,
		Start:      "2026-03-01 00:00:00",
		End:        "2026-03-31 23:59:59",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	xlsx, err := excelize.OpenReader(bytes.NewReader(file.Content))
	if err != nil {
		t.Fatalf("open xlsx failed: %v", err)
	}
	sheet := xlsx.GetSheetName(0)
	rows, err := xlsx.GetRows(sheet)
	if err != nil {
		t.Fatalf("read rows failed: %v", err)
	}
	if len(rows) == 0 {
		t.Fatalf("expected header row")
	}

	expected := []string{
		"Period Type",
		"Organization ID",
		"Start",
		"End",
		"Activities Count",
		"Signups Count",
		"Attendance Count",
		"Workhours Count",
	}
	if len(rows[0]) < len(expected) {
		t.Fatalf("unexpected header length: %v", rows[0])
	}
	for i, header := range expected {
		if rows[0][i] != header {
			t.Fatalf("header[%d] expected %q got %q", i, header, rows[0][i])
		}
	}
}
