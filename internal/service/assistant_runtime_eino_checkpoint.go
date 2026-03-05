package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"volunteer-system/config"

	"github.com/cloudwego/eino/adk"
	goredis "github.com/redis/go-redis/v9"
)

type einoRedisCheckpointStore struct {
	cmd       goredis.Cmdable
	keyPrefix string
	ttl       time.Duration
}

func (s *einoRedisCheckpointStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	if s == nil || s.cmd == nil {
		return nil, false, nil
	}
	value, err := s.cmd.Get(ctx, s.wrapKey(checkPointID)).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return value, true, nil
}

func (s *einoRedisCheckpointStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	if s == nil || s.cmd == nil {
		return errors.New("checkpoint redis store unavailable")
	}
	return s.cmd.Set(ctx, s.wrapKey(checkPointID), checkPoint, s.ttl).Err()
}

func (s *einoRedisCheckpointStore) wrapKey(checkPointID string) string {
	key := strings.TrimSpace(checkPointID)
	if key == "" {
		key = "assistant:unknown"
	}
	return s.keyPrefix + key
}

func (s *AssistantService) newEinoCheckpointStore(aiCfg *config.AIConfig) adk.CheckPointStore {
	if s == nil || s.repo == nil {
		return nil
	}

	rdb := s.repo.GetRedisCmd()
	if rdb == nil {
		return nil
	}

	prefix := ""
	appCfg := config.GetConfig()
	if appCfg != nil && appCfg.Redis != nil {
		prefix = strings.TrimSpace(appCfg.Redis.KeyPrefix)
	}
	if prefix != "" && !strings.HasSuffix(prefix, ":") {
		prefix += ":"
	}

	ttl := 30 * time.Minute
	if aiCfg != nil && aiCfg.Eino.CheckpointTTLSeconds > 0 {
		ttl = time.Duration(aiCfg.Eino.CheckpointTTLSeconds) * time.Second
	}

	return &einoRedisCheckpointStore{
		cmd:       rdb,
		keyPrefix: prefix,
		ttl:       ttl,
	}
}
