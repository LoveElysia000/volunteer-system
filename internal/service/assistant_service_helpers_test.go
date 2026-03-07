package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"volunteer-system/config"
	"volunteer-system/pkg/ai"
)

func TestResolveContextLimit(t *testing.T) {
	if got := resolveContextLimit(nil); got != assistantDefaultContextLimit {
		t.Fatalf("nil cfg context limit=%d want=%d", got, assistantDefaultContextLimit)
	}
	if got := resolveContextLimit(&config.AIConfig{MaxContextMessages: 2}); got != assistantDefaultContextLimit {
		t.Fatalf("small context limit=%d want=%d", got, assistantDefaultContextLimit)
	}
	if got := resolveContextLimit(&config.AIConfig{MaxContextMessages: 30}); got != 30 {
		t.Fatalf("normal context limit=%d want=30", got)
	}
	if got := resolveContextLimit(&config.AIConfig{MaxContextMessages: 999}); got != assistantMaxContextLimit {
		t.Fatalf("large context limit=%d want=%d", got, assistantMaxContextLimit)
	}
}

func TestNormalizeUserMessage(t *testing.T) {
	if _, err := normalizeUserMessage("   "); err == nil {
		t.Fatal("expected empty message error")
	}

	tooLong := strings.Repeat("a", assistantMaxUserMessageRunes+1)
	if _, err := normalizeUserMessage(tooLong); err == nil {
		t.Fatal("expected too long message error")
	}

	msg, err := normalizeUserMessage("  你好，助手  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "你好，助手" {
		t.Fatalf("normalized message=%q want=%q", msg, "你好，助手")
	}
}

func TestResolveUserVisibleModelError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: "模型服务暂时不可用"},
		{name: "api key", err: ai.ErrAPIKeyMissing, want: "AI 配置异常"},
		{name: "disabled", err: ai.ErrAIUnavailable, want: "AI 服务未启用"},
		{name: "deadline", err: context.DeadlineExceeded, want: "模型服务响应超时"},
		{name: "rate limit", err: errors.New("AI 接口调用失败: status=429"), want: "模型服务限流"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveUserVisibleModelError(tc.err); got != tc.want {
				t.Fatalf("resolve error=%q want=%q", got, tc.want)
			}
		})
	}
}

func TestBuildFallbackReplyShouldHideRawError(t *testing.T) {
	svc := &AssistantService{}
	raw := errors.New("database password leaked")

	reply := svc.buildFallbackReply([]*assistantToolResult{
		{ToolName: "activity_stats", Success: false, ErrorMsg: "权限不足"},
	}, raw)

	if strings.Contains(reply, raw.Error()) {
		t.Fatalf("fallback reply should not contain raw error, got: %s", reply)
	}
	if !strings.Contains(reply, "失败原因: 模型服务暂时不可用") {
		t.Fatalf("fallback reason not normalized, got: %s", reply)
	}
}

func TestSessionSceneAndTitleDefaults(t *testing.T) {
	scene, err := normalizeAssistantScene("")
	if err != nil {
		t.Fatalf("normalize empty scene error: %v", err)
	}
	if scene != assistantSceneGeneral {
		t.Fatalf("scene=%q want=%q", scene, assistantSceneGeneral)
	}
	if got := defaultSessionTitle(scene); got != "智能问答" {
		t.Fatalf("default title=%q want=%q", got, "智能问答")
	}

	scene, err = normalizeAssistantScene(assistantSceneOpsAdvisor)
	if err != nil {
		t.Fatalf("normalize ops scene error: %v", err)
	}
	if got := defaultSessionTitle(scene); got != "组织运营助手" {
		t.Fatalf("ops title=%q want=%q", got, "组织运营助手")
	}

	if _, err := normalizeAssistantScene("invalid"); err == nil {
		t.Fatal("expected invalid scene error")
	}
}

func TestGenerateSessionTitle(t *testing.T) {
	svc := &AssistantService{}

	if got := svc.generateSessionTitle("", assistantSceneActivityDraft); got != "活动草案助手" {
		t.Fatalf("empty message title=%q want=%q", got, "活动草案助手")
	}
	if got := svc.generateSessionTitle("  first line\nsecond line  ", assistantSceneGeneral); got != "first line second line" {
		t.Fatalf("generated title=%q want=%q", got, "first line second line")
	}
}
