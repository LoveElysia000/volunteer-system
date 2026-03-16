package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
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

const (
	defaultWorkHourPageSize      = 50
	maxWorkHourPageSize          = 100
	maxWorkHourReasonLength      = 500
	maxWorkHourIdempotencyKeyLen = 128
)

func validateWorkHourMutationInput(
	signupID int64,
	rawReason, rawIdempotencyKey string,
	reasonRequiredMessage, reasonLengthMessage string,
) (string, string, error) {
	if signupID <= 0 {
		return "", "", errors.New("报名ID不能为空")
	}

	reason := strings.TrimSpace(rawReason)
	if reason == "" {
		return "", "", errors.New(reasonRequiredMessage)
	}
	if utf8.RuneCountInString(reason) > maxWorkHourReasonLength {
		return "", "", errors.New(reasonLengthMessage)
	}

	idempotencyKey := strings.TrimSpace(rawIdempotencyKey)
	if idempotencyKey == "" {
		return "", "", errors.New("幂等键不能为空")
	}
	if utf8.RuneCountInString(idempotencyKey) > maxWorkHourIdempotencyKeyLen {
		return "", "", errors.New("幂等键长度不能超过128字符")
	}

	return reason, idempotencyKey, nil
}

func resolveRecalculateTargetHours(rawHours float64, signup *model.ActivitySignup, activity *model.Activity) (float64, error) {
	targetHours := rawHours
	if targetHours <= 0 {
		if signup == nil || signup.CheckInTime == nil || signup.CheckOutTime == nil {
			return 0, errors.New("未完成签到签退，不允许重算工时")
		}
		if activity == nil {
			return 0, errors.New("活动不存在")
		}
		targetHours = util.CalcGrantedHours(activity.Duration, *signup.CheckInTime, *signup.CheckOutTime)
	}
	targetHours = util.RoundHours(targetHours)
	if targetHours < 0 {
		return 0, errors.New("重算工时不能为负数")
	}
	return targetHours, nil
}

func sameTimePtr(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func attendanceDone(signup *model.ActivitySignup) bool {
	return signup != nil &&
		signup.CheckInStatus == model.ActivityCheckInDone &&
		signup.CheckOutStatus == model.ActivityCheckOutDone &&
		signup.CheckInTime != nil &&
		signup.CheckOutTime != nil
}

func sameWorkHourSnapshot(current, base *model.ActivitySignup, withAttendance bool) bool {
	if current == nil || base == nil {
		return false
	}
	if current.ActivityID != base.ActivityID ||
		current.VolunteerID != base.VolunteerID ||
		current.WorkHourStatus != base.WorkHourStatus ||
		current.WorkHourVersion != base.WorkHourVersion ||
		current.LastWorkHourLogID != base.LastWorkHourLogID ||
		util.RoundHours(current.GrantedHours) != util.RoundHours(base.GrantedHours) {
		return false
	}
	if !withAttendance {
		return true
	}
	return current.CheckInStatus == base.CheckInStatus &&
		current.CheckOutStatus == base.CheckOutStatus &&
		sameTimePtr(current.CheckInTime, base.CheckInTime) &&
		sameTimePtr(current.CheckOutTime, base.CheckOutTime)
}

func validateVoidEligibility(signup *model.ActivitySignup) error {
	if signup == nil {
		return errors.New("报名记录不存在")
	}
	if signup.WorkHourStatus != model.WorkHourStatusGranted || signup.LastWorkHourLogID <= 0 {
		return errors.New("当前工时状态不允许作废")
	}
	return nil
}

func validateRegrantStatus(status int32) error {
	switch status {
	case model.WorkHourStatusPending, model.WorkHourStatusGranted, model.WorkHourStatusVoided:
		return nil
	default:
		return errors.New("工时状态无效")
	}
}

func validateWorkHourChain(signup *model.ActivitySignup, lastLog *model.WorkHourLog) error {
	if signup == nil {
		return errors.New("报名记录不存在")
	}

	if signup.LastWorkHourLogID <= 0 {
		if signup.WorkHourStatus == model.WorkHourStatusPending && signup.WorkHourVersion == 0 {
			return nil
		}
		return errors.New("工时流水链路异常")
	}

	if lastLog == nil {
		return errors.New("工时流水链路异常")
	}
	if lastLog.SignupID != signup.ID ||
		lastLog.VolunteerID != signup.VolunteerID ||
		lastLog.ActivityID != signup.ActivityID {
		return errors.New("工时流水链路异常")
	}
	if lastLog.WorkHourVersion != signup.WorkHourVersion {
		return errors.New("工时流水版本不一致")
	}

	switch signup.WorkHourStatus {
	case model.WorkHourStatusGranted:
		if lastLog.OperationType == model.WorkHourOperationVoid {
			return errors.New("工时状态与流水不一致")
		}
	case model.WorkHourStatusVoided:
		if lastLog.OperationType != model.WorkHourOperationVoid {
			return errors.New("工时状态与流水不一致")
		}
	case model.WorkHourStatusPending:
		return errors.New("工时状态与流水不一致")
	default:
		return errors.New("工时状态无效")
	}
	return nil
}

func (s *WorkHourService) loadWorkHourMutationBase(
	tx *gorm.DB,
	signupID, accountID int64,
) (*model.ActivitySignup, *model.Activity, *model.WorkHourLog, error) {
	signup, err := s.repo.GetActivitySignupByID(tx, signupID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, errors.New("报名记录不存在")
		}
		return nil, nil, nil, err
	}

	activity, err := s.ensureActivityOperableByCurrentOrgWithTx(tx, signup.ActivityID, accountID)
	if err != nil {
		return nil, nil, nil, err
	}

	var lastLog *model.WorkHourLog
	if signup.LastWorkHourLogID > 0 {
		lastLog, err = s.repo.GetWorkHourLogByID(tx, signup.LastWorkHourLogID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil, nil, errors.New("工时流水链路异常")
			}
			return nil, nil, nil, err
		}
	}

	if err := validateWorkHourChain(signup, lastLog); err != nil {
		return nil, nil, nil, err
	}

	return signup, activity, lastLog, nil
}

func (s *WorkHourService) insertWorkHourLogIdempotent(
	tx *gorm.DB,
	logItem *model.WorkHourLog,
	isSameRequest func(*model.WorkHourLog) bool,
) (int64, bool, error) {
	if err := s.repo.CreateWorkHourLog(tx, logItem); err != nil {
		if !util.IsDuplicateEntryErr(err) {
			return 0, false, err
		}

		exists, queryErr := s.repo.GetWorkHourLogByIdempotencyKey(tx, logItem.IdempotencyKey)
		if queryErr != nil {
			return 0, false, queryErr
		}
		if exists == nil {
			return 0, false, err
		}
		if !isSameRequest(exists) {
			return 0, false, errors.New("幂等键已被其他请求占用")
		}
		return exists.ID, true, nil
	}
	return logItem.ID, false, nil
}

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

// WorkHourLogList 工时流水查询（志愿者查自己的流水；组织查本组织活动的流水）
func (s *WorkHourService) WorkHourLogList(req *api.WorkHourLogListRequest) (*api.WorkHourLogListResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	resp := &api.WorkHourLogListResponse{
		Total: 0,
		List:  []*api.WorkHourLogItem{},
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = defaultWorkHourPageSize
	}
	if req.PageSize > maxWorkHourPageSize {
		req.PageSize = maxWorkHourPageSize
	}

	if req.OperationType > 0 &&
		req.OperationType != model.WorkHourOperationGrant &&
		req.OperationType != model.WorkHourOperationVoid &&
		req.OperationType != model.WorkHourOperationRegrant {
		return nil, errors.New("工时操作类型无效")
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	account, err := s.repo.FindByID(s.repo.DB, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("账号不存在")
		}
		return nil, err
	}

	queryMap := make(map[string]any)
	activityFilterLocked := false

	switch account.IdentityType {
	case model.RegisterTypeVolunteerCode:
		// 志愿者：只能看自己的流水
		volunteer, err := s.repo.FindVolunteerByAccountID(s.repo.DB, userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("志愿者信息不存在")
			}
			log.Error("工时流水查询失败: 查询志愿者信息异常: %v, user_id=%d", err, userID)
			return nil, err
		}
		if volunteer == nil {
			return nil, errors.New("志愿者信息不存在")
		}
		queryMap["volunteer_id = ?"] = volunteer.ID

	case model.RegisterTypeOrganizationCode:
		// 组织账号：按 RBAC 作用域读取有权限的组织流水。
		if req.ActivityId > 0 {
			activity, err := s.repo.GetActivityByID(s.repo.DB, req.ActivityId)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, errors.New("活动不存在")
				}
				return nil, err
			}
			if err := s.requireOrgPermission(
				userID,
				activity.OrgID,
				model.PermissionResourceOrganization,
				model.PermissionActionManage,
			); err != nil {
				return nil, errors.New("无权查看该活动工时流水")
			}
			queryMap["activity_id = ?"] = req.ActivityId
			activityFilterLocked = true
		} else {
			hasGlobalManage, err := s.hasPermissionByScope(
				userID,
				model.RBACScopeGlobal,
				0,
				model.PermissionResourceOrganization,
				model.PermissionActionManage,
			)
			if err != nil {
				return nil, err
			}
			if !hasGlobalManage {
				orgIDs, err := s.repo.ListOrgScopeIDsByPermission(
					s.repo.DB,
					userID,
					model.PermissionResourceOrganization,
					model.PermissionActionManage,
					0,
				)
				if err != nil {
					return nil, err
				}
				if len(orgIDs) == 0 {
					return nil, errors.New("无权查看工时流水")
				}
				queryMap["activity_id IN (SELECT id FROM activities WHERE org_id IN ?)"] = orgIDs
			}
		}

	default:
		return nil, errors.New("账号身份无效")
	}

	if req.ActivityId > 0 && !activityFilterLocked {
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
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	reason, idempotencyKey, err := validateWorkHourMutationInput(
		req.SignupId,
		req.Reason,
		req.IdempotencyKey,
		"作废原因不能为空",
		"作废原因长度不能超过500字符",
	)
	if err != nil {
		return nil, err
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}
	isSameIdempotentRequest := func(logEntry *model.WorkHourLog) bool {
		return logEntry != nil &&
			logEntry.SignupID == req.SignupId &&
			logEntry.OperationType == model.WorkHourOperationVoid &&
			logEntry.OperatorID == userID &&
			logEntry.Reason == reason
	}

	existingLog, err := s.repo.GetWorkHourLogByIdempotencyKey(s.repo.DB, idempotencyKey)
	if err != nil {
		return nil, err
	}
	// 幂等键命中时，必须与当前请求完全一致。
	if existingLog != nil {
		if !isSameIdempotentRequest(existingLog) {
			return nil, errors.New("幂等键已被其他请求占用")
		}
		return &api.VoidWorkHourResponse{
			Success:       true,
			WorkHourLogId: existingLog.ID,
		}, nil
	}

	preSignup, preActivity, preLastLog, err := s.loadWorkHourMutationBase(s.repo.DB, req.SignupId, userID)
	if err != nil {
		return nil, err
	}
	if preActivity.Status == model.ActivityStatusCanceled {
		return nil, errors.New("已取消活动不允许作废工时")
	}
	if err := validateVoidEligibility(preSignup); err != nil {
		return nil, err
	}

	var workHourLogID int64
	var idempotentLogID int64
	err = s.withTransaction(func(tx *gorm.DB) error {
		activity, err := s.repo.GetActivityByIDForUpdate(tx, preSignup.ActivityID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("活动不存在")
			}
			return err
		}
		if activity.Status == model.ActivityStatusCanceled {
			return errors.New("已取消活动不允许作废工时")
		}

		signup, err := s.repo.GetActivitySignupByIDForUpdate(tx, req.SignupId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("报名记录不存在")
			}
			return err
		}
		if !sameWorkHourSnapshot(signup, preSignup, false) {
			return errors.New("报名工时状态已变化，请刷新后重试")
		}
		if err := validateVoidEligibility(signup); err != nil {
			return err
		}
		refLogID := int64(0)
		if preLastLog != nil {
			refLogID = preLastLog.ID
		}
		hoursDelta := -signup.GrantedHours
		serviceDelta := int64(-1)

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
			OperationType:      model.WorkHourOperationVoid,
			HoursDelta:         hoursDelta,
			ServiceCountDelta:  serviceDelta,
			BeforeTotalHours:   beforeHours,
			AfterTotalHours:    afterHours,
			BeforeServiceCount: beforeCount,
			AfterServiceCount:  afterCount,
			WorkHourVersion:    newVersion,
			IdempotencyKey:     idempotencyKey,
			RefLogID:           refLogID,
			Reason:             reason,
			OperatorID:         userID,
		}
		logID, idempotent, err := s.insertWorkHourLogIdempotent(tx, logItem, isSameIdempotentRequest)
		if err != nil {
			return err
		}
		if idempotent {
			idempotentLogID = logID
			return errWorkHourIdempotentHit
		}

		levelID, err := s.repo.ResolveLevelIDByTotalHours(tx, afterHours)
		if err != nil {
			return err
		}

		if err := s.repo.UpdateVolunteer(tx, volunteer.ID, map[string]interface{}{
			"total_hours":   afterHours,
			"service_count": int32(afterCount),
			"level_id":      levelID,
		}); err != nil {
			return err
		}

		if err := s.repo.CreateRecord(tx, &model.Record{
			VolunteerID: signup.VolunteerID,
			Type:        "HOUR",
			Amount:      hoursDelta,
			CreateTime:  logItem.CreatedAt,
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
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	reason, idempotencyKey, err := validateWorkHourMutationInput(
		req.SignupId,
		req.Reason,
		req.IdempotencyKey,
		"重算原因不能为空",
		"重算原因长度不能超过500字符",
	)
	if err != nil {
		return nil, err
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}
	isSameIdempotentRequest := func(logEntry *model.WorkHourLog) bool {
		return logEntry != nil &&
			logEntry.SignupID == req.SignupId &&
			logEntry.OperationType == model.WorkHourOperationRegrant &&
			logEntry.OperatorID == userID &&
			logEntry.Reason == reason
	}

	existingLog, err := s.repo.GetWorkHourLogByIdempotencyKey(s.repo.DB, idempotencyKey)
	if err != nil {
		return nil, err
	}
	// 幂等键命中时，必须与当前请求完全一致。
	if existingLog != nil {
		if !isSameIdempotentRequest(existingLog) {
			return nil, errors.New("幂等键已被其他请求占用")
		}
		signup, signupErr := s.repo.GetActivitySignupByID(s.repo.DB, req.SignupId)
		if signupErr == nil {
			if signup.LastWorkHourLogID != existingLog.ID {
				return nil, errors.New("幂等键已被其他请求占用")
			}
			var activity *model.Activity
			if req.Hours <= 0 {
				activity, err = s.repo.GetActivityByID(s.repo.DB, signup.ActivityID)
				if err != nil {
					return nil, errors.New("幂等键已被其他请求占用")
				}
			}
			targetHours, targetErr := resolveRecalculateTargetHours(req.Hours, signup, activity)
			if targetErr != nil || util.RoundHours(signup.GrantedHours) != targetHours {
				return nil, errors.New("幂等键已被其他请求占用")
			}
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

	preSignup, preActivity, preLastLog, err := s.loadWorkHourMutationBase(s.repo.DB, req.SignupId, userID)
	if err != nil {
		return nil, err
	}
	if preActivity.Status == model.ActivityStatusCanceled {
		return nil, errors.New("已取消活动不允许重算工时")
	}
	if !attendanceDone(preSignup) {
		return nil, errors.New("未完成签到签退，不允许重算工时")
	}

	var workHourLogID int64
	var grantedHours float64
	var idempotentLogID int64
	err = s.withTransaction(func(tx *gorm.DB) error {
		activity, err := s.repo.GetActivityByIDForUpdate(tx, preSignup.ActivityID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("活动不存在")
			}
			return err
		}
		if activity.Status == model.ActivityStatusCanceled {
			return errors.New("已取消活动不允许重算工时")
		}

		signup, err := s.repo.GetActivitySignupByIDForUpdate(tx, req.SignupId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("报名记录不存在")
			}
			return err
		}
		if !sameWorkHourSnapshot(signup, preSignup, true) {
			return errors.New("报名工时状态已变化，请刷新后重试")
		}
		if err := validateRegrantStatus(signup.WorkHourStatus); err != nil {
			return err
		}

		targetHours, err := resolveRecalculateTargetHours(req.Hours, signup, activity)
		if err != nil {
			return err
		}
		serviceDelta := int64(0)
		if signup.WorkHourStatus != model.WorkHourStatusGranted {
			serviceDelta = 1
		}
		hoursDelta := util.RoundHours(targetHours - signup.GrantedHours)
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
		refLogID := int64(0)
		if preLastLog != nil {
			refLogID = preLastLog.ID
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
			RefLogID:           refLogID,
			Reason:             reason,
			OperatorID:         userID,
		}
		logID, idempotent, err := s.insertWorkHourLogIdempotent(tx, logItem, isSameIdempotentRequest)
		if err != nil {
			return err
		}
		if idempotent {
			idempotentLogID = logID
			return errWorkHourIdempotentHit
		}

		levelID, err := s.repo.ResolveLevelIDByTotalHours(tx, afterHours)
		if err != nil {
			return err
		}

		if err := s.repo.UpdateVolunteer(tx, volunteer.ID, map[string]interface{}{
			"total_hours":   afterHours,
			"service_count": int32(afterCount),
			"level_id":      levelID,
		}); err != nil {
			return err
		}

		if err := s.repo.CreateRecord(tx, &model.Record{
			VolunteerID: signup.VolunteerID,
			Type:        "HOUR",
			Amount:      hoursDelta,
			CreateTime:  logItem.CreatedAt,
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

	if err := s.requireOrgPermission(
		accountID,
		activity.OrgID,
		model.PermissionResourceOrganization,
		model.PermissionActionManage,
	); err != nil {
		return nil, err
	}
	return activity, nil
}
