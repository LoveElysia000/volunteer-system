package service

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
	"volunteer-system/pkg/util"

	"gorm.io/gorm"
)

func collectRegisteredActivityIDs(signups []*model.ActivitySignup) []int64 {
	if len(signups) == 0 {
		return []int64{}
	}

	ids := make([]int64, 0, len(signups))
	for _, signup := range signups {
		if signup == nil || signup.ActivityID <= 0 {
			continue
		}
		switch signup.Status {
		case model.ActivitySignupStatusPending, model.ActivitySignupStatusSuccess:
			ids = append(ids, signup.ActivityID)
		}
	}
	return util.UniquePositiveInt64(ids)
}

func intersectActivityIDs(left, right []int64) []int64 {
	if len(left) == 0 || len(right) == 0 {
		return []int64{}
	}

	rightSet := make(map[int64]struct{}, len(right))
	for _, id := range util.UniquePositiveInt64(right) {
		rightSet[id] = struct{}{}
	}

	intersection := make([]int64, 0, len(left))
	for _, id := range util.UniquePositiveInt64(left) {
		if _, ok := rightSet[id]; ok {
			intersection = append(intersection, id)
		}
	}
	return intersection
}

func mergeActivityIDConstraint(filters map[string]any, ids []int64) bool {
	if len(ids) == 0 {
		return false
	}

	normalized := util.UniquePositiveInt64(ids)
	if len(normalized) == 0 {
		return false
	}

	if existingIDs, ok := filters["act.id IN ?"].([]int64); ok {
		normalized = intersectActivityIDs(existingIDs, normalized)
		if len(normalized) == 0 {
			return false
		}
	}

	filters["act.id IN ?"] = normalized
	return true
}

// buildActivityListFilterMap normalizes ActivityList request filters for repository query.
func buildActivityListFilterMap(req *api.ActivityListRequest) (map[string]any, error) {
	if req == nil {
		return map[string]any{}, nil
	}

	filters := map[string]any{}
	if req.Status > 0 {
		filters["act.status = ?"] = req.Status
	}

	var startFrom *time.Time
	if strings.TrimSpace(req.StartFrom) != "" {
		value, parseErr := util.ParseDateTime(strings.TrimSpace(req.StartFrom))
		if parseErr != nil {
			return nil, errors.New("开始时间格式错误")
		}
		startFrom = &value
		filters["act.start_time >= ?"] = startFrom
	}

	var startTo *time.Time
	if strings.TrimSpace(req.StartTo) != "" {
		value, parseErr := util.ParseDateTime(strings.TrimSpace(req.StartTo))
		if parseErr != nil {
			return nil, errors.New("结束时间格式错误")
		}
		startTo = &value
		filters["act.start_time <= ?"] = startTo
	}

	if startFrom != nil && startTo != nil && startTo.Before(*startFrom) {
		return nil, errors.New("结束时间不能早于开始时间")
	}
	return filters, nil
}

// hasPendingSignupCreateAudit 检查当前用户是否已有同活动同志愿者的待审核报名创建记录。
func (s *ActivityService) hasPendingSignupCreateAudit(db *gorm.DB, activityID, volunteerID, userID int64) (bool, error) {
	// 仅查询“活动报名 + 新增 + 待审核”的记录，再从快照中匹配 activity_id/volunteer_id。
	queryMap := map[string]any{
		"target_type = ?":    model.AuditTargetSignup,
		"operation_type = ?": model.OperationTypeCreate,
		"status = ?":         model.AuditStatusPending,
		"creator_id = ?":     userID,
	}
	records, _, err := s.repo.GetAuditRecordsList(db, queryMap, 0, 0)
	if err != nil {
		log.Error("活动报名前检查失败: 查询审核记录异常: %v, activity_id=%d user_id=%d volunteer_id=%d", err, activityID, userID, volunteerID)
		return false, err
	}

	for _, record := range records {
		if record == nil || record.TargetID > 0 {
			continue
		}

		var signup model.ActivitySignup
		if err := json.Unmarshal([]byte(record.NewContent), &signup); err != nil {
			continue
		}
		if signup.ActivityID == activityID && signup.VolunteerID == volunteerID {
			return true, nil
		}
	}

	return false, nil
}

// getVolunteerIDByAccountID 根据账号ID查询并返回对应志愿者ID。
func (s *ActivityService) getVolunteerIDByAccountID(accountID int64) (int64, error) {
	volunteer, err := s.repo.FindVolunteerByAccountID(s.repo.DB, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errors.New("志愿者信息不存在")
		}
		log.Error("查询志愿者ID失败: 查询志愿者信息异常: %v, account_id=%d", err, accountID)
		return 0, err
	}
	if volunteer == nil || volunteer.ID <= 0 {
		return 0, errors.New("志愿者信息不存在")
	}
	return volunteer.ID, nil
}

// ensureActivityOperableByCurrentOrg 校验活动存在、组织存在且当前组织对该活动有操作权限。
func (s *ActivityService) ensureActivityOperableByCurrentOrg(activityID, accountID int64) (*model.Activity, error) {
	activity, err := s.repo.GetActivityByID(s.repo.DB, activityID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		log.Error("校验活动归属失败: 查询活动异常: %v, activity_id=%d account_id=%d", err, activityID, accountID)
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

// settleSignupWorkHours 统一完成签退结算：计算工时、写流水、更新志愿者统计及报名记录。
func (s *ActivityService) settleSignupWorkHours(
	tx *gorm.DB,
	activity *model.Activity,
	signup *model.ActivitySignup,
	checkInTime time.Time,
	checkOutTime time.Time,
	operatorID int64,
	idempotencyKey string,
	reason string,
	syncCheckIn bool,
) (float64, error) {
	if activity == nil || signup == nil {
		return 0, errors.New("报名记录不存在")
	}
	if checkOutTime.Before(checkInTime) {
		return 0, errors.New("签退时间不能早于签到时间")
	}

	grantedHours := util.CalcGrantedHours(activity.Duration, checkInTime, checkOutTime)
	volunteer, err := s.repo.FindVolunteerByIDForUpdate(tx, signup.VolunteerID)
	if err != nil {
		return 0, err
	}

	beforeHours := volunteer.TotalHours
	beforeCount := int64(volunteer.ServiceCount)
	afterHours := util.RoundHours(beforeHours + grantedHours)
	afterCount := beforeCount + 1
	if afterHours < 0 || afterCount < 0 {
		return 0, errors.New("志愿者统计字段异常")
	}

	newVersion := signup.WorkHourVersion + 1
	workHourLog := &model.WorkHourLog{
		VolunteerID:        signup.VolunteerID,
		ActivityID:         signup.ActivityID,
		SignupID:           signup.ID,
		OperationType:      model.WorkHourOperationGrant,
		HoursDelta:         grantedHours,
		ServiceCountDelta:  1,
		BeforeTotalHours:   beforeHours,
		AfterTotalHours:    afterHours,
		BeforeServiceCount: beforeCount,
		AfterServiceCount:  afterCount,
		WorkHourVersion:    newVersion,
		IdempotencyKey:     idempotencyKey,
		RefLogID:           signup.LastWorkHourLogID,
		Reason:             reason,
		OperatorID:         operatorID,
	}
	if err := s.repo.CreateWorkHourLog(tx, workHourLog); err != nil {
		return 0, err
	}

	levelID, err := s.repo.ResolveLevelIDByTotalHours(tx, afterHours)
	if err != nil {
		return 0, err
	}

	if err := s.repo.UpdateVolunteer(tx, volunteer.ID, map[string]any{
		"total_hours":   afterHours,
		"service_count": int32(afterCount),
		"level_id":      levelID,
	}); err != nil {
		return 0, err
	}

	if err := s.repo.CreateRecord(tx, &model.Record{
		VolunteerID: signup.VolunteerID,
		Type:        "HOUR",
		Amount:      grantedHours,
		CreateTime:  workHourLog.CreatedAt,
	}); err != nil {
		return 0, err
	}

	signupUpdates := map[string]any{
		"check_out_status":      model.ActivityCheckOutDone,
		"check_out_time":        checkOutTime,
		"work_hour_status":      model.WorkHourStatusGranted,
		"work_hour_version":     newVersion,
		"last_work_hour_log_id": workHourLog.ID,
		"granted_hours":         grantedHours,
		"granted_at":            checkOutTime,
	}
	if syncCheckIn {
		signupUpdates["check_in_status"] = model.ActivityCheckInDone
		signupUpdates["check_in_time"] = checkInTime
	}
	if err := s.repo.UpdateActivitySignupByID(tx, signup.ID, signupUpdates); err != nil {
		return 0, err
	}

	return grantedHours, nil
}
