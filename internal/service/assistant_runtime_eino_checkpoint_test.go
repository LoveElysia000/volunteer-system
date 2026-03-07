package service

import (
	"testing"
	"time"

	"volunteer-system/config"
)

func TestResolveEinoCheckpointTTL(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.AIConfig
		wantTTL time.Duration
		enabled bool
	}{
		{name: "nil config", cfg: nil, wantTTL: defaultEinoCheckpointTTL, enabled: true},
		{name: "negative disables", cfg: &config.AIConfig{Eino: config.AIEinoConfig{CheckpointTTLSeconds: -1}}, wantTTL: 0, enabled: false},
		{name: "zero uses default", cfg: &config.AIConfig{Eino: config.AIEinoConfig{CheckpointTTLSeconds: 0}}, wantTTL: defaultEinoCheckpointTTL, enabled: true},
		{name: "positive value", cfg: &config.AIConfig{Eino: config.AIEinoConfig{CheckpointTTLSeconds: 600}}, wantTTL: 600 * time.Second, enabled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTTL, gotEnabled := resolveEinoCheckpointTTL(tt.cfg)
			if gotEnabled != tt.enabled {
				t.Fatalf("enabled=%v want=%v", gotEnabled, tt.enabled)
			}
			if gotTTL != tt.wantTTL {
				t.Fatalf("ttl=%s want=%s", gotTTL, tt.wantTTL)
			}
		})
	}
}
