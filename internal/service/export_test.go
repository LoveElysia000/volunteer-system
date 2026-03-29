package service

import "testing"

func TestResolveOpsReportRangeAcceptsDateOnlyBounds(t *testing.T) {
	start, end, err := resolveOpsReportRange("weekly", "2026-03-29", "2026-03-29")
	if err != nil {
		t.Fatalf("resolveOpsReportRange returned error: %v", err)
	}

	if got := start.Format("2006-01-02 15:04:05"); got != "2026-03-29 00:00:00" {
		t.Fatalf("expected start of day, got %s", got)
	}
	if got := end.Format("2006-01-02 15:04:05"); got != "2026-03-29 23:59:59" {
		t.Fatalf("expected end of day, got %s", got)
	}
}

func TestResolveOpsReportRangePreservesDateTimeBounds(t *testing.T) {
	start, end, err := resolveOpsReportRange("weekly", "2026-03-29 08:00:00", "2026-03-29 18:30:00")
	if err != nil {
		t.Fatalf("resolveOpsReportRange returned error: %v", err)
	}

	if got := start.Format("2006-01-02 15:04:05"); got != "2026-03-29 08:00:00" {
		t.Fatalf("expected explicit start datetime to be preserved, got %s", got)
	}
	if got := end.Format("2006-01-02 15:04:05"); got != "2026-03-29 18:30:00" {
		t.Fatalf("expected explicit end datetime to be preserved, got %s", got)
	}
}
