package service

import (
	"context"
	"volunteer-system/internal/repository"

	"github.com/cloudwego/hertz/pkg/app"
)

type Service struct {
	ctx  context.Context
	c    *app.RequestContext
	repo *repository.Repository
}

// NewService 创建新的服务实例
func NewService(ctx context.Context, c *app.RequestContext) *Service {
	return &Service{
		ctx:  ctx,
		c:    c,
		repo: repository.NewRepository(ctx, c),
	}
}
