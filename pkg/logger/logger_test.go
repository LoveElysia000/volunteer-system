package logger

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func resetLoggerStateForTest() {
	instance = newDefaultLogger()
	once = sync.Once{}
	initErr = nil

	rotationMu.Lock()
	rotationCfg = rotationConfig{
		enabled:   false,
		maxSizeMB: 100,
		maxFiles:  3,
		maxAgeDay: 28,
		compress:  true,
	}
	rotationMu.Unlock()
}

func TestLoggerOutputFormat(t *testing.T) {
	resetLoggerStateForTest()

	logFile := filepath.Join(t.TempDir(), "app.log")
	if err := Init("debug", false, logFile); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	log := GetLogger()
	log.Info("test info id=%d", 123)
	log.Warn("test warn name=%s", "alice")
	log.Error("test error: %v", errors.New("boom"))

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	out := string(data)

	t.Logf("logger output:\n%s", out)

	if !strings.Contains(out, "level=[INFO]") {
		t.Fatalf("expected INFO level in output, got: %s", out)
	}
	if !strings.Contains(out, "level=[WARN]") {
		t.Fatalf("expected WARN level in output, got: %s", out)
	}
	if !strings.Contains(out, "level=[ERROR]") {
		t.Fatalf("expected ERROR level in output, got: %s", out)
	}
	if !strings.Contains(out, "source=") {
		t.Fatalf("expected source field in output, got: %s", out)
	}
	if !strings.Contains(out, "test info id=123") ||
		!strings.Contains(out, "test warn name=alice") ||
		!strings.Contains(out, "test error: boom") {
		t.Fatalf("expected test messages in output, got: %s", out)
	}
}

