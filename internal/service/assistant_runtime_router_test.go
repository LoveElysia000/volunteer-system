package service

import (
	"testing"
)

func TestExpandConfigEnvExpression(t *testing.T) {
	t.Setenv("UNIT_AI_KEY", "abc123")

	if got, ok := expandConfigEnvExpression("${UNIT_AI_KEY:}"); !ok || got != "abc123" {
		t.Fatalf("env expression resolve = (%v, %q), want (true, %q)", ok, got, "abc123")
	}

	if got, ok := expandConfigEnvExpression("${MISSING_ENV:default_value}"); !ok || got != "default_value" {
		t.Fatalf("fallback expression resolve = (%v, %q), want (true, %q)", ok, got, "default_value")
	}

	if got, ok := expandConfigEnvExpression("plain-value"); ok || got != "" {
		t.Fatalf("plain expression resolve = (%v, %q), want (false, %q)", ok, got, "")
	}
}

func TestBuildEinoCheckpointID(t *testing.T) {
	id := buildEinoCheckpointID(123, "req-1")
	if id != "assistant:123:req-1" {
		t.Fatalf("checkpoint id = %q, want %q", id, "assistant:123:req-1")
	}
}
