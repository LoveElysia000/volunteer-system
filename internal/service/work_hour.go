package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

type WorkHourService struct {
	Service
}

var errWorkHourIdempotentHit = errors.New("work hour idempotent hit")

func NewWorkHourService(ctx context.Context, c *app.RequestContext) *WorkHourService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &WorkHourService{
		Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
}

// WorkHourLogList 工时流水查询
func (s *WorkHourService) WorkHourLogList(req *api.WorkHourLogListRequest) (*api.WorkHourLogListResponse, error) {
	resp := &api.WorkHourLogListResponse{
		Total: 0,
		List:  []*api.WorkHourLogItem{},
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	queryMap := make(map[string]any)
	// 志愿者身份：仅允许查询自己的工时
	volunteer, err := s.repo.FindVolunteerByAccountID(s.repo.DB, userID)
	if err != nil {
		log.Error("工时流水查询失败: 查询志愿者信息异常: %v, user_id=%d", err, userID)
		return nil, err
	}

	if volunteer == nil {
		log.Warn("工时流水查询失败: 志愿者信息不存在, user_id=%d", userID)
		return nil, errors.New("查找不到志愿者")
	}
	queryMap["volunteer_id = ?"] = volunteer.ID

	if req.ActivityId > 0 {
		queryMap["activity_id = ?"] = req.ActivityId
	}
	if req.SignupId > 0 {
		queryMap["signup_id = ?"] = req.SignupId
	}
	if req.OperationType > 0 {
		queryMap["operation_type = ?"] = req.OperationType
	}

	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	logs, total, err := s.repo.ListWorkHourLogs(s.repo.DB, queryMap, pageSize, offset)
	if err != nil {
		return nil, err
	}

	if total == 0 {
		return resp, nil
	}

	for _, item := range logs {
		resp.List = append(resp.List, &api.WorkHourLogItem{
			Id:                 item.ID,
			VolunteerId:        item.VolunteerID,
			ActivityId:         item.ActivityID,
			SignupId:           item.SignupID,
			OperationType:      item.OperationType,
			HoursDelta:         item.HoursDelta,
			ServiceCountDelta:  item.ServiceCountDelta,
			BeforeTotalHours:   item.BeforeTotalHours,
			AfterTotalHours:    item.AfterTotalHours,
			BeforeServiceCount: item.BeforeServiceCount,
			AfterServiceCount:  item.AfterServiceCount,
			WorkHourVersion:    item.WorkHourVersion,
			RefLogId:           item.RefLogID,
			Reason:             item.Reason,
			OperatorId:         item.OperatorID,
			IdempotencyKey:     item.IdempotencyKey,
			CreatedAt:          util.FormatDateTimeOrEmpty(item.CreatedAt),
		})
	}
	resp.Total = int32(total)
	return resp, nil
}

// VoidWorkHour 工时作废
func (s *WorkHourService) VoidWorkHour(req *api.VoidWorkHourRequest) (*api.VoidWorkHourResponse, error) {
	if req.SignupId <= 0 {
		return nil, errors.New("报名ID不能为空")
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, errors.New("作废原因不能为空")
	}
	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey == "" {
		return nil, errors.New("幂等键不能为空")
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	existingLog, err := s.repo.GetWorkHourLogByIdempotencyKey(s.repo.DB, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if existingLog != nil {
		return &api.VoidWorkHourResponse{
			Success:       true,
			WorkHourLogId: existingLog.ID,
		}, nil
	}

	var workHourLogID int64
	var idempotentLogID int64
	err = s.withTransaction(func(tx *gorm.DB) error {
		signup, err := s.repo.GetActivitySignupByIDForUpdate(tx, req.SignupId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("报名记录不存在")
			}
			return err
		}

		activity, err := s.ensureActivityOperableByCurrentOrgWithTx(tx, signup.ActivityID, userID)
		if err != nil {
			return err
		}
		if activity.Status == model.ActivityStatusCanceled {
			return errors.New("已取消活动不允许作废工时")
		}

		if signup.WorkHourStatus != model.WorkHourStatusGranted || signup.LastWorkHourLogID <= 0 {
			return errors.New("当前工时状态不允许作废")
		}

		volunteer, err := s.repo.FindVolunteerByIDForUpdate(tx, signup.VolunteerID)
		if err != nil {
			return err
		}
		beforeHours := volunteer.TotalHours
		beforeCount := int64(volunteer.ServiceCount)
		hoursDelta := -signup.GrantedHours
		afterHours := util.RoundHours(beforeHours + hoursDelta)
		afterCount := beforeCount - 1
		if afterHours < 0 || afterCount < 0 {
			return errors.New("志愿者统计字段异常")
		}

		newVersion := signup.WorkHourVersion + 1
		logItem := &model.WorkHourLog{
			VolunteerID:        signup.VolunteerID,
			ActivityID:         signup.ActivityID,
			SignupID:           signup.ID,
			OperationType:      model.WorkHourOperationVoid,
			HoursDelta:         hoursDelta,
			ServiceCountDelta:  -1,
			BeforeTotalHours:   beforeHours,
			AfterTotalHours:    afterHours,
			BeforeServiceCount: beforeCount,
			AfterServiceCount:  afterCount,
			WorkHourVersion:    newVersion,
			IdempotencyKey:     idempotencyKey,
			RefLogID:           signup.LastWorkHourLogID,
			Reason:             reason,
			OperatorID:         userID,
		}
		if err := s.repo.CreateWorkHourLog(tx, logItem); err != nil {
			if util.IsDuplicateEntryErr(err) {
				exists, queryErr := s.repo.GetWorkHourLogByIdempotencyKey(tx, idempotencyKey)
				if queryErr != nil {
					return queryErr
				}
				if exists != nil {
					idempotentLogID = exists.ID
					return errWorkHourIdempotentHit
				}
			}
			return err
		}

		if err := s.repo.UpdateVolunteer(tx, volunteer.ID, map[string]interface{}{
			"total_hours":   afterHours,
			"service_count": int32(afterCount),
		}); err != nil {
			return err
		}

		if err := s.repo.UpdateActivitySignupByID(tx, signup.ID, map[string]any{
			"work_hour_status":      model.WorkHourStatusVoided,
			"work_hour_version":     newVersion,
			"last_work_hour_log_id": logItem.ID,
			"granted_hours":         0,
			"granted_at":            nil,
		}); err != nil {
			return err
		}

		workHourLogID = logItem.ID
		return nil
	})
	if err != nil {
		if errors.Is(err, errWorkHourIdempotentHit) {
			return &api.VoidWorkHourResponse{
				Success:       true,
				WorkHourLogId: idempotentLogID,
			}, nil
		}
		return nil, err
	}

	return &api.VoidWorkHourResponse{
		Success:       true,
		WorkHourLogId: workHourLogID,
	}, nil
}

// RecalculateWorkHour 工时重算
func (s *WorkHourService) RecalculateWorkHour(req *api.RecalculateWorkHourRequest) (*api.RecalculateWorkHourResponse, error) {
	if req.SignupId <= 0 {
		return nil, errors.New("报名ID不能为空")
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, errors.New("重算原因不能为空")
	}
	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey == "" {
		return nil, errors.New("幂等键不能为空")
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	existingLog, err := s.repo.GetWorkHourLogByIdempotencyKey(s.repo.DB, idempotencyKey)
	if err != nil {
		return nil, err
	}
	if existingLog != nil {
		signup, signupErr := s.repo.GetActivitySignupByID(s.repo.DB, req.SignupId)
		if signupErr == nil {
			return &api.RecalculateWorkHourResponse{
				Success:       true,
				WorkHourLogId: existingLog.ID,
				GrantedHours:  signup.GrantedHours,
			}, nil
		}
		return &api.RecalculateWorkHourResponse{
			Success:       true,
			WorkHourLogId: existingLog.ID,
		}, nil
	}

	var workHourLogID int64
	var grantedHours float64
	var idempotentLogID int64
	err = s.withTransaction(func(tx *gorm.DB) error {
		signup, err := s.repo.GetActivitySignupByIDForUpdate(tx, req.SignupId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("报名记录不存在")
			}
			return err
		}

		activity, err := s.ensureActivityOperableByCurrentOrgWithTx(tx, signup.ActivityID, userID)
		if err != nil {
			return err
		}
		if signup.CheckInStatus != model.ActivityCheckInDone || signup.CheckOutStatus != model.ActivityCheckOutDone ||
			signup.CheckInTime == nil || signup.CheckOutTime == nil {
			return errors.New("未完成签到签退，不允许重算工时")
		}
		if activity.Status == model.ActivityStatusCanceled {
			return errors.New("已取消活动不允许重算工时")
		}

		targetHours := req.Hours
		if targetHours <= 0 {
			targetHours = util.CalcGrantedHours(activity.Duration, *signup.CheckInTime, *signup.CheckOutTime)
		}
		targetHours = util.RoundHours(targetHours)
		if targetHours < 0 {
			return errors.New("重算工时不能为负数")
		}

		currentHours := signup.GrantedHours
		hoursDelta := util.RoundHours(targetHours - currentHours)
		serviceDelta := int64(0)
		if signup.WorkHourStatus != model.WorkHourStatusGranted {
			serviceDelta = 1
		}

		// 无变化时直接返回成功，避免无意义的版本递增
		if hoursDelta == 0 && serviceDelta == 0 {
			workHourLogID = signup.LastWorkHourLogID
			grantedHours = signup.GrantedHours
			return nil
		}

		volunteer, err := s.repo.FindVolunteerByIDForUpdate(tx, signup.VolunteerID)
		if err != nil {
			return err
		}
		beforeHours := volunteer.TotalHours
		beforeCount := int64(volunteer.ServiceCount)
		afterHours := util.RoundHours(beforeHours + hoursDelta)
		afterCount := beforeCount + serviceDelta
		if afterHours < 0 || afterCount < 0 {
			return errors.New("志愿者统计字段异常")
		}

		newVersion := signup.WorkHourVersion + 1
		logItem := &model.WorkHourLog{
			VolunteerID:        signup.VolunteerID,
			ActivityID:         signup.ActivityID,
			SignupID:           signup.ID,
			OperationType:      model.WorkHourOperationRegrant,
			HoursDelta:         hoursDelta,
			ServiceCountDelta:  serviceDelta,
			BeforeTotalHours:   beforeHours,
			AfterTotalHours:    afterHours,
			BeforeServiceCount: beforeCount,
			AfterServiceCount:  afterCount,
			WorkHourVersion:    newVersion,
			IdempotencyKey:     idempotencyKey,
			RefLogID:           signup.LastWorkHourLogID,
			Reason:             reason,
			OperatorID:         userID,
		}
		if err := s.repo.CreateWorkHourLog(tx, logItem); err != nil {
			if util.IsDuplicateEntryErr(err) {
				exists, queryErr := s.repo.GetWorkHourLogByIdempotencyKey(tx, idempotencyKey)
				if queryErr != nil {
					return queryErr
				}
				if exists != nil {
					idempotentLogID = exists.ID
					return errWorkHourIdempotentHit
				}
			}
			return err
		}

		if err := s.repo.UpdateVolunteer(tx, volunteer.ID, map[string]interface{}{
			"total_hours":   afterHours,
			"service_count": int32(afterCount),
		}); err != nil {
			return err
		}

		now := time.Now()
		if err := s.repo.UpdateActivitySignupByID(tx, signup.ID, map[string]any{
			"work_hour_status":      model.WorkHourStatusGranted,
			"work_hour_version":     newVersion,
			"last_work_hour_log_id": logItem.ID,
			"granted_hours":         targetHours,
			"granted_at":            now,
		}); err != nil {
			return err
		}

		workHourLogID = logItem.ID
		grantedHours = targetHours
		return nil
	})
	if err != nil {
		if errors.Is(err, errWorkHourIdempotentHit) {
			signup, signupErr := s.repo.GetActivitySignupByID(s.repo.DB, req.SignupId)
			if signupErr == nil {
				return &api.RecalculateWorkHourResponse{
					Success:       true,
					WorkHourLogId: idempotentLogID,
					GrantedHours:  signup.GrantedHours,
				}, nil
			}
			return &api.RecalculateWorkHourResponse{
				Success:       true,
				WorkHourLogId: idempotentLogID,
			}, nil
		}
		return nil, err
	}

	return &api.RecalculateWorkHourResponse{
		Success:       true,
		WorkHourLogId: workHourLogID,
		GrantedHours:  grantedHours,
	}, nil
}

func (s *WorkHourService) ensureActivityOperableByCurrentOrgWithTx(tx *gorm.DB, activityID, accountID int64) (*model.Activity, error) {
	activity, err := s.repo.GetActivityByID(tx, activityID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		return nil, err
	}

	org, err := s.repo.GetOrganizationByAccountID(tx, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("组织信息不存在")
		}
		return nil, err
	}

	if activity.OrgID != org.ID {
		return nil, errors.New("无权操作此活动")
	}
	return activity, nil
}
