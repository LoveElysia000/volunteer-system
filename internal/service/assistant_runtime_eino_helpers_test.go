package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"volunteer-system/config"
	"volunteer-system/pkg/ai"

	"github.com/cloudwego/eino/schema"
)

func TestResolveEinoMaxRetries(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.AIConfig
		want int
	}{
		{name: "nil config", cfg: nil, want: 0},
		{name: "negative", cfg: &config.AIConfig{MaxRetries: -1}, want: 0},
		{name: "normal", cfg: &config.AIConfig{MaxRetries: 2}, want: 2},
		{name: "clamp", cfg: &config.AIConfig{MaxRetries: 99}, want: 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveEinoMaxRetries(tt.cfg)
			if got != tt.want {
				t.Fatalf("resolveEinoMaxRetries()=%d want=%d", got, tt.want)
			}
		})
	}
}

func TestResolveEinoEnableStream(t *testing.T) {
	if resolveEinoEnableStream(nil) {
		t.Fatal("resolveEinoEnableStream(nil) should be false")
	}
	if !resolveEinoEnableStream(&config.AIConfig{
		Eino: config.AIEinoConfig{EnableStream: true},
	}) {
		t.Fatal("resolveEinoEnableStream() should respect config")
	}
}

func TestBuildEinoCheckpointID(t *testing.T) {
	if got := buildEinoCheckpointID(7, "req-1"); got != "assistant:7:req-1" {
		t.Fatalf("session checkpoint id mismatch: %s", got)
	}
	if got := buildEinoCheckpointID(0, "req-1"); got != "assistant:req:req-1" {
		t.Fatalf("request checkpoint id mismatch: %s", got)
	}
	if got := buildEinoCheckpointID(0, ""); !strings.HasPrefix(got, "assistant:req:") {
		t.Fatalf("generated checkpoint id mismatch: %s", got)
	}
}

func TestResolveEinoModelRetryConfig(t *testing.T) {
	if cfg := resolveEinoModelRetryConfig(nil); cfg != nil {
		t.Fatal("retry config should be nil when ai config is nil")
	}

	retryCfg := resolveEinoModelRetryConfig(&config.AIConfig{MaxRetries: 2})
	if retryCfg == nil {
		t.Fatal("retry config should not be nil")
	}
	if retryCfg.MaxRetries != 2 {
		t.Fatalf("unexpected max retries: %d", retryCfg.MaxRetries)
	}
	if retryCfg.IsRetryAble == nil {
		t.Fatal("retry config should include IsRetryAble")
	}
	if retryCfg.BackoffFunc == nil {
		t.Fatal("retry config should include BackoffFunc")
	}
}

func TestResolveEinoRetryBackoff(t *testing.T) {
	if got := resolveEinoRetryBackoff(0); got != 200*time.Millisecond {
		t.Fatalf("attempt=0 backoff mismatch: %s", got)
	}
	if got := resolveEinoRetryBackoff(1); got != 400*time.Millisecond {
		t.Fatalf("attempt=1 backoff mismatch: %s", got)
	}
	if got := resolveEinoRetryBackoff(2); got != 800*time.Millisecond {
		t.Fatalf("attempt=2 backoff mismatch: %s", got)
	}
	if got := resolveEinoRetryBackoff(99); got != 2*time.Second {
		t.Fatalf("attempt=99 backoff mismatch: %s", got)
	}
}

func TestShouldRetryEinoRuntimeError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "api key", err: ai.ErrAPIKeyMissing, want: false},
		{name: "unavailable", err: ai.ErrAIUnavailable, want: false},
		{name: "context canceled", err: context.Canceled, want: false},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: true},
		{name: "permission denied", err: errors.New("PERMISSION_DENIED: no access"), want: false},
		{name: "invalid params", err: errors.New("INVALID_PARAMS: bad input"), want: false},
		{name: "rate limit", err: errors.New("status=429"), want: true},
		{name: "gateway timeout", err: errors.New("status=504"), want: true},
		{name: "other", err: errors.New("unknown error"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRetryEinoRuntimeError(tt.err)
			if got != tt.want {
				t.Fatalf("shouldRetryEinoRuntimeError()=%v want=%v", got, tt.want)
			}
		})
	}
}

func TestBuildEinoInputMessagesKeepsToolAsAssistantRole(t *testing.T) {
	msgs := buildEinoInputMessages([]*aiMessageSnapshot{
		{Role: assistantRoleTool, Content: `{"title":"cleanup"}`},
	})
	if len(msgs) != 1 {
		t.Fatalf("unexpected messages size: %d", len(msgs))
	}
	if msgs[0].Role != schema.Assistant {
		t.Fatalf("unexpected role: %s", msgs[0].Role)
	}
	if !strings.Contains(msgs[0].Content, "工具结果(JSON):") {
		t.Fatalf("unexpected content: %s", msgs[0].Content)
	}
}
