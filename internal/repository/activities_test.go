package repository

import (
	"context"
	"testing"
	"time"

	"volunteer-system/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGetMyActivitiesByFiltersReturnsVisibleActivities(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Activity{}, &model.ActivitySignup{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	repo := &Repository{
		ctx: context.Background(),
		DB:  db,
	}

	now := time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC)
	activities := []*model.Activity{
		{
			ID:               101,
			OrgID:            1,
			Title:            "Alpha Beach Cleanup",
			Description:      "Morning cleanup",
			CoverURL:         "cover-1",
			StartTime:        now,
			EndTime:          now.Add(2 * time.Hour),
			Location:         "Beach",
			Address:          "Addr 1",
			Duration:         2,
			MaxPeople:        20,
			CurrentPeople:    5,
			Status:           model.ActivityStatusRecruiting,
			CheckInCode:      "in-1",
			CheckInCodeHash:  "hash-in-1",
			CheckOutCode:     "out-1",
			CheckOutCodeHash: "hash-out-1",
			CreatedAt:        now.Add(-2 * time.Hour),
			UpdatedAt:        now.Add(-2 * time.Hour),
		},
		{
			ID:               102,
			OrgID:            1,
			Title:            "Bravo Library Support",
			Description:      "Afternoon shift",
			CoverURL:         "cover-2",
			StartTime:        now.Add(24 * time.Hour),
			EndTime:          now.Add(26 * time.Hour),
			Location:         "Library",
			Address:          "Addr 2",
			Duration:         2,
			MaxPeople:        10,
			CurrentPeople:    3,
			Status:           model.ActivityStatusFinished,
			CheckInCode:      "in-2",
			CheckInCodeHash:  "hash-in-2",
			CheckOutCode:     "out-2",
			CheckOutCodeHash: "hash-out-2",
			CreatedAt:        now.Add(-1 * time.Hour),
			UpdatedAt:        now.Add(-1 * time.Hour),
		},
		{
			ID:               103,
			OrgID:            1,
			Title:            "Charlie Park Patrol",
			Description:      "Should be excluded",
			CoverURL:         "cover-3",
			StartTime:        now.Add(48 * time.Hour),
			EndTime:          now.Add(50 * time.Hour),
			Location:         "Park",
			Address:          "Addr 3",
			Duration:         2,
			MaxPeople:        15,
			CurrentPeople:    2,
			Status:           model.ActivityStatusRecruiting,
			CheckInCode:      "in-3",
			CheckInCodeHash:  "hash-in-3",
			CheckOutCode:     "out-3",
			CheckOutCodeHash: "hash-out-3",
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	}
	if err := db.Create(&activities).Error; err != nil {
		t.Fatalf("create activities: %v", err)
	}

	signups := []*model.ActivitySignup{
		{
			ActivityID:     101,
			VolunteerID:    7,
			SignupTime:     now.Add(-30 * time.Minute),
			Status:         model.ActivitySignupStatusPending,
			CheckInStatus:  model.ActivityCheckInPending,
			CheckOutStatus: model.ActivityCheckOutPending,
			WorkHourStatus: model.WorkHourStatusPending,
			CreatedAt:      now.Add(-30 * time.Minute),
			UpdatedAt:      now.Add(-30 * time.Minute),
		},
		{
			ActivityID:     102,
			VolunteerID:    7,
			SignupTime:     now.Add(-20 * time.Minute),
			Status:         model.ActivitySignupStatusSuccess,
			CheckInStatus:  model.ActivityCheckInPending,
			CheckOutStatus: model.ActivityCheckOutPending,
			WorkHourStatus: model.WorkHourStatusPending,
			CreatedAt:      now.Add(-20 * time.Minute),
			UpdatedAt:      now.Add(-20 * time.Minute),
		},
		{
			ActivityID:     103,
			VolunteerID:    7,
			SignupTime:     now.Add(-10 * time.Minute),
			Status:         model.ActivitySignupStatusRejected,
			CheckInStatus:  model.ActivityCheckInPending,
			CheckOutStatus: model.ActivityCheckOutPending,
			WorkHourStatus: model.WorkHourStatusPending,
			CreatedAt:      now.Add(-10 * time.Minute),
			UpdatedAt:      now.Add(-10 * time.Minute),
		},
		{
			ActivityID:     103,
			VolunteerID:    8,
			SignupTime:     now.Add(-5 * time.Minute),
			Status:         model.ActivitySignupStatusSuccess,
			CheckInStatus:  model.ActivityCheckInPending,
			CheckOutStatus: model.ActivityCheckOutPending,
			WorkHourStatus: model.WorkHourStatusPending,
			CreatedAt:      now.Add(-5 * time.Minute),
			UpdatedAt:      now.Add(-5 * time.Minute),
		},
	}
	if err := db.Create(&signups).Error; err != nil {
		t.Fatalf("create signups: %v", err)
	}

	activityFilters := map[string]any{
		"act.status IN ?": []int32{model.ActivityStatusRecruiting, model.ActivityStatusFinished},
	}

	got, signupStatusMap, total, err := repo.GetMyActivitiesByFilters(db, 7, activityFilters, nil, 10, 0)
	if err != nil {
		t.Fatalf("GetMyActivitiesByFilters: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 activities, got %d", len(got))
	}
	if got[0].ID != 103 || got[1].ID != 102 || got[2].ID != 101 {
		t.Fatalf("expected activities ordered by created_at desc [103 102 101], got [%d %d %d]", got[0].ID, got[1].ID, got[2].ID)
	}
	if signupStatusMap[101] != model.ActivitySignupStatusPending ||
		signupStatusMap[102] != model.ActivitySignupStatusSuccess ||
		signupStatusMap[103] != model.ActivitySignupStatusRejected {
		t.Fatalf("expected signup status map to contain pending/success, got %+v", signupStatusMap)
	}

	keywordFilters := map[string]any{
		"act.status IN ?": []int32{model.ActivityStatusRecruiting, model.ActivityStatusFinished},
		"(act.title LIKE ? OR act.description LIKE ? OR act.location LIKE ?)": []any{"%Library%", "%Library%", "%Library%"},
	}
	got, _, total, err = repo.GetMyActivitiesByFilters(db, 7, keywordFilters, nil, 10, 0)
	if err != nil {
		t.Fatalf("GetMyActivitiesByFilters with keyword: %v", err)
	}
	if total != 1 || len(got) != 1 || got[0].ID != 102 {
		t.Fatalf("expected only activity 102 after keyword filter, got total=%d len=%d", total, len(got))
	}

	signupStatuses := []int32{model.ActivitySignupStatusPending}
	got, signupStatusMap, total, err = repo.GetMyActivitiesByFilters(db, 7, nil, signupStatuses, 10, 0)
	if err != nil {
		t.Fatalf("GetMyActivitiesByFilters with signup status: %v", err)
	}
	if total != 1 || len(got) != 1 || got[0].ID != 101 {
		t.Fatalf("expected only pending signup activity 101, got total=%d len=%d", total, len(got))
	}
	if signupStatusMap[101] != model.ActivitySignupStatusPending {
		t.Fatalf("expected activity 101 signup status pending, got %+v", signupStatusMap)
	}
}
