package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"volunteer-system/config"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/ai"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestResolveContextLimit(t *testing.T) {
	if got := resolveContextLimit(nil); got != assistantDefaultContextLimit {
		t.Fatalf("nil cfg context limit = %d, want %d", got, assistantDefaultContextLimit)
	}

	if got := resolveContextLimit(&config.AIConfig{MaxContextMessages: 2}); got != assistantDefaultContextLimit {
		t.Fatalf("too small context limit = %d, want %d", got, assistantDefaultContextLimit)
	}

	if got := resolveContextLimit(&config.AIConfig{MaxContextMessages: 30}); got != 30 {
		t.Fatalf("normal context limit = %d, want 30", got)
	}

	if got := resolveContextLimit(&config.AIConfig{MaxContextMessages: 999}); got != assistantMaxContextLimit {
		t.Fatalf("too large context limit = %d, want %d", got, assistantMaxContextLimit)
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
		t.Fatalf("normalized message = %q, want %q", msg, "你好，助手")
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
				t.Fatalf("resolve error = %q, want %q", got, tc.want)
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
		t.Fatalf("scene = %q, want %q", scene, assistantSceneGeneral)
	}
	if got := defaultSessionTitle(scene); got != "智能问答" {
		t.Fatalf("default title = %q, want %q", got, "智能问答")
	}

	scene, err = normalizeAssistantScene(assistantSceneOpsAdvisor)
	if err != nil {
		t.Fatalf("normalize ops scene error: %v", err)
	}
	if got := defaultSessionTitle(scene); got != "组织运营助手" {
		t.Fatalf("ops title = %q, want %q", got, "组织运营助手")
	}

	if _, err := normalizeAssistantScene("invalid"); err == nil {
		t.Fatal("expected invalid scene error")
	}
}

func TestGenerateSessionTitle(t *testing.T) {
	svc := &AssistantService{}

	if got := svc.generateSessionTitle("", assistantSceneActivityDraft); got != "活动草案助手" {
		t.Fatalf("empty message title = %q, want %q", got, "活动草案助手")
	}

	if got := svc.generateSessionTitle("  first line\nsecond line  ", assistantSceneGeneral); got != "first line second line" {
		t.Fatalf("generated title = %q, want %q", got, "first line second line")
	}
}

func TestBuildPromptMessagesFromSessionHistory(t *testing.T) {
	svc := &AssistantService{}
	history := []*model.AiMessage{
		{Role: assistantRoleUser, Content: "用户问题"},
		{Role: assistantRoleTool, Content: `{"count":2}`},
		{Role: assistantRoleAssistant, Content: "助手回复"},
		{Role: assistantRoleUser, Content: "   "}, // 空消息应被过滤
		{Role: 99, Content: "unknown role"},       // 未知角色应被忽略
	}

	messages := svc.buildPromptMessages(assistantSceneGeneral, history)
	if len(messages) != 4 {
		t.Fatalf("prompt messages len = %d, want 4", len(messages))
	}

	if messages[0].Role != "system" || !strings.Contains(messages[0].Content, "通用问答") {
		t.Fatalf("system prompt mismatch: role=%q content=%q", messages[0].Role, messages[0].Content)
	}

	if messages[1].Role != "user" || messages[1].Content != "用户问题" {
		t.Fatalf("user message mismatch: %#v", messages[1])
	}

	if messages[2].Role != "system" || messages[2].Content != "工具结果(JSON):\n{\"count\":2}" {
		t.Fatalf("tool message mismatch: %#v", messages[2])
	}

	if messages[3].Role != "assistant" || messages[3].Content != "助手回复" {
		t.Fatalf("assistant message mismatch: %#v", messages[3])
	}
}

func TestSessionFlowSmoke(t *testing.T) {
	ctx := context.Background()
	reqCtx := app.NewContext(0)
	reqCtx.Set(middleware.UserIDKey, "1001")

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite error: %v", err)
	}
	if err := db.AutoMigrate(&model.AiSession{}, &model.AiMessage{}, &model.AiToolCall{}, &model.AiUsageDaily{}); err != nil {
		t.Fatalf("auto migrate error: %v", err)
	}

	repo := &repository.Repository{DB: db}
	repo.SetContext(&ctx)

	svc := &AssistantService{
		Service: Service{
			ctx:  ctx,
			c:    reqCtx,
			repo: repo,
		},
		toolService: &AssistantToolService{},
	}

	createResp, err := svc.CreateSession(&api.AssistantCreateSessionRequest{})
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}
	if createResp.SessionId <= 0 {
		t.Fatalf("session id invalid: %d", createResp.SessionId)
	}

	session, err := repo.GetAiSessionByIDAndUser(db, createResp.SessionId, 1001)
	if err != nil {
		t.Fatalf("query session error: %v", err)
	}
	if session.Scene != assistantSceneGeneral {
		t.Fatalf("session scene = %q, want %q", session.Scene, assistantSceneGeneral)
	}
	if session.Title != "智能问答" {
		t.Fatalf("session title = %q, want %q", session.Title, "智能问答")
	}

	firstChat, err := svc.Chat(&api.AssistantChatRequest{
		SessionId: createResp.SessionId,
		Message:   "你好",
		Stream:    false,
	})
	if err != nil {
		t.Fatalf("first Chat error: %v", err)
	}
	if strings.TrimSpace(firstChat.Reply) == "" {
		t.Fatal("first reply should not be empty")
	}

	secondChat, err := svc.Chat(&api.AssistantChatRequest{
		SessionId: createResp.SessionId,
		Message:   "继续",
		Stream:    false,
	})
	if err != nil {
		t.Fatalf("second Chat error: %v", err)
	}
	if strings.TrimSpace(secondChat.Reply) == "" {
		t.Fatal("second reply should not be empty")
	}

	msgResp, err := svc.GetSessionMessages(&api.AssistantSessionMessagesRequest{Id: createResp.SessionId})
	if err != nil {
		t.Fatalf("GetSessionMessages error: %v", err)
	}
	if len(msgResp.List) != 4 {
		t.Fatalf("messages len = %d, want 4", len(msgResp.List))
	}

	wantRoles := []int32{assistantRoleUser, assistantRoleAssistant, assistantRoleUser, assistantRoleAssistant}
	for i, item := range msgResp.List {
		if item.SessionId != createResp.SessionId {
			t.Fatalf("message[%d] session_id = %d, want %d", i, item.SessionId, createResp.SessionId)
		}
		if item.Role != wantRoles[i] {
			t.Fatalf("message[%d] role = %d, want %d", i, item.Role, wantRoles[i])
		}
		if item.SeqNo != int32(i+1) {
			t.Fatalf("message[%d] seq_no = %d, want %d", i, item.SeqNo, i+1)
		}
	}
}
