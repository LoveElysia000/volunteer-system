package service

import (
	"context"
	"errors"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
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

func (s *Service) currentAccountID() (int64, error) {
	return middleware.GetAccountIDInt(s.c)
}

func (s *Service) currentVolunteer() (*model.Volunteer, int64, error) {
	accountID, err := s.currentAccountID()
	if err != nil {
		return nil, 0, err
	}

	volunteer, err := s.repo.FindVolunteerByAccountID(s.repo.DB, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, accountID, errors.New("志愿者不存在")
		}
		return nil, accountID, err
	}
	if volunteer == nil {
		return nil, accountID, errors.New("志愿者不存在")
	}
	return volunteer, accountID, nil
}
