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

// assistant_runtime_eino_checkpoint.go 提供 Eino checkpoint 的 Redis 存储实现。
// 该能力用于在同一会话内复用运行态上下文，减少重复推理开销。

const defaultEinoCheckpointTTL = 30 * time.Minute

type einoRedisCheckpointStore struct {
	cmd       goredis.Cmdable
	keyPrefix string
	ttl       time.Duration
}

func (s *einoRedisCheckpointStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	if s == nil || s.cmd == nil {
		return nil, false, nil
	}
	// 未命中返回 (nil,false,nil)，调用方可按“无历史 checkpoint”处理。
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
	// 使用统一 TTL 过期策略，避免 checkpoint 长期堆积。
	return s.cmd.Set(ctx, s.wrapKey(checkPointID), checkPoint, s.ttl).Err()
}

func (s *einoRedisCheckpointStore) wrapKey(checkPointID string) string {
	key := strings.TrimSpace(checkPointID)
	if key == "" {
		key = "assistant:unknown"
	}
	return s.keyPrefix + key
}

// newEinoCheckpointStore 根据 AI 配置构造 checkpoint store。
// 返回 nil 表示显式关闭或运行环境不具备 Redis 能力。
func (s *AssistantService) newEinoCheckpointStore(aiCfg *config.AIConfig) adk.CheckPointStore {
	if s == nil || s.repo == nil {
		return nil
	}
	ttl, enabled := resolveEinoCheckpointTTL(aiCfg)
	if !enabled {
		// 显式禁用 checkpoint。
		return nil
	}

	rdb := s.repo.GetRedisCmd()
	if rdb == nil {
		// 运行环境无 Redis 时降级为无 checkpoint 模式。
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

	return &einoRedisCheckpointStore{
		cmd:       rdb,
		keyPrefix: prefix,
		ttl:       ttl,
	}
}

// resolveEinoCheckpointTTL 约定：
// - ttl < 0: 禁用 checkpoint；
// - ttl = 0: 使用默认 30 分钟；
// - ttl > 0: 使用配置值（秒）。
func resolveEinoCheckpointTTL(aiCfg *config.AIConfig) (time.Duration, bool) {
	if aiCfg == nil {
		return defaultEinoCheckpointTTL, true
	}
	if aiCfg.Eino.CheckpointTTLSeconds < 0 {
		return 0, false
	}
	if aiCfg.Eino.CheckpointTTLSeconds == 0 {
		return defaultEinoCheckpointTTL, true
	}
	return time.Duration(aiCfg.Eino.CheckpointTTLSeconds) * time.Second, true
}
