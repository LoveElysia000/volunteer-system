package service

import (
	"testing"
	"time"
	"volunteer-system/internal/api"
)

func TestBuildActivityListFilterMap(t *testing.T) {
	req := &api.ActivityListRequest{
		Status:    1,
		Keyword:   "  cleanup  ",
		StartFrom: "2026-03-01 09:00:00",
		StartTo:   "2026-03-01 18:00:00",
		SortBy:    " start_time ",
		SortOrder: " asc ",
	}

	filters, err := buildActivityListFilterMap(req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if got, ok := filters["act.status = ?"].(int32); !ok || got != 1 {
		t.Fatalf("unexpected status filter: %v", filters["act.status = ?"])
	}
	if _, exists := filters["sort_by = ?"]; exists {
		t.Fatalf("sort_by should not be passed in filters map")
	}
	if _, exists := filters["sort_order = ?"]; exists {
		t.Fatalf("sort_order should not be passed in filters map")
	}
	if _, exists := filters["keyword"]; exists {
		t.Fatalf("keyword should be handled separately, got filter=%v", filters["keyword"])
	}

	startFrom, ok := filters["act.start_time >= ?"].(*time.Time)
	if !ok || startFrom == nil || startFrom.Format("2006-01-02 15:04:05") != "2026-03-01 09:00:00" {
		t.Fatalf("unexpected start_from filter: %v", filters["act.start_time >= ?"])
	}
	startTo, ok := filters["act.start_time <= ?"].(*time.Time)
	if !ok || startTo == nil || startTo.Format("2006-01-02 15:04:05") != "2026-03-01 18:00:00" {
		t.Fatalf("unexpected start_to filter: %v", filters["act.start_time <= ?"])
	}
}

func TestBuildActivityListFilterMapInvalidTime(t *testing.T) {
	req := &api.ActivityListRequest{
		StartFrom: "invalid-time",
	}
	_, err := buildActivityListFilterMap(req)
	if err == nil || err.Error() != "开始时间格式错误" {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestBuildActivityListFilterMapTimeRange(t *testing.T) {
	req := &api.ActivityListRequest{
		StartFrom: "2026-03-02 18:00:00",
		StartTo:   "2026-03-02 09:00:00",
	}
	_, err := buildActivityListFilterMap(req)
	if err == nil || err.Error() != "结束时间不能早于开始时间" {
		t.Fatalf("unexpected err: %v", err)
	}
}
