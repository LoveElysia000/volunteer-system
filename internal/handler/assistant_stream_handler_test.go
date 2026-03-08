package handler

import (
	"bytes"
	"strings"
	"testing"
)

func TestAssistantChatStream_WriteSSEEvent(t *testing.T) {
	buf := &bytes.Buffer{}
	err := writeSSEEvent(buf, "delta", map[string]any{"text": "hello"})
	if err != nil {
		t.Fatalf("writeSSEEvent error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "event: delta") {
		t.Fatalf("missing event field: %s", output)
	}
	if !strings.Contains(output, "data:") {
		t.Fatalf("missing data field: %s", output)
	}
	if !strings.HasSuffix(output, "\n\n") {
		t.Fatalf("sse frame should end with blank line: %q", output)
	}
}
