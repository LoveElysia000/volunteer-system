package service

import "testing"

func TestShouldPlanActivitySearch(t *testing.T) {
	cases := []struct {
		msg  string
		want bool
	}{
		{msg: "帮我查询最近活动", want: true},
		{msg: "最近有什么活动可以报名", want: true},
		{msg: "给我生成活动草案", want: false},
		{msg: "做一个环保活动方案", want: false},
	}

	for _, tc := range cases {
		if got := shouldPlanActivitySearch(tc.msg); got != tc.want {
			t.Fatalf("shouldPlanActivitySearch(%q) = %v, want %v", tc.msg, got, tc.want)
		}
	}
}

func TestTrimByStopWords(t *testing.T) {
	if got := trimByStopWords("社区清洁；附带说明"); got != "社区清洁" {
		t.Fatalf("trim result = %q, want %q", got, "社区清洁")
	}
	if got := trimByStopWords("河道清理。\n后续"); got != "河道清理" {
		t.Fatalf("trim result = %q, want %q", got, "河道清理")
	}
}

func TestAsIntFallback(t *testing.T) {
	if got := asInt(-1, 5); got != 5 {
		t.Fatalf("asInt negative = %d, want 5", got)
	}
	if got := asInt(0, 5); got != 5 {
		t.Fatalf("asInt zero = %d, want 5", got)
	}
	if got := asInt(8, 5); got != 8 {
		t.Fatalf("asInt positive = %d, want 8", got)
	}
}
