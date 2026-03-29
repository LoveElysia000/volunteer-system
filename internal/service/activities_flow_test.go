package service

import (
	"testing"
	"time"

	"volunteer-system/internal/api"
)

func TestBuildActivityListFilterMapAcceptsDateOnlyBounds(t *testing.T) {
	req := &api.ActivityListRequest{
		StartFrom: "2026-03-29",
		StartTo:   "2026-03-29",
	}

	filters, err := buildActivityListFilterMap(req)
	if err != nil {
		t.Fatalf("buildActivityListFilterMap returned error: %v", err)
	}

	startFrom, ok := filters["act.start_time >= ?"].(*time.Time)
	if !ok || startFrom == nil {
		t.Fatalf("expected startFrom filter, got %#v", filters["act.start_time >= ?"])
	}
	if got := startFrom.Format("2006-01-02 15:04:05"); got != "2026-03-29 00:00:00" {
		t.Fatalf("expected normalized startFrom to be start of day, got %s", got)
	}

	startTo, ok := filters["act.start_time <= ?"].(*time.Time)
	if !ok || startTo == nil {
		t.Fatalf("expected startTo filter, got %#v", filters["act.start_time <= ?"])
	}
	if got := startTo.Format("2006-01-02 15:04:05"); got != "2026-03-29 23:59:59" {
		t.Fatalf("expected normalized startTo to be end of day, got %s", got)
	}
}

func TestBuildActivityListFilterMapAcceptsDateTimeBounds(t *testing.T) {
	req := &api.ActivityListRequest{
		StartFrom: "2026-03-29 09:30:00",
		StartTo:   "2026-03-29 18:45:30",
	}

	filters, err := buildActivityListFilterMap(req)
	if err != nil {
		t.Fatalf("buildActivityListFilterMap returned error: %v", err)
	}

	startFrom := filters["act.start_time >= ?"].(*time.Time)
	if got := startFrom.Format("2006-01-02 15:04:05"); got != "2026-03-29 09:30:00" {
		t.Fatalf("expected datetime startFrom to be preserved, got %s", got)
	}

	startTo := filters["act.start_time <= ?"].(*time.Time)
	if got := startTo.Format("2006-01-02 15:04:05"); got != "2026-03-29 18:45:30" {
		t.Fatalf("expected datetime startTo to be preserved, got %s", got)
	}
}
