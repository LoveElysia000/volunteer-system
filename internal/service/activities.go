package service

import (
	"context"
	"crypto/subtle"
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
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

var log = logger.GetLogger()

const (
	volunteerCheckoutEarliestWindow = 30 * time.Minute
	attendanceCodeLength            = 6
	attendanceCodeDigits            = "23456789"
	attendanceCodeLetters           = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	attendanceCodeCharset           = attendanceCodeDigits + attendanceCodeLetters
)

type ActivityService struct {
	Service
}

// NewActivityService 创建活动服务实例，并注入请求上下文与仓储依赖。
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

	// 获取当前账号身份，用于动态返回 is_registered。
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("活动列表查询失败: 获取当前用户ID异常: %v, status=%d page=%d page_size=%d", err, req.Status, req.Page, req.PageSize)
		return nil, err
	}

	signupMap := make(map[int64]*model.ActivitySignup)
	if len(activities) > 0 {
		account, findErr := s.repo.FindByID(s.repo.DB, userID)
		if findErr != nil {
			if errors.Is(findErr, gorm.ErrRecordNotFound) {
				log.Warn("活动列表查询失败: 账号不存在, user_id=%d", userID)
				return nil, errors.New("账号不存在")
			}
			log.Error("活动列表查询失败: 查询账号信息异常: %v, user_id=%d", findErr, userID)
			return nil, findErr
		}

		// 仅志愿者账号需要计算报名态；组织账号统一返回 false。
		if account.IdentityType == model.RegisterTypeVolunteerCode {
			volunteerID, getVolunteerErr := s.getVolunteerIDByAccountID(userID)
			if getVolunteerErr != nil {
				log.Error("活动列表查询失败: 查询志愿者身份异常: %v, user_id=%d", getVolunteerErr, userID)
				return nil, getVolunteerErr
			}

			activityIDs := make([]int64, 0, len(activities))
			for _, act := range activities {
				if act == nil {
					continue
				}
				activityIDs = append(activityIDs, act.ID)
			}

			signups, listErr := s.repo.ListUserSignupsByActivityIDs(s.repo.DB, volunteerID, activityIDs)
			if listErr != nil {
				log.Error("活动列表查询失败: 查询用户报名记录异常: %v, user_id=%d volunteer_id=%d", listErr, userID, volunteerID)
				return nil, listErr
			}
			for _, signup := range signups {
				if signup == nil {
					continue
				}
				signupMap[signup.ActivityID] = signup
			}
		}
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
			IsRegistered:  signupMap[act.ID] != nil,
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
		log.Error("活动报名失败: 获取当前用户ID异常: %v, activity_id=%d", err, req.ActivityId)
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
	record, err := buildPendingCreateAuditRecordByModel(
		model.AuditTargetSignup,
		userID,
		signupSnapshot,
		time.Now(),
	)
	if err != nil {
		log.Error("活动报名失败: 构建审核记录异常: %v, activity_id=%d user_id=%d volunteer_id=%d", err, req.ActivityId, userID, volunteerID)
		return nil, err
	}
	if err := s.repo.CreateAuditRecord(s.repo.DB, record); err != nil {
		log.Error("活动报名失败: 创建审核记录异常: %v, activity_id=%d user_id=%d volunteer_id=%d", err, req.ActivityId, userID, volunteerID)
		return nil, err
	}

	log.Info("活动报名申请已提交: activity_id=%d user_id=%d volunteer_id=%d record_id=%d", req.ActivityId, userID, volunteerID, record.ID)
	return &api.ActivitySignupResponse{Success: true}, nil
}

// hasPendingSignupCreateAudit 检查当前用户是否已有同活动同志愿者的待审核报名创建记录。
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

// ActivityCancel 取消报名
func (s *ActivityService) ActivityCancel(req *api.ActivityCancelRequest) (*api.ActivityCancelResponse, error) {
	// 获取当前用户ID
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("取消报名失败: 获取当前用户ID异常: %v, activity_id=%d", err, req.ActivityId)
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
		log.Error("活动详情查询失败: 获取当前用户ID异常: %v, activity_id=%d", err, req.Id)
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
		log.Error("我的活动列表查询失败: 获取当前用户ID异常: %v, status=%d page=%d page_size=%d", err, req.Status, req.Page, req.PageSize)
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

	// 提取活动ID与报名ID列表（去重）
	activityIDs := make([]int64, 0, len(signups))
	signupIDs := make([]int64, 0, len(signups))
	for _, signup := range signups {
		if signup == nil {
			continue
		}
		signupIDs = append(signupIDs, signup.ID)
		activityIDs = append(activityIDs, signup.ActivityID)
	}
	signupIDs = util.UniquePositiveInt64(signupIDs)
	activityIDs = util.UniquePositiveInt64(activityIDs)

	// 并发查询活动信息与报名驳回记录。
	var (
		activitiesList     []*model.Activity
		rejectAuditRecords []*model.AuditRecord
	)
	group, _ := errgroup.WithContext(s.ctx)
	group.SetLimit(2)
	group.Go(func() error {
		activities, queryErr := s.repo.ListActivitiesByIDs(s.repo.DB, activityIDs)
		if queryErr != nil {
			log.Error("我的活动列表查询失败: 批量查询活动异常: %v, user_id=%d volunteer_id=%d activity_count=%d", queryErr, userID, volunteerID, len(activityIDs))
			return queryErr
		}
		activitiesList = activities
		return nil
	})
	group.Go(func() error {
		rejectRecords, queryErr := s.repo.ListSignupRejectAuditRecords(s.repo.DB, signupIDs)
		if queryErr != nil {
			log.Error("我的活动列表查询失败: 查询报名驳回原因异常: %v, user_id=%d volunteer_id=%d signup_count=%d", queryErr, userID, volunteerID, len(signupIDs))
			return queryErr
		}
		rejectAuditRecords = rejectRecords
		return nil
	})
	if waitErr := group.Wait(); waitErr != nil {
		return nil, waitErr
	}

	activityMap := make(map[int64]*model.Activity, len(activitiesList))
	orgIDs := make([]int64, 0, len(activitiesList))
	for _, act := range activitiesList {
		if act == nil {
			continue
		}
		activityMap[act.ID] = act
		orgIDs = append(orgIDs, act.OrgID)
	}
	orgIDs = util.UniquePositiveInt64(orgIDs)

	// 批量获取组织名称
	orgNameMap := make(map[int64]string)
	organizations, queryErr := s.repo.ListOrganizationsByIDs(s.repo.DB, orgIDs)
	if queryErr != nil {
		log.Error("我的活动列表查询失败: 批量查询组织名称异常: %v, user_id=%d volunteer_id=%d org_count=%d", queryErr, userID, volunteerID, len(orgIDs))
		return nil, queryErr
	}
	for _, org := range organizations {
		if org == nil {
			continue
		}
		orgNameMap[org.ID] = org.OrgName
	}

	// 在 service 层按 signup_id 聚合最近一条驳回原因。
	auditReasonMap := make(map[int64]string, len(rejectAuditRecords))
	for _, record := range rejectAuditRecords {
		if record == nil || record.TargetID <= 0 {
			continue
		}
		if _, exists := auditReasonMap[record.TargetID]; exists {
			continue
		}
		auditReasonMap[record.TargetID] = record.RejectReason
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

		orgName := orgNameMap[activity.OrgID]

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
			SignupStatus:   signup.Status,
			AuditReason:    auditReasonMap[signup.ID],
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
		log.Error("创建活动失败: 获取当前用户ID异常: %v, org_id=%d", err, req.OrgId)
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
		log.Error("创建活动失败: 解析开始时间异常: %v, org_id=%d user_id=%d start_time=%s", err, req.OrgId, userID, req.StartTime)
		return nil, errors.New("开始时间格式错误")
	}
	endTime, err := time.Parse("2006-01-02 15:04:05", req.EndTime)
	if err != nil {
		log.Error("创建活动失败: 解析结束时间异常: %v, org_id=%d user_id=%d end_time=%s", err, req.OrgId, userID, req.EndTime)
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

	PublishNotificationEvent(NotificationEvent{
		EventType:   model.NotificationEventActivityCreated,
		BizType:     model.NotificationBizTypeActivity,
		BizID:       activity.ID,
		SourceOrgID: activity.OrgID,
		ActorID:     userID,
		CreatedAt:   time.Now(),
		Payload: map[string]any{
			"activityTitle": activity.Title,
			"startTime":     util.FormatDateTimeOrEmpty(activity.StartTime),
			"endTime":       util.FormatDateTimeOrEmpty(activity.EndTime),
		},
		DedupeKey: fmt.Sprintf("activity.created:%d", activity.ID),
	})

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
		log.Error("更新活动失败: 获取当前用户ID异常: %v, activity_id=%d", err, req.Id)
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
			log.Error("更新活动失败: 解析开始时间异常: %v, activity_id=%d user_id=%d start_time=%s", err, req.Id, userID, req.StartTime)
			return nil, errors.New("开始时间格式错误")
		}
		activity.StartTime = startTime
	}
	if req.EndTime != "" {
		endTime, err := time.Parse("2006-01-02 15:04:05", req.EndTime)
		if err != nil {
			log.Error("更新活动失败: 解析结束时间异常: %v, activity_id=%d user_id=%d end_time=%s", err, req.Id, userID, req.EndTime)
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
	updatedAt := activity.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}

	PublishNotificationEvent(NotificationEvent{
		EventType:   model.NotificationEventActivityUpdated,
		BizType:     model.NotificationBizTypeActivity,
		BizID:       activity.ID,
		SourceOrgID: activity.OrgID,
		ActorID:     userID,
		CreatedAt:   time.Now(),
		Payload: map[string]any{
			"activityTitle": activity.Title,
			"startTime":     util.FormatDateTimeOrEmpty(activity.StartTime),
			"endTime":       util.FormatDateTimeOrEmpty(activity.EndTime),
			"updatedAt":     util.FormatDateTimeOrEmpty(updatedAt),
		},
		// 使用业务更新时间生成稳定幂等键，避免 time.Now() 造成每次都视为新事件。
		DedupeKey: fmt.Sprintf("activity.updated:%d:%d", activity.ID, updatedAt.UnixNano()),
	})

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
		log.Error("删除活动失败: 获取当前用户ID异常: %v, activity_id=%d", err, req.Id)
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
		log.Error("取消活动失败: 获取当前用户ID异常: %v, activity_id=%d", err, req.Id)
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
		log.Error("完结活动失败: 获取当前用户ID异常: %v, activity_id=%d", err, req.Id)
		return nil, err
	}

	activity, err := s.ensureActivityOperableByCurrentOrg(req.Id, userID)
	if err != nil {
		log.Error("完结活动失败: 校验活动归属异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
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

// GenerateAttendanceCodes 生成签到码/签退码（组织侧，初次生成同时生成两个码）
func (s *ActivityService) GenerateAttendanceCodes(req *api.GenerateAttendanceCodesRequest) (*api.GenerateAttendanceCodesResponse, error) {
	if req.Id <= 0 {
		return nil, errors.New("活动ID不能为空")
	}
	if req.CheckInValidMinutes < 0 || req.CheckOutValidMinutes < 0 {
		return nil, errors.New("有效时长不能为负数")
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("生成签到签退码失败: 获取当前用户ID异常: %v, activity_id=%d", err, req.Id)
		return nil, err
	}

	activity, err := s.ensureActivityOperableByCurrentOrg(req.Id, userID)
	if err != nil {
		log.Error("生成签到签退码失败: 校验活动归属异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}
	if activity.Status == model.ActivityStatusCanceled {
		return nil, errors.New("已取消活动不能生成签到签退码")
	}
	if activity.Status == model.ActivityStatusFinished {
		return nil, errors.New("已结束活动不能生成签到签退码")
	}

	now := time.Now()
	checkInCode, checkInCodeHash, checkInExpireAt, err := generateAttendanceCodeForWrite(now, req.CheckInValidMinutes)
	if err != nil {
		log.Error("生成签到签退码失败: 生成签到码异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}
	checkOutCode, checkOutCodeHash, checkOutExpireAt, err := generateAttendanceCodeForWrite(now, req.CheckOutValidMinutes)
	if err != nil {
		log.Error("生成签到签退码失败: 生成签退码异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}
	// 初次生成会同时刷新两个码及对应过期时间，并统一推进版本号。
	updates := map[string]any{
		"check_in_code":              checkInCode,
		"check_in_code_hash":         checkInCodeHash,
		"check_in_code_expire_at":    checkInExpireAt,
		"check_out_code":             checkOutCode,
		"check_out_code_hash":        checkOutCodeHash,
		"check_out_code_expire_at":   checkOutExpireAt,
		"attendance_code_version":    gorm.Expr("attendance_code_version + 1"),
		"attendance_code_updated_at": now,
	}
	if err := s.repo.UpdateActivityAttendanceCodeByID(s.repo.DB, req.Id, updates); err != nil {
		log.Error("生成签到签退码失败: 更新活动码字段异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}

	codeInfo, err := s.repo.GetActivityByID(s.repo.DB, req.Id)
	if err != nil {
		log.Error("生成签到签退码失败: 查询活动码字段异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}

	resp := &api.GenerateAttendanceCodesResponse{
		Success:                 true,
		CheckInCode:             checkInCode,
		CheckOutCode:            checkOutCode,
		AttendanceCodeVersion:   codeInfo.AttendanceCodeVersion,
		AttendanceCodeUpdatedAt: util.FormatDateTimePtr(codeInfo.AttendanceCodeUpdatedAt),
		CheckInExpireAt:         util.FormatDateTimePtr(codeInfo.CheckInCodeExpireAt),
		CheckOutExpireAt:        util.FormatDateTimePtr(codeInfo.CheckOutCodeExpireAt),
	}

	log.Info("生成签到签退码成功: activity_id=%d user_id=%d version=%d", req.Id, userID, resp.AttendanceCodeVersion)
	return resp, nil
}

// ResetAttendanceCode 重置签到码/签退码（组织侧，单次仅重置一种码）
func (s *ActivityService) ResetAttendanceCode(req *api.ResetAttendanceCodeRequest) (*api.ResetAttendanceCodeResponse, error) {
	if req.Id <= 0 {
		return nil, errors.New("活动ID不能为空")
	}
	if !model.IsValidAttendanceCodeType(req.CodeType) {
		return nil, errors.New("重置码类型不合法")
	}
	if req.ValidMinutes < 0 {
		return nil, errors.New("有效时长不能为负数")
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("重置签到签退码失败: 获取当前用户ID异常: %v, activity_id=%d", err, req.Id)
		return nil, err
	}

	activity, err := s.ensureActivityOperableByCurrentOrg(req.Id, userID)
	if err != nil {
		log.Error("重置签到签退码失败: 校验活动归属异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}
	if activity.Status == model.ActivityStatusCanceled {
		return nil, errors.New("已取消活动不能重置签到签退码")
	}
	if activity.Status == model.ActivityStatusFinished {
		return nil, errors.New("已结束活动不能重置签到签退码")
	}

	now := time.Now()
	code, codeHash, expireAt, err := generateAttendanceCodeForWrite(now, req.ValidMinutes)
	if err != nil {
		log.Error("重置签到签退码失败: 生成随机码异常: %v, activity_id=%d user_id=%d code_type=%d", err, req.Id, userID, req.CodeType)
		return nil, err
	}

	updates := map[string]any{
		"attendance_code_version":    gorm.Expr("attendance_code_version + 1"),
		"attendance_code_updated_at": now,
	}
	// 单次只更新一种码，避免误覆盖另一种码的当前值与过期时间。
	switch req.CodeType {
	case model.AttendanceCodeTypeCheckIn:
		updates["check_in_code"] = code
		updates["check_in_code_hash"] = codeHash
		updates["check_in_code_expire_at"] = expireAt
	case model.AttendanceCodeTypeCheckOut:
		updates["check_out_code"] = code
		updates["check_out_code_hash"] = codeHash
		updates["check_out_code_expire_at"] = expireAt
	default:
		return nil, errors.New("重置码类型不合法")
	}

	if err := s.repo.UpdateActivityAttendanceCodeByID(s.repo.DB, req.Id, updates); err != nil {
		log.Error("重置签到签退码失败: 更新活动码字段异常: %v, activity_id=%d user_id=%d code_type=%d", err, req.Id, userID, req.CodeType)
		return nil, err
	}

	codeInfo, err := s.repo.GetActivityByID(s.repo.DB, req.Id)
	if err != nil {
		log.Error("重置签到签退码失败: 查询活动码字段异常: %v, activity_id=%d user_id=%d code_type=%d", err, req.Id, userID, req.CodeType)
		return nil, err
	}

	resp := &api.ResetAttendanceCodeResponse{
		Success:                 true,
		CodeType:                req.CodeType,
		Code:                    code,
		AttendanceCodeVersion:   codeInfo.AttendanceCodeVersion,
		AttendanceCodeUpdatedAt: util.FormatDateTimePtr(codeInfo.AttendanceCodeUpdatedAt),
	}
	// 返回被重置的码对应过期时间，方便前端直接展示。
	if req.CodeType == model.AttendanceCodeTypeCheckIn {
		resp.ExpireAt = util.FormatDateTimePtr(codeInfo.CheckInCodeExpireAt)
	} else {
		resp.ExpireAt = util.FormatDateTimePtr(codeInfo.CheckOutCodeExpireAt)
	}

	log.Info("重置签到签退码成功: activity_id=%d user_id=%d code_type=%d version=%d", req.Id, userID, req.CodeType, resp.AttendanceCodeVersion)
	return resp, nil
}

// GetActivityAttendanceCodes 查询活动签到码/签退码（组织侧）
func (s *ActivityService) GetActivityAttendanceCodes(req *api.GetActivityAttendanceCodesRequest) (*api.GetActivityAttendanceCodesResponse, error) {
	if req.Id <= 0 {
		return nil, errors.New("活动ID不能为空")
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("查询活动签到签退码失败: 获取当前用户ID异常: %v, activity_id=%d", err, req.Id)
		return nil, err
	}

	activity, err := s.ensureActivityOperableByCurrentOrg(req.Id, userID)
	if err != nil {
		log.Error("查询活动签到签退码失败: 校验活动归属异常: %v, activity_id=%d user_id=%d", err, req.Id, userID)
		return nil, err
	}

	resp := &api.GetActivityAttendanceCodesResponse{
		Success:                 true,
		CheckInCode:             activity.CheckInCode,
		CheckOutCode:            activity.CheckOutCode,
		CheckInExpireAt:         util.FormatDateTimePtr(activity.CheckInCodeExpireAt),
		CheckOutExpireAt:        util.FormatDateTimePtr(activity.CheckOutCodeExpireAt),
		AttendanceCodeVersion:   activity.AttendanceCodeVersion,
		AttendanceCodeUpdatedAt: util.FormatDateTimePtr(activity.AttendanceCodeUpdatedAt),
	}
	return resp, nil
}

// ActivityCheckIn 活动签到（志愿者侧）
func (s *ActivityService) ActivityCheckIn(req *api.ActivityCheckInRequest) (*api.ActivityCheckInResponse, error) {
	if req.ActivityId <= 0 {
		return nil, errors.New("活动ID不能为空")
	}
	if strings.TrimSpace(req.CheckInCode) == "" {
		return nil, errors.New("签到码不能为空")
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("活动签到失败: 获取当前用户ID异常: %v, activity_id=%d", err, req.ActivityId)
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
		log.Error("活动签到失败: 查询活动异常: %v, activity_id=%d user_id=%d", err, req.ActivityId, userID)
		return nil, err
	}
	if activity.Status == model.ActivityStatusCanceled {
		return nil, errors.New("已取消活动不允许签到")
	}
	if err := validateAttendanceCodeForActivity(activity, req.CheckInCode, model.AttendanceCodeTypeCheckIn); err != nil {
		log.Error("活动签到失败: 校验签到码异常: %v, activity_id=%d user_id=%d volunteer_id=%d", err, req.ActivityId, userID, volunteerID)
		return nil, err
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
	if strings.TrimSpace(req.CheckOutCode) == "" {
		return nil, errors.New("签退码不能为空")
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("活动签退失败: 获取当前用户ID异常: %v, activity_id=%d", err, req.ActivityId)
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
		log.Error("活动签退失败: 查询活动异常: %v, activity_id=%d user_id=%d", err, req.ActivityId, userID)
		return nil, err
	}
	if activity.Status == model.ActivityStatusCanceled {
		return nil, errors.New("已取消活动不允许签退")
	}
	if err := validateAttendanceCodeForActivity(activity, req.CheckOutCode, model.AttendanceCodeTypeCheckOut); err != nil {
		log.Error("活动签退失败: 校验签退码异常: %v, activity_id=%d user_id=%d volunteer_id=%d", err, req.ActivityId, userID, volunteerID)
		return nil, err
	}

	var checkOutTime time.Time
	var grantedHours float64
	// 事务外预检查：尽早失败，减少不必要的行锁持有时间。
	// Fast-fail before entering transaction to shorten lock hold time.
	preSignup, err := s.repo.GetSignup(s.repo.DB, req.ActivityId, volunteerID)
	if err != nil {
		log.Error("活动签退失败: 预检查报名记录异常: %v, activity_id=%d user_id=%d volunteer_id=%d", err, req.ActivityId, userID, volunteerID)
		return nil, err
	}

	if preSignup == nil {
		return nil, errors.New("报名记录不存在")
	}
	if preSignup.Status != model.ActivitySignupStatusSuccess {
		return nil, errors.New("当前报名状态不允许签退")
	}
	if preSignup.CheckInStatus != model.ActivityCheckInDone || preSignup.CheckInTime == nil {
		return nil, errors.New("未签到，无法签退")
	}

	// 签退时间校验
	now := time.Now()
	if now.Before(activity.EndTime.Add(-volunteerCheckoutEarliestWindow)) {
		return nil, errors.New("未到签退开始时间，还不能签退")
	}

	// Re-check under row lock to prevent races between pre-check and final write.
	err = s.withTransaction(func(tx *gorm.DB) error {
		// 事务内二次校验：防止预检查与加锁之间的并发状态变化。
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

		// Idempotent return for repeated checkout requests.
		if signup.CheckOutStatus == model.ActivityCheckOutDone {
			if signup.CheckOutTime != nil {
				checkOutTime = *signup.CheckOutTime
			}
			grantedHours = signup.GrantedHours
			return nil
		}

		now = time.Now()
		if now.Before(*signup.CheckInTime) {
			now = *signup.CheckInTime
		}
		checkOutTime = now
		nextVersion := signup.WorkHourVersion + 1
		idempotencyKey := fmt.Sprintf("checkout:%d:%d", signup.ID, nextVersion)
		grantedHours, err = s.settleSignupWorkHours(
			tx,
			activity,
			signup,
			*signup.CheckInTime,
			checkOutTime,
			userID,
			idempotencyKey,
			"签到签退自动结算",
			false,
		)
		return err
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
		log.Error("活动补录失败: 解析签退时间异常: %v, activity_id=%d volunteer_id=%d check_out_time=%s", err, req.ActivityId, req.VolunteerId, checkOutText)
		return nil, errors.New("签退时间格式错误")
	}

	checkInText := strings.TrimSpace(req.CheckInTime)
	var checkInAt time.Time
	hasCheckInInput := false
	if checkInText != "" {
		checkInAt, err = util.ParseDateTime(checkInText)
		if err != nil {
			log.Error("活动补录失败: 解析签到时间异常: %v, activity_id=%d volunteer_id=%d check_in_time=%s", err, req.ActivityId, req.VolunteerId, checkInText)
			return nil, errors.New("签到时间格式错误")
		}
		hasCheckInInput = true
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("活动补录失败: 获取当前用户ID异常: %v, activity_id=%d volunteer_id=%d", err, req.ActivityId, req.VolunteerId)
		return nil, err
	}

	activity, err := s.ensureActivityOperableByCurrentOrg(req.ActivityId, userID)
	if err != nil {
		log.Error("活动补录失败: 校验活动归属异常: %v, activity_id=%d volunteer_id=%d user_id=%d", err, req.ActivityId, req.VolunteerId, userID)
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

	// Keep signup status, work-hour log and volunteer aggregate in one atomic transaction.
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
		// Idempotent branch: already checked out means we return existing settlement result.
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
		nextVersion := signup.WorkHourVersion + 1
		idempotencyKey := fmt.Sprintf("org-supplement:%d:%d:%d", signup.ID, nextVersion, finalCheckOut.Unix())
		grantedHours, err = s.settleSignupWorkHours(
			tx,
			activity,
			signup,
			finalCheckIn,
			finalCheckOut,
			userID,
			idempotencyKey,
			reason,
			true,
		)
		return err
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

	org, err := s.repo.GetOrganizationByAccountID(s.repo.DB, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("组织信息不存在")
		}
		log.Error("校验活动归属失败: 查询组织异常: %v, activity_id=%d account_id=%d", err, activityID, accountID)
		return nil, err
	}

	if activity.OrgID != org.ID {
		return nil, errors.New("无权操作此活动")
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

// validateAttendanceCodeForActivity 根据码类型校验活动签到码或签退码是否匹配且未过期。
func validateAttendanceCodeForActivity(activity *model.Activity, inputCode string, codeType int32) error {
	if activity == nil {
		return errors.New("活动不存在")
	}

	switch codeType {
	case model.AttendanceCodeTypeCheckIn:
		return validateAttendanceCodeValue(
			inputCode,
			activity.CheckInCode,
			activity.CheckInCodeHash,
			activity.CheckInCodeExpireAt,
			"签到码错误或已过期",
		)
	case model.AttendanceCodeTypeCheckOut:
		return validateAttendanceCodeValue(
			inputCode,
			activity.CheckOutCode,
			activity.CheckOutCodeHash,
			activity.CheckOutCodeExpireAt,
			"签退码错误或已过期",
		)
	default:
		return errors.New("码类型不合法")
	}
}

// validateAttendanceCodeValue 统一处理码存在性、过期性与值匹配校验。
func validateAttendanceCodeValue(inputCode, expectedCode, expectedCodeHash string, expireAt *time.Time, errMsg string) error {
	normalizedInputCode := strings.TrimSpace(inputCode)
	normalizedExpectedCode := strings.TrimSpace(expectedCode)
	normalizedExpectedCodeHash := strings.TrimSpace(expectedCodeHash)
	if expireAt != nil && time.Now().After(*expireAt) {
		return errors.New(errMsg)
	}

	// 优先使用哈希字段校验；保留明文字段回退以兼容历史数据。
	if normalizedExpectedCodeHash != "" {
		// Prefer hashed comparison when hash exists; plaintext fallback is for legacy rows only.
		inputCodeHash, err := util.HashSensitiveField(normalizedInputCode)
		if err != nil {
			return err
		}
		if subtle.ConstantTimeCompare([]byte(inputCodeHash), []byte(normalizedExpectedCodeHash)) != 1 {
			return errors.New(errMsg)
		}
		return nil
	}

	if normalizedExpectedCode == "" {
		return errors.New(errMsg)
	}
	if subtle.ConstantTimeCompare([]byte(normalizedInputCode), []byte(normalizedExpectedCode)) != 1 {
		return errors.New(errMsg)
	}
	return nil
}

// generateAttendanceCodeForWrite 生成随机码、哈希值和过期时间，用于写入活动码字段。
func generateAttendanceCodeForWrite(now time.Time, validMinutes int32) (string, string, *time.Time, error) {
	code, err := generateRandomAttendanceCode(attendanceCodeLength)
	if err != nil {
		return "", "", nil, err
	}
	codeHash, err := util.HashSensitiveField(strings.TrimSpace(code))
	if err != nil {
		return "", "", nil, err
	}
	return code, codeHash, buildAttendanceCodeExpireAt(now, validMinutes), nil
}

// buildAttendanceCodeExpireAt 根据有效分钟数计算过期时间；<=0 表示不过期。
func buildAttendanceCodeExpireAt(now time.Time, validMinutes int32) *time.Time {
	if validMinutes <= 0 {
		return nil
	}
	expireAt := now.Add(time.Duration(validMinutes) * time.Minute)
	return &expireAt
}

// generateRandomAttendanceCode 生成固定长度码，且至少包含 1 位数字与 1 位字母。

func generateRandomAttendanceCode(length int) (string, error) {
	if length < 2 {
		return "", errors.New("无效的签到签退码长度")
	}

	digitPos, err := util.RandomIndex(length)
	if err != nil {
		return "", err
	}
	letterPos, err := util.RandomIndex(length)
	if err != nil {
		return "", err
	}
	for letterPos == digitPos {
		letterPos, err = util.RandomIndex(length)
		if err != nil {
			return "", err
		}
	}

	chars := []byte(attendanceCodeCharset)
	digits := []byte(attendanceCodeDigits)
	letters := []byte(attendanceCodeLetters)

	result := make([]byte, length)
	// 先固定一个数字位和一个字母位，确保复杂度下限。
	digitIdx, err := util.RandomIndex(len(digits))
	if err != nil {
		return "", err
	}
	letterIdx, err := util.RandomIndex(len(letters))
	if err != nil {
		return "", err
	}
	result[digitPos] = digits[digitIdx]
	result[letterPos] = letters[letterIdx]

	for i := range result {
		if i == digitPos || i == letterPos {
			continue
		}
		charIdx, idxErr := util.RandomIndex(len(chars))
		if idxErr != nil {
			return "", idxErr
		}
		result[i] = chars[charIdx]
	}
	return string(result), nil
}
