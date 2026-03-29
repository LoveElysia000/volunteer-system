package util

import "testing"

func TestParseDateFilterBoundAcceptsDateOnly(t *testing.T) {
	start, err := ParseDateFilterBound("2026-03-29", false)
	if err != nil {
		t.Fatalf("ParseDateFilterBound returned error: %v", err)
	}
	if got := start.Format(DateTimeLayout); got != "2026-03-29 00:00:00" {
		t.Fatalf("expected start of day, got %s", got)
	}

	end, err := ParseDateFilterBound("2026-03-29", true)
	if err != nil {
		t.Fatalf("ParseDateFilterBound returned error: %v", err)
	}
	if got := end.Format(DateTimeLayout); got != "2026-03-29 23:59:59" {
		t.Fatalf("expected end of day, got %s", got)
	}
}

func TestParseDateFilterBoundPreservesDateTime(t *testing.T) {
	value, err := ParseDateFilterBound("2026-03-29 09:30:00", true)
	if err != nil {
		t.Fatalf("ParseDateFilterBound returned error: %v", err)
	}
	if got := value.Format(DateTimeLayout); got != "2026-03-29 09:30:00" {
		t.Fatalf("expected datetime to be preserved, got %s", got)
	}
}
