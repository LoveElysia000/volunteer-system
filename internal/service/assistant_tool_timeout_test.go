package service

import (
	"testing"
	"time"

	"volunteer-system/config"
)

func TestResolveEinoToolTimeout(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.AIConfig
		want time.Duration
	}{
		{name: "nil config", cfg: nil, want: assistantToolDefaultTimeout},
		{name: "zero", cfg: &config.AIConfig{Eino: config.AIEinoConfig{ToolTimeoutMS: 0}}, want: assistantToolDefaultTimeout},
		{name: "min clamp", cfg: &config.AIConfig{Eino: config.AIEinoConfig{ToolTimeoutMS: 100}}, want: assistantToolMinTimeout},
		{name: "normal", cfg: &config.AIConfig{Eino: config.AIEinoConfig{ToolTimeoutMS: 1200}}, want: 1200 * time.Millisecond},
		{name: "max clamp", cfg: &config.AIConfig{Eino: config.AIEinoConfig{ToolTimeoutMS: 60000}}, want: assistantToolMaxTimeout},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveEinoToolTimeout(tt.cfg)
			if got != tt.want {
				t.Fatalf("resolveEinoToolTimeout()=%s want=%s", got, tt.want)
			}
		})
	}
}
