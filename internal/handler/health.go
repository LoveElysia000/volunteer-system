package handler

import (
	"context"
	"time"

	"volunteer-system/pkg/database/mysql"
	"volunteer-system/pkg/database/redis"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

const healthCheckTimeout = 2 * time.Second

// Livez 存活探针：只要 HTTP 服务能响应即返回 200，不检查外部依赖。
// 用于 Kubernetes liveness probe。
func Livez(_ context.Context, c *app.RequestContext) {
	c.JSON(consts.StatusOK, map[string]string{"status": "ok"})
}

// Healthz 就绪探针：检查 MySQL 与 Redis 的连通性。
// 任一依赖不可达时返回 503，用于 Kubernetes readiness probe，
// 摘除流量但保持容器存活，等待依赖恢复。
func Healthz(ctx context.Context, c *app.RequestContext) {
	checks := make(map[string]string, 2)
	healthy := true

	// MySQL
	mysqlCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()
	if db := mysql.GetDB(); db != nil {
		sqlDB, err := db.DB()
		if err == nil {
			err = sqlDB.PingContext(mysqlCtx)
		}
		if err != nil {
			checks["mysql"] = "unavailable"
			healthy = false
		} else {
			checks["mysql"] = "ok"
		}
	} else {
		checks["mysql"] = "not-initialized"
		healthy = false
	}

	// Redis
	if client := redis.GetRedis(); client != nil {
		redisCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
		defer cancel()
		if err := client.Ping(redisCtx).Err(); err != nil {
			checks["redis"] = "unavailable"
			healthy = false
		} else {
			checks["redis"] = "ok"
		}
	} else {
		checks["redis"] = "not-initialized"
		healthy = false
	}

	status := consts.StatusOK
	if !healthy {
		status = consts.StatusServiceUnavailable
	}
	c.JSON(status, map[string]interface{}{
		"status": map[bool]string{true: "ok", false: "degraded"}[healthy],
		"checks": checks,
	})
}
