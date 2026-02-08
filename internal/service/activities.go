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

	// 获取db
	db := s.repo.DB

	// 查询活动列表
	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	activities, total, err := s.repo.GetActivitiesByStatus(db, req.Status, pageSize, offset)
	if err != nil {
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

	// 获取db
	db := s.repo.DB

	// 查询活动信息
	activity, err := s.repo.GetActivityByID(db, req.ActivityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		return nil, err
	}

	// 校验活动状态
	if activity.Status != 1 {
		return nil, errors.New("活动已结束或已取消")
	}

	// 校验名额
	if activity.MaxPeople > 0 && activity.CurrentPeople >= activity.MaxPeople {
		return nil, errors.New("名额已满")
	}

	// 校验是否重复报名
	existing, _ := s.repo.GetSignup(db, req.ActivityId, userID)
	if existing != nil && existing.Status == 1 {
		return nil, errors.New("请勿重复报名")
	}

	// 事务处理
	err = db.Transaction(func(tx *gorm.DB) error {
		// 创建报名记录
		signup := &model.ActivitySignup{
			ActivityID:  req.ActivityId,
			VolunteerID: userID,
			Status:      1,
		}
		if err := s.repo.CreateSignup(tx, signup); err != nil {
			return err
		}

		// 增加活动当前报名人数（原子操作）
		if err := s.repo.IncrementActivityPeople(tx, req.ActivityId); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("名额已满")
			}
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &api.ActivitySignupResponse{Success: true}, nil
}

// ActivityCancel 取消报名
func (s *ActivityService) ActivityCancel(req *api.ActivityCancelRequest) (*api.ActivityCancelResponse, error) {
	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	// 获取db
	db := s.repo.DB

	// 查询报名记录
	signup, err := s.repo.GetSignup(db, req.ActivityId, userID)
	if err != nil {
		return nil, err
	}

	// 校验报名记录是否存在
	if signup == nil {
		return nil, errors.New("报名记录不存在")
	}

	// 校验报名状态
	if signup.Status != 1 {
		return nil, errors.New("当前状态不允许取消")
	}

	// 事务处理
	err = db.Transaction(func(tx *gorm.DB) error {
		// 更新报名状态为已取消
		signup.Status = 2
		if err := s.repo.UpdateSignupStatus(tx, signup); err != nil {
			return err
		}

		// 减少活动当前报名人数（原子操作）
		if err := s.repo.DecrementActivityPeople(tx, req.ActivityId); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &api.ActivityCancelResponse{Success: true}, nil
}

// ActivityDetail 获取活动详情
func (s *ActivityService) ActivityDetail(req *api.ActivityDetailRequest) (*api.ActivityDetailResponse, error) {
	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	// 获取db
	db := s.repo.DB

	// 查询活动信息及组织名称
	activity, orgName, err := s.repo.GetActivityWithOrg(db, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		return nil, err
	}

	// 查询用户是否已报名
	signup, _ := s.repo.GetSignup(db, req.Id, userID)
	isRegistered := signup != nil && signup.Status == 1

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
		return nil, err
	}

	// 获取db
	db := s.repo.DB

	// 查询我的报名记录
	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	signups, total, err := s.repo.GetMyActivities(db, userID, req.Status, pageSize, offset)
	if err != nil {
		return nil, err
	}

	// 提取活动ID列表
	activityIDs := make([]int64, 0, len(signups))
	for _, signup := range signups {
		activityIDs = append(activityIDs, signup.ActivityID)
	}

	// 批量获取活动信息
	activityMap, err := s.repo.GetActivitiesByIDs(db, activityIDs)
	if err != nil {
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
	orgNameMap, err := s.repo.GetOrgNamesByIDs(db, orgIDs)
	if err != nil {
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
		return nil, err
	}

	// 校验必填字段
	if req.OrgId <= 0 {
		return nil, errors.New("org_id 不能为空")
	}

	// 获取db
	db := s.repo.DB

	// 根据传入的 org_id 查询组织信息
	org, err := s.repo.GetOrganizationByID(db, req.OrgId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("组织不存在")
		}
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
		Status:        1, // 1-报名中
	}

	if err := s.repo.CreateActivity(db, activity); err != nil {
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
		return nil, err
	}

	// 获取db
	db := s.repo.DB

	// 查询活动信息
	activity, err := s.repo.GetActivityByID(db, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		return nil, err
	}

	// 查询组织信息
	org, err := s.repo.GetOrganizationByAccountID(db, userID)
	if err != nil {
		return nil, errors.New("组织信息不存在")
	}

	// 校验活动归属
	if activity.OrgID != org.ID {
		return nil, errors.New("无权操作此活动")
	}

	// 校验活动状态
	if activity.Status == 2 || activity.Status == 3 {
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

	if err := s.repo.UpdateActivity(db, activity); err != nil {
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
		return nil, err
	}

	// 获取db
	db := s.repo.DB

	// 查询活动信息
	activity, err := s.repo.GetActivityByID(db, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		return nil, err
	}

	// 查询组织信息
	org, err := s.repo.GetOrganizationByAccountID(db, userID)
	if err != nil {
		return nil, errors.New("组织信息不存在")
	}

	// 校验活动归属
	if activity.OrgID != org.ID {
		return nil, errors.New("无权操作此活动")
	}

	// 校验活动状态
	if activity.Status == 2 {
		return nil, errors.New("已结束的活动不能删除")
	}

	// 检查是否有已报名的志愿者
	if activity.CurrentPeople > 0 {
		// 可以选择允许删除或拒绝
		// 这里选择允许删除，记录日志
		log.Warn("删除有报名人数的活动 activity_id=%d current_people=%d", activity.ID, activity.CurrentPeople)
	}

	if err := s.repo.DeleteActivity(db, req.Id); err != nil {
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
		return nil, err
	}

	// 获取db
	db := s.repo.DB

	// 查询活动信息
	activity, err := s.repo.GetActivityByID(db, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		return nil, err
	}

	// 查询组织信息
	org, err := s.repo.GetOrganizationByAccountID(db, userID)
	if err != nil {
		return nil, errors.New("组织信息不存在")
	}

	// 校验活动归属
	if activity.OrgID != org.ID {
		return nil, errors.New("无权操作此活动")
	}

	// 校验活动状态
	if activity.Status == 2 || activity.Status == 3 {
		return nil, errors.New("已结束或已取消的活动不能取消")
	}

	if err := s.repo.CancelActivity(db, req.Id); err != nil {
		return nil, err
	}

	log.Info("取消活动成功 activity_id=%d reason=%s org_id=%d user_id=%d", req.Id, req.Reason, org.ID, userID)

	return &api.CancelActivityResponse{
		Message: "取消活动成功",
	}, nil
}
