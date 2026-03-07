package util

import "testing"

func TestContainsAny(t *testing.T) {
	if !ContainsAny("最近有什么活动", "活动", "报名") {
		t.Fatal("expected contains any to be true")
	}
	if ContainsAny("最近有什么活动", "草案", "运营") {
		t.Fatal("expected contains any to be false")
	}
}

func TestTruncateText(t *testing.T) {
	if got := TruncateText("abcdef", 0); got != "" {
		t.Fatalf("truncate max=0 got=%q", got)
	}
	if got := TruncateText("abc", 10); got != "abc" {
		t.Fatalf("truncate no-op got=%q", got)
	}
	if got := TruncateText("abcdef", 3); got != "abc" {
		t.Fatalf("truncate max=3 got=%q", got)
	}
	if got := TruncateText("abcdef", 5); got != "ab..." {
		t.Fatalf("truncate with ellipsis got=%q", got)
	}
}

func TestContainsInt64(t *testing.T) {
	if !ContainsInt64([]int64{1, 2, 3}, 2) {
		t.Fatal("expected contains int64 true")
	}
	if ContainsInt64([]int64{1, 2, 3}, 4) {
		t.Fatal("expected contains int64 false")
	}
}

func TestClampInt32(t *testing.T) {
	if got := ClampInt32(-1); got != 0 {
		t.Fatalf("negative clamp got=%d want=0", got)
	}
	if got := ClampInt32(12); got != 12 {
		t.Fatalf("normal clamp got=%d want=12", got)
	}
	if got := ClampInt32(int(^uint32(0)>>1) + 10); got != int32(^uint32(0)>>1) {
		t.Fatalf("overflow clamp got=%d", got)
	}
}

func TestReverseInPlace(t *testing.T) {
	items := []int{1, 2, 3, 4}
	ReverseInPlace(items)
	want := []int{4, 3, 2, 1}
	for i := range want {
		if items[i] != want[i] {
			t.Fatalf("reverse result=%v want=%v", items, want)
		}
	}
}
