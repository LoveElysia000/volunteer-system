package repository

import (
	"context"
	"testing"
	"time"
	"volunteer-system/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupRepositoryTestDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatalf("auto migrate failed: %v", err)
		}
	}
	return db
}

func TestGetActivitiesByFiltersWithoutOrganizationJoin(t *testing.T) {
	db := setupRepositoryTestDB(t, &model.Activity{})
	now := time.Now()
	if err := db.Create(&model.Activity{
		ID:            1,
		OrgID:         100,
		Title:         "Beach Cleanup",
		Description:   "desc",
		CoverURL:      "cover",
		StartTime:     now.Add(time.Hour),
		EndTime:       now.Add(2 * time.Hour),
		Location:      "city",
		Address:       "addr",
		Duration:      1,
		MaxPeople:     100,
		CurrentPeople: 1,
		Status:        model.ActivityStatusRecruiting,
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error; err != nil {
		t.Fatalf("seed activity failed: %v", err)
	}

	repo := &Repository{ctx: context.Background()}
	filters := map[string]any{
		"act.status = ?": model.ActivityStatusRecruiting,
	}

	list, total, err := repo.GetActivitiesByFilters(db, filters, 10, 0)
	if err != nil {
		t.Fatalf("GetActivitiesByFilters failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("unexpected total: %d", total)
	}
	if len(list) != 1 || list[0].ID != 1 {
		t.Fatalf("unexpected result list: %+v", list)
	}
}
