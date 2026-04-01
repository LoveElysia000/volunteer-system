package repository

import (
	"context"
	"testing"
	"time"

	"volunteer-system/internal/api"
	"volunteer-system/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListVolunteerExportRecordsAppliesMultiStatusFilters(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Organization{}, &model.SysAccount{}, &model.Volunteer{}, &model.OrgMember{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	repo := &Repository{
		ctx: context.Background(),
		DB:  db,
	}

	now := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	org := &model.Organization{
		ID:            1,
		AccountID:     100,
		OrgName:       "Sunrise Org",
		LicenseCode:   "LIC-001",
		ContactPerson: "Owner",
		ContactPhone:  "123456",
		Address:       "Somewhere",
		LogoURL:       "logo",
		Introduction:  "intro",
		Status:        1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := db.Create(org).Error; err != nil {
		t.Fatalf("create organization: %v", err)
	}

	accounts := []*model.SysAccount{
		{ID: 10, UserName: "alpha", Mobile: "enc1", MobileHash: "hash1", Email: "a@example.com", Password: "pwd", IdentityType: 1, Status: 1, CreatedAt: now, UpdatedAt: now},
		{ID: 11, UserName: "bravo", Mobile: "enc2", MobileHash: "hash2", Email: "b@example.com", Password: "pwd", IdentityType: 1, Status: 1, CreatedAt: now, UpdatedAt: now},
		{ID: 12, UserName: "charlie", Mobile: "enc3", MobileHash: "hash3", Email: "c@example.com", Password: "pwd", IdentityType: 1, Status: 1, CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&accounts).Error; err != nil {
		t.Fatalf("create accounts: %v", err)
	}

	volunteers := []*model.Volunteer{
		{ID: 101, AccountID: 10, RealName: "Alice", Gender: 1, IDCard: "id-1", AvatarURL: "a", Introduction: "intro", TotalHours: 12, TotalPoints: 1, LevelID: 1, ServiceCount: 2, CreditScore: 100, Status: model.VolunteerActiveStatus, AuditStatus: model.VolunteerAuditStatusApproved, CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-3 * time.Hour)},
		{ID: 102, AccountID: 11, RealName: "Bob", Gender: 1, IDCard: "id-2", AvatarURL: "b", Introduction: "intro", TotalHours: 8, TotalPoints: 1, LevelID: 1, ServiceCount: 1, CreditScore: 100, Status: model.VolunteerInactiveStatus, AuditStatus: model.VolunteerAuditStatusPending, CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: 103, AccountID: 12, RealName: "Cindy", Gender: 2, IDCard: "id-3", AvatarURL: "c", Introduction: "intro", TotalHours: 6, TotalPoints: 1, LevelID: 1, ServiceCount: 1, CreditScore: 100, Status: model.VolunteerEtcStatus, AuditStatus: model.VolunteerAuditStatusRejected, CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
	}
	if err := db.Create(&volunteers).Error; err != nil {
		t.Fatalf("create volunteers: %v", err)
	}

	members := []*model.OrgMember{
		{ID: 1, OrgID: 1, VolunteerID: 101, Role: 1, Status: model.MemberStatusActive, AppliedAt: now, CreatedAt: now, UpdatedAt: now},
		{ID: 2, OrgID: 1, VolunteerID: 102, Role: 1, Status: model.MemberStatusActive, AppliedAt: now, CreatedAt: now, UpdatedAt: now},
		{ID: 3, OrgID: 1, VolunteerID: 103, Role: 1, Status: model.MemberStatusActive, AppliedAt: now, CreatedAt: now, UpdatedAt: now},
	}
	if err := db.Create(&members).Error; err != nil {
		t.Fatalf("create org members: %v", err)
	}

	req := &api.ExportVolunteersRequest{
		Status:      []int32{model.VolunteerActiveStatus, model.VolunteerInactiveStatus},
		AuditStatus: []int32{model.VolunteerAuditStatusApproved, model.VolunteerAuditStatusPending},
	}

	rows, err := repo.ListVolunteerExportRecords(db, req, 1, 10, 0)
	if err != nil {
		t.Fatalf("ListVolunteerExportRecords: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows after multi-status filter, got %d", len(rows))
	}
	if rows[0].VolunteerID != 102 || rows[1].VolunteerID != 101 {
		t.Fatalf("expected volunteers [102 101] ordered by created_at desc, got [%d %d]", rows[0].VolunteerID, rows[1].VolunteerID)
	}
	if rows[0].OrgName != "Sunrise Org" || rows[1].OrgName != "Sunrise Org" {
		t.Fatalf("expected org name to be filled, got %+v", rows)
	}
}
