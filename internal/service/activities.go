package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/logger"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

var log = logger.GetLogger()

type ActivityService struct {
	Service
}

func NewActivityService(ctx context.Context, c *app.RequestContext) *ActivityService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &ActivityService{
		Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
}

// ActivityList 获取活动列表（活动总览）
func (s *ActivityService) ActivityList(req *api.ActivityListRequest) (*api.ActivityListResponse, error) {
	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}

	// 查询活动列表
	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	activities, total, err := s.repo.GetActivitiesByStatus(s.repo.DB, req.Status, pageSize, offset)
	if err != nil {
		log.Error("活动列表查询失败: %v, status=%d page=%d page_size=%d", err, req.Status, req.Page, req.PageSize)
		return nil, err
	}

	// 组装返回数据
	resp := &api.ActivityListResponse{
		Total: int32(total),
		List:  make([]*api.ActivityItem, 0, len(activities)),
	}

	for _, act := range activities {
		item := &api.ActivityItem{
			Id:            act.ID,
			Title:         act.Title,
			Description:   act.Description,
			CoverUrl:      act.CoverURL,
			StartTime:     act.StartTime.Format("2006-01-02 15:04:05"),
			EndTime:       act.EndTime.Format("2006-01-02 15:04:05"),
			Location:      act.Location,
			Duration:      act.Duration,
			MaxPeople:     act.MaxPeople,
			CurrentPeople: act.CurrentPeople,
			Status:        act.Status,
			IsFull:        act.MaxPeople > 0 && act.CurrentPeople >= act.MaxPeople,
		}
		resp.List = append(resp.List, item)
	}

	return resp, nil
}

// ActivitySignup 活动报名
func (s *ActivityService) ActivitySignup(req *api.ActivitySignupRequest) (*api.ActivitySignupResponse, error) {
	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}
	volunteerID, err := s.getVolunteerIDByAccountID(userID)
	if err != nil {
		log.Error("活动报名失败: 查询志愿者身份异常: %v, user_id=%d", err, userID)
		return nil, err
	}

	// 查询活动信息
	activity, err := s.repo.GetActivityByID(s.repo.DB, req.ActivityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		log.Error("活动报名失败: 查询活动异常: %v, activity_id=%d user_id=%d", err, req.ActivityId, userID)
		return nil, err
	}

	// 校验活动状态
	if activity.Status != model.ActivityStatusRecruiting {
		return nil, errors.New("活动已结束或已取消")
	}

	// 校验名额
	if activity.MaxPeople > 0 && activity.CurrentPeople >= activity.MaxPeople {
		return nil, errors.New("名额已满")
	}

	// 第一层去重：检查报名表（activity_signups）里是否已有有效报名记录（已落库）
	existing, signupErr := s.repo.GetSignup(s.repo.DB, req.ActivityId, volunteerID)
	if signupErr != nil {
		log.Error("活动报名前检查失败: 查询报名记录异常: %v, activity_id=%d user_id=%d volunteer_id=%d", signupErr, req.ActivityId, userID, volunteerID)
		return nil, signupErr
	}
	if existing != nil && (existing.Status == model.ActivitySignupStatusPending || existing.Status == model.ActivitySignupStatusSuccess) {
		return nil, errors.New("请勿重复报名")
	}

	// 第二层去重：检查审核表（audit_records）里是否已有待审核的创建申请（未落库）
	hasPendingAudit, err := s.hasPendingSignupCreateAudit(req.ActivityId, volunteerID, userID)
	if err != nil {
		log.Error("活动报名失败: 查询待审核报名异常: %v, activity_id=%d user_id=%d volunteer_id=%d", err, req.ActivityId, userID, volunteerID)
		return nil, err
	}
	if hasPendingAudit {
		return nil, errors.New("请勿重复报名")
	}

	signupSnapshot := &model.ActivitySignup{
		ActivityID:  req.ActivityId,
		VolunteerID: volunteerID,
		Status:      model.ActivitySignupStatusPending,
	}
	newContent, err := json.Marshal(signupSnapshot)
	if err != nil {
		log.Error("活动报名失败: 序列化报名快照异常: %v, activity_id=%d user_id=%d volunteer_id=%d", err, req.ActivityId, userID, volunteerID)
		return nil, err
	}

	record := &model.AuditRecord{
		TargetType:    model.AuditTargetSignup,
		TargetID:      0,
		CreatorID:     userID,
		AuditorID:     0,
		OldContent:    "{}",
		NewContent:    string(newContent),
		AuditResult:   0,
		RejectReason:  "",
		AuditTime:     time.Now(),
		OperationType: model.OperationTypeCreate,
		Status:        model.AuditStatusPending,
	}
	if err := s.repo.CreateAuditRecord(s.repo.DB, record); err != nil {
		log.Error("活动报名失败: 创建审核记录异常: %v, activity_id=%d user_id=%d volunteer_id=%d", err, req.ActivityId, userID, volunteerID)
		return nil, err
	}

	log.Info("活动报名申请已提交: activity_id=%d user_id=%d volunteer_id=%d record_id=%d", req.ActivityId, userID, volunteerID, record.ID)
	return &api.ActivitySignupResponse{Success: true}, nil
}

func (s *ActivityService) hasPendingSignupCreateAudit(activityID, volunteerID, userID int64) (bool, error) {
	// 仅查询“活动报名 + 新增 + 待审核”的记录，再从快照中匹配 activity_id/volunteer_id。
	queryMap := map[string]any{
		"target_type = ?":    model.AuditTargetSignup,
		"operation_type = ?": model.OperationTypeCreate,
		"status = ?":         model.AuditStatusPending,
		"creator_id = ?":     userID,
	}
	records, _, err := s.repo.GetAuditRecordsList(s.repo.DB, queryMap, 0, 0)
	if err != nil {
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

// ActivityCancel 取消报名
func (s *ActivityService) ActivityCancel(req *api.ActivityCancelRequest) (*api.ActivityCancelResponse, error) {
	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}
	volunteerID, err := s.getVolunteerIDByAccountID(userID)
	if err != nil {
		log.Error("取消报名失败: 查询志愿者身份异常: %v, user_id=%d", err, userID)
		return nil, err
	}

	// 查询报名记录
	signup, err := s.repo.GetSignup(s.repo.DB, req.ActivityId, volunteerID)
	if err != nil {
		log.Error("取消报名失败: 查询报名记录异常: %v, activity_id=%d user_id=%d volunteer_id=%d", err, req.ActivityId, userID, volunteerID)
		return nil, err
	}

	// 校验报名记录是否存在
	if signup == nil {
		return nil, errors.New("报名记录不存在")
	}

	// 校验报名状态
	if signup.Status != model.ActivitySignupStatusPending && signup.Status != model.ActivitySignupStatusSuccess {
		return nil, errors.New("当前状态不允许取消")
	}

	// 事务处理
	err = s.repo.DB.Transaction(func(tx *gorm.DB) error {
		// 更新报名状态为已取消
		signup.Status = model.ActivitySignupStatusCanceled
		if err := s.repo.UpdateSignupStatus(tx, signup); err != nil {
			log.Error("取消报名失败: 更新报名状态异常: %v, activity_id=%d user_id=%d volunteer_id=%d signup_id=%d", err, req.ActivityId, userID, volunteerID, signup.ID)
			return err
		}

		// 减少活动当前报名人数（原子操作）
		if err := s.repo.DecrementActivityPeople(tx, req.ActivityId); err != nil {
			log.Error("取消报名失败: 减少活动人数异常: %v, activity_id=%d user_id=%d", err, req.ActivityId, userID)
			return err
		}

		return nil
	})

	if err != nil {
		log.Error("取消报名失败: 事务执行失败: %v, activity_id=%d user_id=%d", err, req.ActivityId, userID)
		return nil, err
	}

	log.Info("取消报名成功: activity_id=%d user_id=%d volunteer_id=%d signup_id=%d", req.ActivityId, userID, volunteerID, signup.ID)
	return &api.ActivityCancelResponse{Success: true}, nil
}

// ActivityDetail 获取活动详情
func (s *ActivityService) ActivityDetail(req *api.ActivityDetailRequest) (*api.ActivityDetailResponse, error) {
	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	// 查询活动信息及组织名称
	activity, orgName, err := s.repo.GetActivityWithOrg(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		log.Error("活动详情查询失败: 查询活动异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}

	// 组装返回数据
	resp := &api.ActivityDetailResponse{
		Activity: &api.ActivityInfo{
			Id:             activity.ID,
			OrgId:          activity.OrgID,
			OrgName:        orgName,
			Title:          activity.Title,
			Description:    activity.Description,
			CoverUrl:       activity.CoverURL,
			StartTime:      util.FormatDateTimeOrEmpty(activity.StartTime),
			EndTime:        util.FormatDateTimeOrEmpty(activity.EndTime),
			Location:       activity.Location,
			Address:        activity.Address,
			Duration:       activity.Duration,
			MaxPeople:      activity.MaxPeople,
			CurrentPeople:  activity.CurrentPeople,
			Status:         activity.Status,
			IsRegistered:   false,
			CreatedAt:      util.FormatDateTimeOrEmpty(activity.CreatedAt),
			CheckInStatus:  model.ActivityCheckInPending,
			CheckInTime:    util.FormatDateTimePtr(nil),
			CheckOutStatus: model.ActivityCheckOutPending,
			CheckOutTime:   util.FormatDateTimePtr(nil),
			WorkHourStatus: model.WorkHourStatusPending,
			GrantedHours:   0,
		},
	}

	return resp, nil
}

// MyActivities 获取我的活动列表
func (s *ActivityService) MyActivities(req *api.MyActivitiesRequest) (*api.MyActivitiesResponse, error) {
	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}

	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}
	volunteerID, err := s.getVolunteerIDByAccountID(userID)
	if err != nil {
		log.Error("我的活动列表查询失败: 查询志愿者身份异常: %v, user_id=%d", err, userID)
		return nil, err
	}

	// 查询我的报名记录
	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	signups, total, err := s.repo.GetMyActivities(s.repo.DB, volunteerID, req.Status, pageSize, offset)
	if err != nil {
		log.Error("我的活动列表查询失败: 查询报名记录异常: %v, user_id=%d volunteer_id=%d status=%d page=%d page_size=%d", err, userID, volunteerID, req.Status, req.Page, req.PageSize)
		return nil, err
	}

	// 提取活动ID列表
	activityIDs := make([]int64, 0, len(signups))
	for _, signup := range signups {
		activityIDs = append(activityIDs, signup.ActivityID)
	}

	// 批量获取活动信息
	activityMap, err := s.repo.GetActivitiesByIDs(s.repo.DB, activityIDs)
	if err != nil {
		log.Error("我的活动列表查询失败: 批量查询活动异常: %v, user_id=%d volunteer_id=%d activity_count=%d", err, userID, volunteerID, len(activityIDs))
		return nil, err
	}

	// 提取组织ID列表
	orgIDs := make([]int64, 0, len(activityMap))
	for _, act := range activityMap {
		if act.OrgID > 0 {
			orgIDs = append(orgIDs, act.OrgID)
		}
	}

	// 批量获取组织名称
	orgNameMap, err := s.repo.GetOrgNamesByIDs(s.repo.DB, orgIDs)
	if err != nil {
		log.Error("我的活动列表查询失败: 批量查询组织名称异常: %v, user_id=%d volunteer_id=%d org_count=%d", err, userID, volunteerID, len(orgIDs))
		return nil, err
	}

	// 组装返回数据
	resp := &api.MyActivitiesResponse{
		Total: int32(total),
		List:  make([]*api.MyActivityItem, 0, len(signups)),
	}

	for _, signup := range signups {
		activity := activityMap[signup.ActivityID]
		if activity == nil {
			continue
		}

		checkInTime := util.FormatDateTimePtr(signup.CheckInTime)
		checkOutTime := util.FormatDateTimePtr(signup.CheckOutTime)

		orgName := ""
		if activity.OrgID > 0 {
			orgName = orgNameMap[activity.OrgID]
		}

		item := &api.MyActivityItem{
			Id:             activity.ID,
			OrgId:          activity.OrgID,
			OrgName:        orgName,
			Title:          activity.Title,
			Description:    activity.Description,
			CoverUrl:       activity.CoverURL,
			StartTime:      util.FormatDateTimeOrEmpty(activity.StartTime),
			EndTime:        util.FormatDateTimeOrEmpty(activity.EndTime),
			Location:       activity.Location,
			Duration:       activity.Duration,
			MaxPeople:      activity.MaxPeople,
			CurrentPeople:  activity.CurrentPeople,
			Status:         activity.Status,
			SignupTime:     util.FormatDateTimeOrEmpty(signup.SignupTime),
			CheckInStatus:  signup.CheckInStatus,
			CheckInTime:    checkInTime,
			CheckOutStatus: signup.CheckOutStatus,
			CheckOutTime:   checkOutTime,
			WorkHourStatus: signup.WorkHourStatus,
			GrantedHours:   signup.GrantedHours,
		}
		resp.List = append(resp.List, item)
	}

	return resp, nil
}

// ========== 组织端活动管理 ==========

// CreateActivity 创建活动
func (s *ActivityService) CreateActivity(req *api.CreateActivityRequest) (*api.CreateActivityResponse, error) {
	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	// 校验必填字段
	if req.OrgId <= 0 {
		return nil, errors.New("组织ID不能为空")
	}

	// 根据传入的 org_id 查询组织信息
	org, err := s.repo.GetOrganizationByID(s.repo.DB, req.OrgId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("组织不存在")
		}
		log.Error("创建活动失败: 查询组织异常: %v, org_id=%d user_id=%d", err, req.OrgId, userID)
		return nil, err
	}

	// 校验组织是否属于当前登录的管理者
	if org.AccountID != userID {
		return nil, errors.New("无权为该组织创建活动")
	}

	// 解析时间
	startTime, err := time.Parse("2006-01-02 15:04:05", req.StartTime)
	if err != nil {
		return nil, errors.New("开始时间格式错误")
	}
	endTime, err := time.Parse("2006-01-02 15:04:05", req.EndTime)
	if err != nil {
		return nil, errors.New("结束时间格式错误")
	}

	// 校验时间
	if endTime.Before(startTime) {
		return nil, errors.New("结束时间不能早于开始时间")
	}
	if startTime.Before(time.Now()) {
		return nil, errors.New("开始时间不能早于当前时间")
	}

	// 创建活动
	activity := &model.Activity{
		OrgID:         req.OrgId,
		Title:         req.Title,
		Description:   req.Description,
		CoverURL:      req.CoverUrl,
		StartTime:     startTime,
		EndTime:       endTime,
		Location:      req.Location,
		Address:       req.Address,
		Duration:      req.Duration,
		MaxPeople:     req.MaxPeople,
		CurrentPeople: 0,
		Status:        model.ActivityStatusRecruiting,
	}

	if err := s.repo.CreateActivity(s.repo.DB, activity); err != nil {
		log.Error("创建活动失败: 写入活动异常: %v, org_id=%d user_id=%d", err, req.OrgId, userID)
		return nil, err
	}

	log.Info("创建活动成功: activity_id=%d org_id=%d user_id=%d", activity.ID, req.OrgId, userID)
	return &api.CreateActivityResponse{
		Id:      activity.ID,
		Message: "创建活动成功",
	}, nil
}

// UpdateActivity 更新活动
func (s *ActivityService) UpdateActivity(req *api.UpdateActivityRequest) (*api.UpdateActivityResponse, error) {
	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	// 查询活动信息
	activity, err := s.repo.GetActivityByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		log.Error("更新活动失败: 查询活动异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}

	// 查询组织信息
	org, err := s.repo.GetOrganizationByAccountID(s.repo.DB, userID)
	if err != nil {
		log.Error("更新活动失败: 查询组织信息异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, errors.New("组织信息不存在")
	}

	// 校验活动归属
	if activity.OrgID != org.ID {
		return nil, errors.New("无权操作此活动")
	}

	// 校验活动状态
	if activity.Status == model.ActivityStatusFinished || activity.Status == model.ActivityStatusCanceled {
		return nil, errors.New("已结束或已取消的活动不能修改")
	}

	// 解析时间
	if req.StartTime != "" {
		startTime, err := time.Parse("2006-01-02 15:04:05", req.StartTime)
		if err != nil {
			return nil, errors.New("开始时间格式错误")
		}
		activity.StartTime = startTime
	}
	if req.EndTime != "" {
		endTime, err := time.Parse("2006-01-02 15:04:05", req.EndTime)
		if err != nil {
			return nil, errors.New("结束时间格式错误")
		}
		activity.EndTime = endTime
	}

	// 校验时间
	if activity.EndTime.Before(activity.StartTime) {
		return nil, errors.New("结束时间不能早于开始时间")
	}

	// 更新字段
	if req.Title != "" {
		activity.Title = req.Title
	}
	if req.Description != "" {
		activity.Description = req.Description
	}
	if req.CoverUrl != "" {
		activity.CoverURL = req.CoverUrl
	}
	if req.Location != "" {
		activity.Location = req.Location
	}
	if req.Address != "" {
		activity.Address = req.Address
	}
	if req.Duration > 0 {
		activity.Duration = req.Duration
	}
	if req.MaxPeople >= 0 {
		// 检查是否会导致报名人数超过新设定的最大人数
		if req.MaxPeople > 0 && activity.CurrentPeople > req.MaxPeople {
			return nil, errors.New("当前报名人数超过新设定的最大人数")
		}
		activity.MaxPeople = req.MaxPeople
	}

	if err := s.repo.UpdateActivity(s.repo.DB, activity); err != nil {
		log.Error("更新活动失败: 更新活动异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}

	log.Info("更新活动成功: activity_id=%d org_id=%d user_id=%d", activity.ID, org.ID, userID)
	return &api.UpdateActivityResponse{
		Message: "更新活动成功",
	}, nil
}

// DeleteActivity 删除活动
func (s *ActivityService) DeleteActivity(req *api.DeleteActivityRequest) (*api.DeleteActivityResponse, error) {
	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	// 查询活动信息
	activity, err := s.repo.GetActivityByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		log.Error("删除活动失败: 查询活动异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}

	// 查询组织信息
	org, err := s.repo.GetOrganizationByAccountID(s.repo.DB, userID)
	if err != nil {
		log.Error("删除活动失败: 查询组织信息异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, errors.New("组织信息不存在")
	}

	// 校验活动归属
	if activity.OrgID != org.ID {
		return nil, errors.New("无权操作此活动")
	}

	// 校验活动状态
	if activity.Status == model.ActivityStatusFinished {
		return nil, errors.New("已结束的活动不能删除")
	}

	// 检查是否有已报名的志愿者
	if activity.CurrentPeople > 0 {
		// 可以选择允许删除或拒绝
		// 这里选择允许删除，记录日志
	}

	if err := s.repo.DeleteActivity(s.repo.DB, req.Id); err != nil {
		log.Error("删除活动失败: 删除活动异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}

	log.Info("删除活动成功: activity_id=%d org_id=%d user_id=%d", req.Id, org.ID, userID)
	return &api.DeleteActivityResponse{
		Message: "删除活动成功",
	}, nil
}

// CancelActivity 取消活动
func (s *ActivityService) CancelActivity(req *api.CancelActivityRequest) (*api.CancelActivityResponse, error) {
	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	// 查询活动信息
	activity, err := s.repo.GetActivityByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		log.Error("取消活动失败: 查询活动异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}

	// 查询组织信息
	org, err := s.repo.GetOrganizationByAccountID(s.repo.DB, userID)
	if err != nil {
		log.Error("取消活动失败: 查询组织信息异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, errors.New("组织信息不存在")
	}

	// 校验活动归属
	if activity.OrgID != org.ID {
		return nil, errors.New("无权操作此活动")
	}

	// 校验活动状态
	if activity.Status == model.ActivityStatusFinished || activity.Status == model.ActivityStatusCanceled {
		return nil, errors.New("已结束或已取消的活动不能取消")
	}

	if err := s.repo.CancelActivity(s.repo.DB, req.Id); err != nil {
		log.Error("取消活动失败: 更新活动状态异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}

	log.Info("取消活动成功: activity_id=%d org_id=%d user_id=%d", req.Id, org.ID, userID)
	return &api.CancelActivityResponse{
		Message: "取消活动成功",
	}, nil
}

// FinishActivity 完结活动
func (s *ActivityService) FinishActivity(req *api.FinishActivityRequest) (*api.FinishActivityResponse, error) {
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	activity, err := s.ensureActivityOperableByCurrentOrg(req.Id, userID)
	if err != nil {
		return nil, err
	}
	if activity.Status == model.ActivityStatusFinished {
		return nil, errors.New("活动已结束")
	}
	if activity.Status == model.ActivityStatusCanceled {
		return nil, errors.New("已取消活动不能完结")
	}

	if err := s.repo.FinishActivity(s.repo.DB, req.Id); err != nil {
		log.Error("完结活动失败: 更新活动状态异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}

	log.Info("完结活动成功: activity_id=%d user_id=%d", req.Id, userID)
	return &api.FinishActivityResponse{Message: "完结活动成功"}, nil
}

// ActivityCheckIn 活动签到（志愿者侧）
func (s *ActivityService) ActivityCheckIn(req *api.ActivityCheckInRequest) (*api.ActivityCheckInResponse, error) {
	if req.ActivityId <= 0 {
		return nil, errors.New("活动ID不能为空")
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}
	volunteerID, err := s.getVolunteerIDByAccountID(userID)
	if err != nil {
		log.Error("活动签到失败: 查询志愿者身份异常: %v, user_id=%d", err, userID)
		return nil, err
	}

	activity, err := s.repo.GetActivityByID(s.repo.DB, req.ActivityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		return nil, err
	}
	if activity.Status == model.ActivityStatusCanceled {
		return nil, errors.New("已取消活动不允许签到")
	}

	var checkInTime time.Time
	err = s.withTransaction(func(tx *gorm.DB) error {
		signup, err := s.repo.GetSignupForUpdate(tx, req.ActivityId, volunteerID)
		if err != nil {
			return err
		}
		if signup == nil {
			return errors.New("报名记录不存在")
		}
		if signup.Status != model.ActivitySignupStatusSuccess {
			return errors.New("当前报名状态不允许签到")
		}
		if signup.CheckOutStatus == model.ActivityCheckOutDone {
			return errors.New("已签退，无法再次签到")
		}
		if signup.CheckInStatus == model.ActivityCheckInDone {
			if signup.CheckInTime != nil {
				checkInTime = *signup.CheckInTime
			}
			return nil
		}

		now := time.Now()
		checkInTime = now
		return s.repo.UpdateActivitySignupByID(tx, signup.ID, map[string]any{
			"check_in_status": model.ActivityCheckInDone,
			"check_in_time":   now,
		})
	})
	if err != nil {
		log.Error("活动签到失败: %v, activity_id=%d volunteer_id=%d user_id=%d", err, req.ActivityId, volunteerID, userID)
		return nil, err
	}

	log.Info("活动签到成功: activity_id=%d volunteer_id=%d user_id=%d", req.ActivityId, volunteerID, userID)
	return &api.ActivityCheckInResponse{
		Success:     true,
		CheckInTime: util.FormatDateTimeOrEmpty(checkInTime),
	}, nil
}

// ActivityCheckOut 活动签退（志愿者侧，签退后自动结算工时）
func (s *ActivityService) ActivityCheckOut(req *api.ActivityCheckOutRequest) (*api.ActivityCheckOutResponse, error) {
	if req.ActivityId <= 0 {
		return nil, errors.New("活动ID不能为空")
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}
	volunteerID, err := s.getVolunteerIDByAccountID(userID)
	if err != nil {
		log.Error("活动签退失败: 查询志愿者身份异常: %v, user_id=%d", err, userID)
		return nil, err
	}

	activity, err := s.repo.GetActivityByID(s.repo.DB, req.ActivityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		return nil, err
	}
	if activity.Status == model.ActivityStatusCanceled {
		return nil, errors.New("已取消活动不允许签退")
	}

	var checkOutTime time.Time
	var grantedHours float64
	err = s.withTransaction(func(tx *gorm.DB) error {
		signup, err := s.repo.GetSignupForUpdate(tx, req.ActivityId, volunteerID)
		if err != nil {
			return err
		}
		if signup == nil {
			return errors.New("报名记录不存在")
		}
		if signup.Status != model.ActivitySignupStatusSuccess {
			return errors.New("当前报名状态不允许签退")
		}
		if signup.CheckInStatus != model.ActivityCheckInDone || signup.CheckInTime == nil {
			return errors.New("未签到，无法签退")
		}

		if signup.CheckOutStatus == model.ActivityCheckOutDone {
			if signup.CheckOutTime != nil {
				checkOutTime = *signup.CheckOutTime
			}
			grantedHours = signup.GrantedHours
			return nil
		}

		now := time.Now()
		if now.Before(*signup.CheckInTime) {
			now = *signup.CheckInTime
		}
		checkOutTime = now
		grantedHours = util.CalcGrantedHours(activity.Duration, *signup.CheckInTime, now)

		volunteer, err := s.repo.FindVolunteerByIDForUpdate(tx, signup.VolunteerID)
		if err != nil {
			return err
		}
		beforeHours := volunteer.TotalHours
		beforeCount := int64(volunteer.ServiceCount)
		afterHours := util.RoundHours(beforeHours + grantedHours)
		afterCount := beforeCount + 1
		if afterHours < 0 || afterCount < 0 {
			return errors.New("志愿者统计字段异常")
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
			IdempotencyKey:     fmt.Sprintf("checkout:%d:%d", signup.ID, newVersion),
			RefLogID:           signup.LastWorkHourLogID,
			Reason:             "签到签退自动结算",
			OperatorID:         userID,
		}
		if err := s.repo.CreateWorkHourLog(tx, workHourLog); err != nil {
			return err
		}

		if err := s.repo.UpdateVolunteer(tx, volunteer.ID, map[string]interface{}{
			"total_hours":   afterHours,
			"service_count": int32(afterCount),
		}); err != nil {
			return err
		}

		return s.repo.UpdateActivitySignupByID(tx, signup.ID, map[string]any{
			"check_out_status":      model.ActivityCheckOutDone,
			"check_out_time":        now,
			"work_hour_status":      model.WorkHourStatusGranted,
			"work_hour_version":     newVersion,
			"last_work_hour_log_id": workHourLog.ID,
			"granted_hours":         grantedHours,
			"granted_at":            now,
		})
	})
	if err != nil {
		log.Error("活动签退失败: %v, activity_id=%d volunteer_id=%d user_id=%d", err, req.ActivityId, volunteerID, userID)
		return nil, err
	}

	log.Info("活动签退成功: activity_id=%d volunteer_id=%d user_id=%d granted_hours=%.2f", req.ActivityId, volunteerID, userID, grantedHours)
	return &api.ActivityCheckOutResponse{
		Success:      true,
		CheckOutTime: util.FormatDateTimeOrEmpty(checkOutTime),
		GrantedHours: grantedHours,
	}, nil
}

// ActivitySupplementAttendance 活动签到签退补录（组织侧）
func (s *ActivityService) ActivitySupplementAttendance(req *api.ActivitySupplementAttendanceRequest) (*api.ActivitySupplementAttendanceResponse, error) {
	if req.ActivityId <= 0 || req.VolunteerId <= 0 {
		return nil, errors.New("活动ID和志愿者ID不能为空")
	}

	checkOutText := strings.TrimSpace(req.CheckOutTime)
	if checkOutText == "" {
		return nil, errors.New("签退时间不能为空")
	}
	checkOutAt, err := util.ParseDateTime(checkOutText)
	if err != nil {
		return nil, errors.New("签退时间格式错误")
	}

	checkInText := strings.TrimSpace(req.CheckInTime)
	var checkInAt time.Time
	hasCheckInInput := false
	if checkInText != "" {
		checkInAt, err = util.ParseDateTime(checkInText)
		if err != nil {
			return nil, errors.New("签到时间格式错误")
		}
		hasCheckInInput = true
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	activity, err := s.ensureActivityOperableByCurrentOrg(req.ActivityId, userID)
	if err != nil {
		return nil, err
	}
	if activity.Status == model.ActivityStatusCanceled {
		return nil, errors.New("已取消活动不允许补录")
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		reason = "组织补录签到签退"
	}

	var finalCheckIn time.Time
	var finalCheckOut time.Time
	var grantedHours float64

	err = s.withTransaction(func(tx *gorm.DB) error {
		signup, err := s.repo.GetSignupForUpdate(tx, req.ActivityId, req.VolunteerId)
		if err != nil {
			return err
		}
		if signup == nil {
			return errors.New("报名记录不存在")
		}
		if signup.Status != model.ActivitySignupStatusSuccess {
			return errors.New("当前报名状态不允许补录")
		}

		// 已签退场景直接视为幂等成功，返回已有结果。
		if signup.CheckOutStatus == model.ActivityCheckOutDone {
			if signup.CheckInTime != nil {
				finalCheckIn = *signup.CheckInTime
			}
			if signup.CheckOutTime != nil {
				finalCheckOut = *signup.CheckOutTime
			}
			grantedHours = signup.GrantedHours
			return nil
		}

		if signup.CheckInStatus == model.ActivityCheckInDone {
			if signup.CheckInTime == nil {
				return errors.New("签到数据异常")
			}
			finalCheckIn = *signup.CheckInTime
			if hasCheckInInput && !checkInAt.Equal(finalCheckIn) {
				return errors.New("已签到，不允许补录签到时间")
			}
		} else {
			if !hasCheckInInput {
				return errors.New("未签到时必须补录签到时间")
			}
			finalCheckIn = checkInAt
		}

		if checkOutAt.Before(finalCheckIn) {
			return errors.New("签退时间不能早于签到时间")
		}
		finalCheckOut = checkOutAt
		grantedHours = util.CalcGrantedHours(activity.Duration, finalCheckIn, finalCheckOut)

		volunteer, err := s.repo.FindVolunteerByIDForUpdate(tx, signup.VolunteerID)
		if err != nil {
			return err
		}
		beforeHours := volunteer.TotalHours
		beforeCount := int64(volunteer.ServiceCount)
		afterHours := util.RoundHours(beforeHours + grantedHours)
		afterCount := beforeCount + 1
		if afterHours < 0 || afterCount < 0 {
			return errors.New("志愿者统计字段异常")
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
			IdempotencyKey:     fmt.Sprintf("org-supplement:%d:%d:%d", signup.ID, newVersion, finalCheckOut.Unix()),
			RefLogID:           signup.LastWorkHourLogID,
			Reason:             reason,
			OperatorID:         userID,
		}
		if err := s.repo.CreateWorkHourLog(tx, workHourLog); err != nil {
			return err
		}

		if err := s.repo.UpdateVolunteer(tx, volunteer.ID, map[string]interface{}{
			"total_hours":   afterHours,
			"service_count": int32(afterCount),
		}); err != nil {
			return err
		}

		return s.repo.UpdateActivitySignupByID(tx, signup.ID, map[string]any{
			"check_in_status":       model.ActivityCheckInDone,
			"check_in_time":         finalCheckIn,
			"check_out_status":      model.ActivityCheckOutDone,
			"check_out_time":        finalCheckOut,
			"work_hour_status":      model.WorkHourStatusGranted,
			"work_hour_version":     newVersion,
			"last_work_hour_log_id": workHourLog.ID,
			"granted_hours":         grantedHours,
			"granted_at":            finalCheckOut,
		})
	})
	if err != nil {
		log.Error("活动补录失败: %v, activity_id=%d volunteer_id=%d user_id=%d", err, req.ActivityId, req.VolunteerId, userID)
		return nil, err
	}

	log.Info("活动补录成功: activity_id=%d volunteer_id=%d user_id=%d granted_hours=%.2f", req.ActivityId, req.VolunteerId, userID, grantedHours)
	return &api.ActivitySupplementAttendanceResponse{
		Success:      true,
		CheckInTime:  util.FormatDateTimeOrEmpty(finalCheckIn),
		CheckOutTime: util.FormatDateTimeOrEmpty(finalCheckOut),
		GrantedHours: grantedHours,
	}, nil
}

func (s *ActivityService) getVolunteerIDByAccountID(accountID int64) (int64, error) {
	volunteer, err := s.repo.FindVolunteerByAccountID(s.repo.DB, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errors.New("志愿者信息不存在")
		}
		return 0, err
	}
	if volunteer == nil || volunteer.ID <= 0 {
		return 0, errors.New("志愿者信息不存在")
	}
	return volunteer.ID, nil
}

func (s *ActivityService) ensureActivityOperableByCurrentOrg(activityID, accountID int64) (*model.Activity, error) {
	activity, err := s.repo.GetActivityByID(s.repo.DB, activityID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		return nil, err
	}

	org, err := s.repo.GetOrganizationByAccountID(s.repo.DB, accountID)
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
