package service

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
)

type AnalyticsService struct {
	Service
}

func NewAnalyticsService(ctx context.Context, c *app.RequestContext) *AnalyticsService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &AnalyticsService{
		Service: Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
}

func (s *AnalyticsService) OrgFunnelSummary(req *api.OrgFunnelSummaryRequest) (*api.OrgFunnelSummaryResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if req.OrgId <= 0 {
		return nil, errors.New("组织ID不能为空")
	}

	var startTime *time.Time
	if strings.TrimSpace(req.Start) != "" {
		rawStart := strings.TrimSpace(req.Start)
		value, err := util.ParseDateOrDateTime(rawStart)
		if err != nil {
			return nil, errors.New("开始时间格式错误")
		}
		if !strings.Contains(rawStart, " ") {
			value = time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
		}
		startTime = &value
	}
	var endTime *time.Time
	if strings.TrimSpace(req.End) != "" {
		rawEnd := strings.TrimSpace(req.End)
		value, err := util.ParseDateOrDateTime(rawEnd)
		if err != nil {
			return nil, errors.New("结束时间格式错误")
		}
		if !strings.Contains(rawEnd, " ") {
			value = time.Date(value.Year(), value.Month(), value.Day(), 23, 59, 59, 0, value.Location())
		}
		endTime = &value
	}
	if startTime != nil && endTime != nil && endTime.Before(*startTime) {
		return nil, errors.New("结束时间不能早于开始时间")
	}

	operatorAccountID, err := s.currentAccountID()
	if err != nil {
		return nil, err
	}
	if err := s.requireOrgPermission(
		operatorAccountID,
		req.OrgId,
		model.PermissionResourceAnalytics,
		model.PermissionActionOrgRead,
	); err != nil {
		return nil, err
	}

	metrics, err := s.repo.GetOrgFunnelMetrics(s.repo.DB, req.OrgId, startTime, endTime)
	if err != nil {
		return nil, err
	}

	return &api.OrgFunnelSummaryResponse{
		RegistrationCount:            metrics.RegistrationCount,
		MembershipCount:              metrics.MembershipCount,
		SignupCount:                  metrics.SignupCount,
		AttendanceCount:              metrics.AttendanceCount,
		WorkhourCount:                metrics.WorkhourCount,
		RegistrationToMembershipRate: computeRate(metrics.MembershipCount, metrics.RegistrationCount),
		MembershipToSignupRate:       computeRate(metrics.SignupCount, metrics.MembershipCount),
		SignupToAttendanceRate:       computeRate(metrics.AttendanceCount, metrics.SignupCount),
		AttendanceToWorkhourRate:     computeRate(metrics.WorkhourCount, metrics.AttendanceCount),
		Start:                        strings.TrimSpace(req.Start),
		End:                          strings.TrimSpace(req.End),
	}, nil
}

func (s *AnalyticsService) OpsDashboardSummary(req *api.OpsDashboardSummaryRequest) (*api.OpsDashboardSummaryResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if req.OrgId <= 0 {
		return nil, errors.New("组织ID不能为空")
	}

	var startTime *time.Time
	if strings.TrimSpace(req.Start) != "" {
		rawStart := strings.TrimSpace(req.Start)
		value, err := util.ParseDateOrDateTime(rawStart)
		if err != nil {
			return nil, errors.New("开始时间格式错误")
		}
		if !strings.Contains(rawStart, " ") {
			value = time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
		}
		startTime = &value
	}
	var endTime *time.Time
	if strings.TrimSpace(req.End) != "" {
		rawEnd := strings.TrimSpace(req.End)
		value, err := util.ParseDateOrDateTime(rawEnd)
		if err != nil {
			return nil, errors.New("结束时间格式错误")
		}
		if !strings.Contains(rawEnd, " ") {
			value = time.Date(value.Year(), value.Month(), value.Day(), 23, 59, 59, 0, value.Location())
		}
		endTime = &value
	}
	if startTime != nil && endTime != nil && endTime.Before(*startTime) {
		return nil, errors.New("结束时间不能早于开始时间")
	}

	operatorAccountID, err := s.currentAccountID()
	if err != nil {
		return nil, err
	}
	if err := s.requireOrgPermission(
		operatorAccountID,
		req.OrgId,
		model.PermissionResourceAnalytics,
		model.PermissionActionOrgRead,
	); err != nil {
		return nil, err
	}

	metrics, err := s.repo.GetOpsDashboardMetrics(s.repo.DB, req.OrgId, startTime, endTime)
	if err != nil {
		return nil, err
	}

	return &api.OpsDashboardSummaryResponse{
		SignupCount:         metrics.SignupCount,
		ApprovedSignupCount: metrics.ApprovedSignupCount,
		AttendanceCount:     metrics.AttendanceCount,
		AttendanceRate:      computeRate(metrics.AttendanceCount, metrics.ApprovedSignupCount),
		GrantedWorkHours:    roundMetric(metrics.GrantedWorkHours),
		Start:               strings.TrimSpace(req.Start),
		End:                 strings.TrimSpace(req.End),
	}, nil
}

func computeRate(numerator, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	rate := float64(numerator) * 100 / float64(denominator)
	return roundMetric(rate)
}

func roundMetric(value float64) float64 {
	return math.Round(value*100) / 100
}
