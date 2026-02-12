package service

import (
	"context"
	"errors"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/logger"

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

// todo 活动报名不是直接创建数据，而是创建审核数据，走审核流程
// ActivitySignup 活动报名
func (s *ActivityService) ActivitySignup(req *api.ActivitySignupRequest) (*api.ActivitySignupResponse, error) {
	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Warn("活动报名失败: 获取当前用户失败: %v, activity_id=%d", err, req.ActivityId)
		return nil, err
	}

	// 查询活动信息
	activity, err := s.repo.GetActivityByID(s.repo.DB, req.ActivityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("活动报名失败: 活动不存在, activity_id=%d user_id=%d", req.ActivityId, userID)
			return nil, errors.New("活动不存在")
		}
		log.Error("活动报名失败: 查询活动异常: %v, activity_id=%d user_id=%d", err, req.ActivityId, userID)
		return nil, err
	}

	// 校验活动状态
	if activity.Status != 1 {
		log.Warn("活动报名失败: 活动状态不允许报名, activity_id=%d user_id=%d status=%d", req.ActivityId, userID, activity.Status)
		return nil, errors.New("活动已结束或已取消")
	}

	// 校验名额
	if activity.MaxPeople > 0 && activity.CurrentPeople >= activity.MaxPeople {
		log.Warn("活动报名失败: 名额已满, activity_id=%d user_id=%d max_people=%d current_people=%d", req.ActivityId, userID, activity.MaxPeople, activity.CurrentPeople)
		return nil, errors.New("名额已满")
	}

	// 校验是否重复报名
	existing, signupErr := s.repo.GetSignup(s.repo.DB, req.ActivityId, userID)
	if signupErr != nil {
		log.Error("活动报名前检查失败: 查询报名记录异常: %v, activity_id=%d user_id=%d", signupErr, req.ActivityId, userID)
	}
	if existing != nil && (existing.Status == model.ActivitySignupStatusPending || existing.Status == model.ActivitySignupStatusSuccess) {
		log.Warn("活动报名失败: 重复报名, activity_id=%d user_id=%d", req.ActivityId, userID)
		return nil, errors.New("请勿重复报名")
	}

	// 事务处理
	err = s.repo.DB.Transaction(func(tx *gorm.DB) error {
		// 创建报名记录
		signup := &model.ActivitySignup{
			ActivityID:  req.ActivityId,
			VolunteerID: userID,
			Status:      model.ActivitySignupStatusPending,
		}
		if err := s.repo.CreateSignup(tx, signup); err != nil {
			log.Error("活动报名失败: 创建报名记录异常: %v, activity_id=%d user_id=%d", err, req.ActivityId, userID)
			return err
		}

		// 增加活动当前报名人数（原子操作）
		if err := s.repo.IncrementActivityPeople(tx, req.ActivityId); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Warn("活动报名失败: 增加报名人数时活动不存在或已满, activity_id=%d user_id=%d", req.ActivityId, userID)
				return errors.New("名额已满")
			}
			log.Error("活动报名失败: 增加报名人数异常: %v, activity_id=%d user_id=%d", err, req.ActivityId, userID)
			return err
		}

		return nil
	})

	if err != nil {
		log.Warn("活动报名失败: 事务执行失败: %v, activity_id=%d user_id=%d", err, req.ActivityId, userID)
		return nil, err
	}
	return &api.ActivitySignupResponse{Success: true}, nil
}

// ActivityCancel 取消报名
func (s *ActivityService) ActivityCancel(req *api.ActivityCancelRequest) (*api.ActivityCancelResponse, error) {
	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Warn("取消报名失败: 获取当前用户失败: %v, activity_id=%d", err, req.ActivityId)
		return nil, err
	}

	// 查询报名记录
	signup, err := s.repo.GetSignup(s.repo.DB, req.ActivityId, userID)
	if err != nil {
		log.Error("取消报名失败: 查询报名记录异常: %v, activity_id=%d user_id=%d", err, req.ActivityId, userID)
		return nil, err
	}

	// 校验报名记录是否存在
	if signup == nil {
		log.Warn("取消报名失败: 报名记录不存在, activity_id=%d user_id=%d", req.ActivityId, userID)
		return nil, errors.New("报名记录不存在")
	}

	// 校验报名状态
	if signup.Status != model.ActivitySignupStatusPending && signup.Status != model.ActivitySignupStatusSuccess {
		log.Warn("取消报名失败: 当前状态不允许取消, activity_id=%d user_id=%d signup_status=%d", req.ActivityId, userID, signup.Status)
		return nil, errors.New("当前状态不允许取消")
	}

	// 事务处理
	err = s.repo.DB.Transaction(func(tx *gorm.DB) error {
		// 更新报名状态为已取消
		signup.Status = model.ActivitySignupStatusCanceled
		if err := s.repo.UpdateSignupStatus(tx, signup); err != nil {
			log.Error("取消报名失败: 更新报名状态异常: %v, activity_id=%d user_id=%d signup_id=%d", err, req.ActivityId, userID, signup.ID)
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
	return &api.ActivityCancelResponse{Success: true}, nil
}

// ActivityDetail 获取活动详情
func (s *ActivityService) ActivityDetail(req *api.ActivityDetailRequest) (*api.ActivityDetailResponse, error) {
	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Warn("活动详情查询失败: 获取当前用户失败: %v, activity_id=%d", err, req.Id)
		return nil, err
	}

	// 查询活动信息及组织名称
	activity, orgName, err := s.repo.GetActivityWithOrg(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("活动详情查询失败: 活动不存在, activity_id=%d user_id=%d", req.Id, userID)
			return nil, errors.New("活动不存在")
		}
		log.Error("活动详情查询失败: 查询活动异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}

	// 查询用户是否已报名
	signup, signupErr := s.repo.GetSignup(s.repo.DB, req.Id, userID)
	if signupErr != nil {
		log.Warn("活动详情查询: 查询报名状态失败: %v, activity_id=%d user_id=%d", signupErr, req.Id, userID)
	}
	isRegistered := signup != nil && (signup.Status == model.ActivitySignupStatusPending || signup.Status == model.ActivitySignupStatusSuccess)

	// 组装返回数据
	resp := &api.ActivityDetailResponse{
		Activity: &api.ActivityInfo{
			Id:            activity.ID,
			OrgId:         activity.OrgID,
			OrgName:       orgName,
			Title:         activity.Title,
			Description:   activity.Description,
			CoverUrl:      activity.CoverURL,
			StartTime:     activity.StartTime.Format("2006-01-02 15:04:05"),
			EndTime:       activity.EndTime.Format("2006-01-02 15:04:05"),
			Location:      activity.Location,
			Address:       activity.Address,
			Duration:      activity.Duration,
			MaxPeople:     activity.MaxPeople,
			CurrentPeople: activity.CurrentPeople,
			Status:        activity.Status,
			IsRegistered:  isRegistered,
			CreatedAt:     activity.CreatedAt.Format("2006-01-02 15:04:05"),
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
		log.Warn("我的活动列表查询失败: 获取当前用户失败: %v", err)
		return nil, err
	}

	// 查询我的报名记录
	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	signups, total, err := s.repo.GetMyActivities(s.repo.DB, userID, req.Status, pageSize, offset)
	if err != nil {
		log.Error("我的活动列表查询失败: 查询报名记录异常: %v, user_id=%d status=%d page=%d page_size=%d", err, userID, req.Status, req.Page, req.PageSize)
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
		log.Error("我的活动列表查询失败: 批量查询活动异常: %v, user_id=%d activity_count=%d", err, userID, len(activityIDs))
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
		log.Error("我的活动列表查询失败: 批量查询组织名称异常: %v, user_id=%d org_count=%d", err, userID, len(orgIDs))
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

		checkInTime := ""
		if signup.CheckInTime != nil {
			checkInTime = signup.CheckInTime.Format("2006-01-02 15:04:05")
		}

		orgName := ""
		if activity.OrgID > 0 {
			orgName = orgNameMap[activity.OrgID]
		}

		item := &api.MyActivityItem{
			Id:            activity.ID,
			OrgId:         activity.OrgID,
			OrgName:       orgName,
			Title:         activity.Title,
			Description:   activity.Description,
			CoverUrl:      activity.CoverURL,
			StartTime:     activity.StartTime.Format("2006-01-02 15:04:05"),
			EndTime:       activity.EndTime.Format("2006-01-02 15:04:05"),
			Location:      activity.Location,
			Duration:      activity.Duration,
			MaxPeople:     activity.MaxPeople,
			CurrentPeople: activity.CurrentPeople,
			Status:        activity.Status,
			SignupTime:    signup.SignupTime.Format("2006-01-02 15:04:05"),
			CheckInStatus: signup.CheckInStatus,
			CheckInTime:   checkInTime,
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
		log.Warn("创建活动失败: 获取当前用户失败: %v, org_id=%d", err, req.OrgId)
		return nil, err
	}

	// 校验必填字段
	if req.OrgId <= 0 {
		log.Warn("创建活动失败: 组织ID为空, user_id=%d", userID)
		return nil, errors.New("组织ID不能为空")
	}

	// 根据传入的 org_id 查询组织信息
	org, err := s.repo.GetOrganizationByID(s.repo.DB, req.OrgId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("创建活动失败: 组织不存在, org_id=%d user_id=%d", req.OrgId, userID)
			return nil, errors.New("组织不存在")
		}
		log.Error("创建活动失败: 查询组织异常: %v, org_id=%d user_id=%d", err, req.OrgId, userID)
		return nil, err
	}

	// 校验组织是否属于当前登录的管理者
	if org.AccountID != userID {
		log.Warn("创建活动失败: 无权为该组织创建活动, org_id=%d user_id=%d owner_id=%d", req.OrgId, userID, org.AccountID)
		return nil, errors.New("无权为该组织创建活动")
	}

	// 解析时间
	startTime, err := time.Parse("2006-01-02 15:04:05", req.StartTime)
	if err != nil {
		log.Warn("创建活动失败: 开始时间格式错误, start_time=%s org_id=%d user_id=%d", req.StartTime, req.OrgId, userID)
		return nil, errors.New("开始时间格式错误")
	}
	endTime, err := time.Parse("2006-01-02 15:04:05", req.EndTime)
	if err != nil {
		log.Warn("创建活动失败: 结束时间格式错误, end_time=%s org_id=%d user_id=%d", req.EndTime, req.OrgId, userID)
		return nil, errors.New("结束时间格式错误")
	}

	// 校验时间
	if endTime.Before(startTime) {
		log.Warn("创建活动失败: 结束时间早于开始时间, org_id=%d user_id=%d start_time=%s end_time=%s", req.OrgId, userID, req.StartTime, req.EndTime)
		return nil, errors.New("结束时间不能早于开始时间")
	}
	if startTime.Before(time.Now()) {
		log.Warn("创建活动失败: 开始时间早于当前时间, org_id=%d user_id=%d start_time=%s", req.OrgId, userID, req.StartTime)
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
		Status:        1, // 1-报名中
	}

	if err := s.repo.CreateActivity(s.repo.DB, activity); err != nil {
		log.Error("创建活动失败: 写入活动异常: %v, org_id=%d user_id=%d", err, req.OrgId, userID)
		return nil, err
	}

	log.Info("创建活动成功 activity_id=%d org_id=%d user_id=%d", activity.ID, req.OrgId, userID)

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
		log.Warn("更新活动失败: 获取当前用户失败: %v, activity_id=%d", err, req.Id)
		return nil, err
	}

	// 查询活动信息
	activity, err := s.repo.GetActivityByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("更新活动失败: 活动不存在, activity_id=%d user_id=%d", req.Id, userID)
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
		log.Warn("更新活动失败: 无权操作此活动, activity_id=%d activity_org_id=%d user_org_id=%d user_id=%d", req.Id, activity.OrgID, org.ID, userID)
		return nil, errors.New("无权操作此活动")
	}

	// 校验活动状态
	if activity.Status == 2 || activity.Status == 3 {
		log.Warn("更新活动失败: 活动状态不允许修改, activity_id=%d user_id=%d status=%d", req.Id, userID, activity.Status)
		return nil, errors.New("已结束或已取消的活动不能修改")
	}

	// 解析时间
	if req.StartTime != "" {
		startTime, err := time.Parse("2006-01-02 15:04:05", req.StartTime)
		if err != nil {
			log.Warn("更新活动失败: 开始时间格式错误, activity_id=%d user_id=%d start_time=%s", req.Id, userID, req.StartTime)
			return nil, errors.New("开始时间格式错误")
		}
		activity.StartTime = startTime
	}
	if req.EndTime != "" {
		endTime, err := time.Parse("2006-01-02 15:04:05", req.EndTime)
		if err != nil {
			log.Warn("更新活动失败: 结束时间格式错误, activity_id=%d user_id=%d end_time=%s", req.Id, userID, req.EndTime)
			return nil, errors.New("结束时间格式错误")
		}
		activity.EndTime = endTime
	}

	// 校验时间
	if activity.EndTime.Before(activity.StartTime) {
		log.Warn("更新活动失败: 结束时间早于开始时间, activity_id=%d user_id=%d", req.Id, userID)
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
			log.Warn("更新活动失败: 新名额小于当前报名人数, activity_id=%d user_id=%d current_people=%d new_max_people=%d", req.Id, userID, activity.CurrentPeople, req.MaxPeople)
			return nil, errors.New("当前报名人数超过新设定的最大人数")
		}
		activity.MaxPeople = req.MaxPeople
	}

	if err := s.repo.UpdateActivity(s.repo.DB, activity); err != nil {
		log.Error("更新活动失败: 更新活动异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}

	log.Info("更新活动成功 activity_id=%d org_id=%d user_id=%d", activity.ID, org.ID, userID)

	return &api.UpdateActivityResponse{
		Message: "更新活动成功",
	}, nil
}

// DeleteActivity 删除活动
func (s *ActivityService) DeleteActivity(req *api.DeleteActivityRequest) (*api.DeleteActivityResponse, error) {
	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Warn("删除活动失败: 获取当前用户失败: %v, activity_id=%d", err, req.Id)
		return nil, err
	}

	// 查询活动信息
	activity, err := s.repo.GetActivityByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("删除活动失败: 活动不存在, activity_id=%d user_id=%d", req.Id, userID)
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
		log.Warn("删除活动失败: 无权操作此活动, activity_id=%d activity_org_id=%d user_org_id=%d user_id=%d", req.Id, activity.OrgID, org.ID, userID)
		return nil, errors.New("无权操作此活动")
	}

	// 校验活动状态
	if activity.Status == 2 {
		log.Warn("删除活动失败: 活动已结束, activity_id=%d user_id=%d", req.Id, userID)
		return nil, errors.New("已结束的活动不能删除")
	}

	// 检查是否有已报名的志愿者
	if activity.CurrentPeople > 0 {
		// 可以选择允许删除或拒绝
		// 这里选择允许删除，记录日志
		log.Warn("删除有报名人数的活动 activity_id=%d current_people=%d", activity.ID, activity.CurrentPeople)
	}

	if err := s.repo.DeleteActivity(s.repo.DB, req.Id); err != nil {
		log.Error("删除活动失败: 删除活动异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}

	log.Info("删除活动成功 activity_id=%d org_id=%d user_id=%d", req.Id, org.ID, userID)

	return &api.DeleteActivityResponse{
		Message: "删除活动成功",
	}, nil
}

// CancelActivity 取消活动
func (s *ActivityService) CancelActivity(req *api.CancelActivityRequest) (*api.CancelActivityResponse, error) {
	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Warn("取消活动失败: 获取当前用户失败: %v, activity_id=%d", err, req.Id)
		return nil, err
	}

	// 查询活动信息
	activity, err := s.repo.GetActivityByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("取消活动失败: 活动不存在, activity_id=%d user_id=%d", req.Id, userID)
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
		log.Warn("取消活动失败: 无权操作此活动, activity_id=%d activity_org_id=%d user_org_id=%d user_id=%d", req.Id, activity.OrgID, org.ID, userID)
		return nil, errors.New("无权操作此活动")
	}

	// 校验活动状态
	if activity.Status == 2 || activity.Status == 3 {
		log.Warn("取消活动失败: 活动状态不允许取消, activity_id=%d user_id=%d status=%d", req.Id, userID, activity.Status)
		return nil, errors.New("已结束或已取消的活动不能取消")
	}

	if err := s.repo.CancelActivity(s.repo.DB, req.Id); err != nil {
		log.Error("取消活动失败: 更新活动状态异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}

	log.Info("取消活动成功 activity_id=%d org_id=%d user_id=%d", req.Id, org.ID, userID)

	return &api.CancelActivityResponse{
		Message: "取消活动成功",
	}, nil
}
