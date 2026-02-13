package repository

import (
	"context"
	"volunteer-system/pkg/database/mysql"
	"volunteer-system/pkg/database/redis"

	"github.com/cloudwego/hertz/pkg/app"
	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Repository struct {
	ctx context.Context
	c   *app.RequestContext
	DB  *gorm.DB
	rDB *goredis.Client
}

func (r *Repository) SetContext(ctx *context.Context) {
	r.ctx = *ctx
}

func NewRepository(ctx context.Context, c *app.RequestContext) *Repository {
	db := mysql.GetDBWithContext(ctx, c).Debug()
	rdb := redis.GetRedis()

	return &Repository{
		ctx: ctx,
		c:   c,
		DB:  db,
		rDB: rdb,
	}
}

func (r *Repository) GetRedisCmd() goredis.Cmdable {
	return r.rDB
}
