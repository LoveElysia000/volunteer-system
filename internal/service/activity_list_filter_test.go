package service

import (
	"context"
	"testing"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/repository"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newActivityServiceForListTest(t *testing.T) *ActivityService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	schema := []string{
		`CREATE TABLE sys_accounts (
			id INTEGER PRIMARY KEY,
			identity_type INTEGER NOT NULL,
			status INTEGER NOT NULL,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE organizations (
			id INTEGER PRIMARY KEY,
			account_id INTEGER NOT NULL,
			org_name TEXT
		)`,
		`CREATE TABLE activities (
			id INTEGER PRIMARY KEY,
			org_id INTEGER NOT NULL,
			title TEXT,
			description TEXT,
			cover_url TEXT,
			start_time DATETIME,
			end_time DATETIME,
			location TEXT,
			address TEXT,
			duration REAL,
			max_people INTEGER,
			current_people INTEGER,
			status INTEGER,
			created_at DATETIME,
			updated_at DATETIME
		)`,
	}
	for _, stmt := range schema {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create schema failed: %v", err)
		}
	}

	seed := []string{
		`INSERT INTO sys_accounts(id, identity_type, status) VALUES (1, 2, 1)`,
		`INSERT INTO organizations(id, account_id, org_name) VALUES (1001, 1, 'Org A')`,
		`INSERT INTO activities(id, org_id, title, description, start_time, end_time, location, duration, max_people, current_people, status, created_at, updated_at) VALUES (1, 1001, 'Beach Cleanup', 'clean beach', '2026-03-10 10:00:00', '2026-03-10 12:00:00', 'beach', 2.0, 50, 10, 1, '2026-03-01 00:00:00', '2026-03-01 00:00:00')`,
		`INSERT INTO activities(id, org_id, title, description, start_time, end_time, location, duration, max_people, current_people, status, created_at, updated_at) VALUES (2, 1001, 'Park Trees', 'plant trees', '2026-03-25 10:00:00', '2026-03-25 12:00:00', 'park', 2.0, 50, 10, 1, '2026-03-02 00:00:00', '2026-03-02 00:00:00')`,
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
	c.Set(middleware.UserIDKey, "1")

	return &ActivityService{Service: Service{ctx: ctx, c: c, repo: repo}}
}

func TestActivityList_FilterByKeywordAndTimeRange(t *testing.T) {
	svc := newActivityServiceForListTest(t)

	resp, err := svc.ActivityList(&api.ActivityListRequest{
		Page:      1,
		PageSize:  20,
		Status:    1,
		Keyword:   "Beach",
		StartFrom: "2026-03-01 00:00:00",
		StartTo:   "2026-03-15 23:59:59",
		SortBy:    "start_time",
		SortOrder: "asc",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.Total != 1 {
		t.Fatalf("expected total=1, got %d", resp.Total)
	}
	if len(resp.List) != 1 || resp.List[0].Title != "Beach Cleanup" {
		t.Fatalf("unexpected list result: %#v", resp.List)
	}
}
