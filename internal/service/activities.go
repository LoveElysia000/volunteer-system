package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/logger"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

var log = logger.GetLogger()

const (
	volunteerCheckoutEarliestWindow = 30 * time.Minute
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

// ========== 志愿者端：活动浏览与报名 ==========

// ActivityList 获取活动列表（活动总览）
func (s *ActivityService) ActivityList(req *api.ActivityListRequest) (*api.ActivityListResponse, error) {
	if req == nil {
		req = &api.ActivityListRequest{}
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}

	// 先校验当前账号上下文与账号存在性，避免被后续“空结果快速返回”分支绕过。
	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("活动列表查询失败: 获取当前账户ID异常: %v, status=%v page=%d page_size=%d", err, req.Status, req.Page, req.PageSize)
		return nil, err
	}

	account, findErr := s.repo.FindByID(s.repo.DB, accountID)
	if findErr != nil {
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			log.Warn("活动列表查询失败: 账号不存在, account_id=%d", accountID)
			return nil, errors.New("账号不存在")
		}
		log.Error("活动列表查询失败: 查询账号信息异常: %v, account_id=%d", findErr, accountID)
		return nil, findErr
	}

	// 构建查询map
	actMap, err := buildActivityListFilterMap(req)
	if err != nil {
		return nil, err
	}

	volunteerID, isVolunteer, err := s.resolveActivityListVolunteerID(accountID, account)
	if err != nil {
		log.Error("活动列表查询失败: 查询志愿者身份异常: %v, account_id=%d", err, accountID)
		return nil, err
	}
	if req.RegisteredOnly && !isVolunteer {
		return &api.ActivityListResponse{
			Total: 0,
			List:  []*api.ActivityItem{},
		}, nil
	}

	if ok, filterErr := s.applyActivityListScopeFilters(req, accountID, volunteerID, actMap); filterErr != nil {
		return nil, filterErr
	} else if !ok {
		return &api.ActivityListResponse{
			Total: 0,
			List:  []*api.ActivityItem{},
		}, nil
	}

	// 查询活动列表
	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	activities, total, err := s.repo.GetActivitiesByFilters(s.repo.DB, actMap, pageSize, offset)
	if err != nil {
		log.Error("活动列表查询失败: %v, status=%v page=%d page_size=%d", err, req.Status, req.Page, req.PageSize)
		return nil, err
	}

	signupMap := make(map[int64]*model.ActivitySignup)
	// 仅志愿者账号需要计算报名态；组织账号统一返回 false。
	if isVolunteer && len(activities) > 0 {
		signupMap, err = s.loadActivitySignupMap(accountID, volunteerID, activities)
		if err != nil {
			return nil, err
		}
	}

	// 组装返回数据
	resp := &api.ActivityListResponse{
		Total: int32(total),
		List:  make([]*api.ActivityItem, 0, len(activities)),
	}

	for _, act := range activities {
		if act == nil {
			continue
		}
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

// MyActivityList 获取当前志愿者的活动列表。
func (s *ActivityService) MyActivityList(req *api.MyActivityListRequest) (*api.ActivityListResponse, error) {
	if req == nil {
		req = &api.MyActivityListRequest{}
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}

	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("我的活动列表查询失败: 获取当前账户ID异常: %v, status=%v page=%d page_size=%d", err, req.Status, req.Page, req.PageSize)
		return nil, err
	}

	account, findErr := s.repo.FindByID(s.repo.DB, accountID)
	if findErr != nil {
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			log.Warn("我的活动列表查询失败: 账号不存在, account_id=%d", accountID)
			return nil, errors.New("账号不存在")
		}
		log.Error("我的活动列表查询失败: 查询账号信息异常: %v, account_id=%d", findErr, accountID)
		return nil, findErr
	}

	volunteerID, isVolunteer, err := s.resolveActivityListVolunteerID(accountID, account)
	if err != nil {
		log.Error("我的活动列表查询失败: 查询志愿者身份异常: %v, account_id=%d", err, accountID)
		return nil, err
	}
	if !isVolunteer {
		return &api.ActivityListResponse{
			Total: 0,
			List:  []*api.ActivityItem{},
		}, nil
	}

	actMap := map[string]any{}
	if len(req.Status) > 0 {
		signupStatuses := make([]int32, 0, len(req.Status))
		seenStatuses := make(map[int32]struct{}, len(req.Status))
		for _, status := range req.Status {
			if status != model.ActivitySignupStatusPending &&
				status != model.ActivitySignupStatusSuccess &&
				status != model.ActivitySignupStatusRejected &&
				status != model.ActivitySignupStatusCanceled {
				return nil, errors.New("报名状态不合法")
			}
			if _, ok := seenStatuses[status]; ok {
				continue
			}
			seenStatuses[status] = struct{}{}
			signupStatuses = append(signupStatuses, status)
		}
		req.Status = signupStatuses
	}

	var startFrom *time.Time
	if strings.TrimSpace(req.StartFrom) != "" {
		value, parseErr := util.ParseDateTime(strings.TrimSpace(req.StartFrom))
		if parseErr != nil {
			return nil, errors.New("开始时间格式错误")
		}
		startFrom = &value
		actMap["act.start_time >= ?"] = startFrom
	}

	var startTo *time.Time
	if strings.TrimSpace(req.StartTo) != "" {
		value, parseErr := util.ParseDateTime(strings.TrimSpace(req.StartTo))
		if parseErr != nil {
			return nil, errors.New("结束时间格式错误")
		}
		startTo = &value
		actMap["act.start_time <= ?"] = startTo
	}
	if startFrom != nil && startTo != nil && startTo.Before(*startFrom) {
		return nil, errors.New("结束时间不能早于开始时间")
	}

	keyword := strings.TrimSpace(req.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		actMap["(act.title LIKE ? OR act.description LIKE ? OR act.location LIKE ?)"] = []any{like, like, like}
	}

	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	activities, signupStatusMap, total, err := s.repo.GetMyActivitiesByFilters(s.repo.DB, volunteerID, actMap, req.Status, pageSize, offset)
	if err != nil {
		log.Error("我的活动列表查询失败: %v, status=%v page=%d page_size=%d", err, req.Status, req.Page, req.PageSize)
		return nil, err
	}

	resp := &api.ActivityListResponse{
		Total: int32(total),
		List:  make([]*api.ActivityItem, 0, len(activities)),
	}
	for _, act := range activities {
		if act == nil {
			continue
		}
		resp.List = append(resp.List, &api.ActivityItem{
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
			IsRegistered:  true,
			SignupStatus:  signupStatusMap[act.ID],
			IsFull:        act.MaxPeople > 0 && act.CurrentPeople >= act.MaxPeople,
		})
	}

	return resp, nil
}

func (s *ActivityService) resolveActivityListVolunteerID(accountID int64, account *model.SysAccount) (int64, bool, error) {
	if account == nil || account.IdentityType != model.RegisterTypeVolunteerCode {
		return 0, false, nil
	}

	volunteerID, err := s.getVolunteerIDByAccountID(accountID)
	if err != nil {
		return 0, false, err
	}
	return volunteerID, true, nil
}

func (s *ActivityService) applyActivityListScopeFilters(
	req *api.ActivityListRequest,
	accountID, volunteerID int64,
	actMap map[string]any,
) (bool, error) {
	if req.RegisteredOnly {
		signups, err := s.repo.ListVisibleSignupsByVolunteer(s.repo.DB, volunteerID)
		if err != nil {
			log.Error("活动列表查询失败: 查询用户报名记录异常: %v, account_id=%d volunteer_id=%d", err, accountID, volunteerID)
			return false, err
		}
		if ok := mergeActivityIDConstraint(actMap, collectRegisteredActivityIDs(signups)); !ok {
			return false, nil
		}
	}

	keyword := strings.TrimSpace(req.Keyword)
	if keyword == "" {
		return true, nil
	}

	activityIDs, err := s.repo.ListActivityIDsByKeyword(s.repo.DB, keyword)
	if err != nil {
		log.Error("活动列表查询失败: 关键字查询活动ID异常: %v, keyword=%s", err, keyword)
		return false, err
	}
	if ok := mergeActivityIDConstraint(actMap, activityIDs); !ok {
		return false, nil
	}
	return true, nil
}

func (s *ActivityService) loadActivitySignupMap(
	accountID, volunteerID int64,
	activities []*model.Activity,
) (map[int64]*model.ActivitySignup, error) {
	activityIDs := make([]int64, 0, len(activities))
	for _, act := range activities {
		if act == nil {
			continue
		}
		activityIDs = append(activityIDs, act.ID)
	}

	signups, err := s.repo.ListUserSignupsByActivityIDs(s.repo.DB, volunteerID, activityIDs)
	if err != nil {
		log.Error("活动列表查询失败: 查询用户报名记录异常: %v, account_id=%d volunteer_id=%d", err, accountID, volunteerID)
		return nil, err
	}

	signupMap := make(map[int64]*model.ActivitySignup, len(signups))
	for _, signup := range signups {
		if signup == nil {
			continue
		}
		signupMap[signup.ActivityID] = signup
	}
	return signupMap, nil
}

// ActivitySignup 活动报名
func (s *ActivityService) ActivitySignup(req *api.ActivitySignupRequest) (*api.ActivitySignupResponse, error) {
	// 获取当前账号ID
	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("活动报名失败: 获取当前账户ID异常: %v, activity_id=%d", err, req.ActivityId)
		return nil, err
	}
	volunteerID, err := s.getVolunteerIDByAccountID(accountID)
	if err != nil {
		log.Error("活动报名失败: 查询志愿者身份异常: %v, account_id=%d", err, accountID)
		return nil, err
	}

	var recordID int64
	err = s.withTransaction(func(tx *gorm.DB) error {
		// 锁定活动，串行化同活动报名请求，避免并发穿透重复校验。
		activity, activityErr := s.repo.GetActivityByIDForUpdate(tx, req.ActivityId)
		if activityErr != nil {
			if errors.Is(activityErr, gorm.ErrRecordNotFound) {
				return errors.New("活动不存在")
			}
			return activityErr
		}

		existing, signupErr := s.repo.GetSignup(tx, req.ActivityId, volunteerID)
		if signupErr != nil {
			return signupErr
		}

		hasPendingAudit, pendingErr := s.hasPendingSignupCreateAudit(tx, req.ActivityId, volunteerID, accountID)
		if pendingErr != nil {
			return pendingErr
		}
		if canErr := canSignupActivity(activity, existing, hasPendingAudit); canErr != nil {
			return canErr
		}

		signupSnapshot := &model.ActivitySignup{
			ActivityID:  req.ActivityId,
			VolunteerID: volunteerID,
			Status:      model.ActivitySignupStatusPending,
		}
		record, buildErr := buildPendingCreateAuditRecordByModel(
			model.AuditTargetSignup,
			accountID,
			signupSnapshot,
			time.Now(),
		)
		if buildErr != nil {
			return buildErr
		}
		if createErr := s.repo.CreateAuditRecord(tx, record); createErr != nil {
			return createErr
		}
		recordID = record.ID
		return nil
	})
	if err != nil {
		log.Error("活动报名失败: %v, activity_id=%d account_id=%d volunteer_id=%d", err, req.ActivityId, accountID, volunteerID)
		return nil, err
	}

	log.Info("活动报名申请已提交: activity_id=%d account_id=%d volunteer_id=%d record_id=%d", req.ActivityId, accountID, volunteerID, recordID)
	return &api.ActivitySignupResponse{Success: true}, nil
}

// ActivityCancel 取消报名
func (s *ActivityService) ActivityCancel(req *api.ActivityCancelRequest) (*api.ActivityCancelResponse, error) {
	// 获取当前账号ID
	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("取消报名失败: 获取当前账户ID异常: %v, activity_id=%d", err, req.ActivityId)
		return nil, err
	}
	volunteerID, err := s.getVolunteerIDByAccountID(accountID)
	if err != nil {
		log.Error("取消报名失败: 查询志愿者身份异常: %v, account_id=%d", err, accountID)
		return nil, err
	}

	var signupID int64
	// 事务处理：在行锁下读取并判断，避免并发重复取消导致的人数重复扣减。
	err = s.repo.DB.Transaction(func(tx *gorm.DB) error {
		signup, getErr := s.repo.GetSignupForUpdate(tx, req.ActivityId, volunteerID)
		if getErr != nil {
			return getErr
		}
		decision, transitionErr := resolveSignupTransition(signupTransitionCancel, signup)
		if transitionErr != nil {
			return transitionErr
		}
		signupID = signup.ID

		if !decision.apply {
			return nil
		}
		if err := s.repo.UpdateActivitySignupStatusByID(tx, signup.ID, decision.toStatus); err != nil {
			log.Error("取消报名失败: 更新报名状态异常: %v, activity_id=%d account_id=%d volunteer_id=%d signup_id=%d", err, req.ActivityId, accountID, volunteerID, signup.ID)
			return err
		}

		if decision.peopleDelta < 0 {
			if err := s.repo.DecrementActivityPeople(tx, req.ActivityId); err != nil {
				log.Error("取消报名失败: 减少活动人数异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
				return err
			}
		}

		return nil
	})

	if err != nil {
		log.Error("取消报名失败: 事务执行失败: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
		return nil, err
	}

	log.Info("取消报名成功: activity_id=%d account_id=%d volunteer_id=%d signup_id=%d", req.ActivityId, accountID, volunteerID, signupID)
	return &api.ActivityCancelResponse{Success: true}, nil
}

// ActivityDetail 获取活动详情
func (s *ActivityService) ActivityDetail(req *api.ActivityDetailRequest) (*api.ActivityDetailResponse, error) {
	// 获取当前账号ID
	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("活动详情查询失败: 获取当前账户ID异常: %v, activity_id=%d", err, req.ActivityId)
		return nil, err
	}
	account, findErr := s.repo.FindByID(s.repo.DB, accountID)
	if findErr != nil {
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			log.Warn("活动详情查询失败: 账号不存在, account_id=%d", accountID)
			return nil, errors.New("账号不存在")
		}
		log.Error("活动详情查询失败: 查询账号信息异常: %v, account_id=%d", findErr, accountID)
		return nil, findErr
	}
	// 查询活动信息及组织名称
	activity, orgName, err := s.repo.GetActivityWithOrg(s.repo.DB, req.ActivityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		log.Error("活动详情查询失败: 查询活动异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
		return nil, err
	}

	isRegistered := false
	checkInStatus := model.ActivityCheckInPending
	checkInTime := util.FormatDateTimePtr(nil)
	checkOutStatus := model.ActivityCheckOutPending
	checkOutTime := util.FormatDateTimePtr(nil)
	workHourStatus := model.WorkHourStatusPending
	grantedHours := float64(0)

	if account.IdentityType == model.RegisterTypeVolunteerCode {
		volunteerID, getVolunteerErr := s.getVolunteerIDByAccountID(accountID)
		if getVolunteerErr != nil {
			log.Error("活动详情查询失败: 查询志愿者身份异常: %v, account_id=%d", getVolunteerErr, accountID)
			return nil, getVolunteerErr
		}
		signup, signupErr := s.repo.GetSignup(s.repo.DB, req.ActivityId, volunteerID)
		if signupErr != nil {
			log.Error("活动详情查询失败: 查询报名记录异常: %v, activity_id=%d account_id=%d volunteer_id=%d", signupErr, req.ActivityId, accountID, volunteerID)
			return nil, signupErr
		}
		if signup != nil {
			if signup.Status == model.ActivitySignupStatusPending || signup.Status == model.ActivitySignupStatusSuccess {
				isRegistered = true
				checkInStatus = signup.CheckInStatus
				checkInTime = util.FormatDateTimePtr(signup.CheckInTime)
				checkOutStatus = signup.CheckOutStatus
				checkOutTime = util.FormatDateTimePtr(signup.CheckOutTime)
				workHourStatus = signup.WorkHourStatus
				grantedHours = signup.GrantedHours
			}
		}
		if !isRegistered {
			pendingAudit, auditErr := s.hasPendingSignupCreateAudit(s.repo.DB, req.ActivityId, volunteerID, accountID)
			if auditErr != nil {
				log.Error("活动详情查询失败: 查询待审核报名异常: %v, activity_id=%d account_id=%d volunteer_id=%d", auditErr, req.ActivityId, accountID, volunteerID)
				return nil, auditErr
			}
			isRegistered = pendingAudit
		}
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
			IsRegistered:   isRegistered,
			CreatedAt:      util.FormatDateTimeOrEmpty(activity.CreatedAt),
			CheckInStatus:  checkInStatus,
			CheckInTime:    checkInTime,
			CheckOutStatus: checkOutStatus,
			CheckOutTime:   checkOutTime,
			WorkHourStatus: workHourStatus,
			GrantedHours:   grantedHours,
		},
	}

	return resp, nil
}

// ========== 组织端：活动基础管理 ==========

// CreateActivity 创建活动
func (s *ActivityService) CreateActivity(req *api.CreateActivityRequest) (*api.CreateActivityResponse, error) {
	// 获取当前账号ID
	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("创建活动失败: 获取当前账户ID异常: %v, org_id=%d", err, req.OrgId)
		return nil, err
	}

	// 校验必填字段
	if req.OrgId <= 0 {
		return nil, errors.New("组织ID不能为空")
	}

	// 根据传入的 org_id 查询组织信息
	if _, err := s.repo.GetOrganizationByID(s.repo.DB, req.OrgId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("组织不存在")
		}
		log.Error("创建活动失败: 查询组织异常: %v, org_id=%d account_id=%d", err, req.OrgId, accountID)
		return nil, err
	}

	if err := s.requireOrgPermission(
		accountID,
		req.OrgId,
		model.PermissionResourceOrganization,
		model.PermissionActionManage,
	); err != nil {
		return nil, err
	}

	// 解析时间
	startTime, err := time.Parse("2006-01-02 15:04:05", req.StartTime)
	if err != nil {
		log.Error("创建活动失败: 解析开始时间异常: %v, org_id=%d account_id=%d start_time=%s", err, req.OrgId, accountID, req.StartTime)
		return nil, errors.New("开始时间格式错误")
	}
	endTime, err := time.Parse("2006-01-02 15:04:05", req.EndTime)
	if err != nil {
		log.Error("创建活动失败: 解析结束时间异常: %v, org_id=%d account_id=%d end_time=%s", err, req.OrgId, accountID, req.EndTime)
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
		log.Error("创建活动失败: 写入活动异常: %v, org_id=%d account_id=%d", err, req.OrgId, accountID)
		return nil, err
	}

	log.Info("创建活动成功: activity_id=%d org_id=%d account_id=%d", activity.ID, req.OrgId, accountID)
	return &api.CreateActivityResponse{
		Id:      activity.ID,
		Message: "创建活动成功",
	}, nil
}

// UpdateActivity 更新活动
func (s *ActivityService) UpdateActivity(req *api.UpdateActivityRequest) (*api.UpdateActivityResponse, error) {
	// 获取当前账号ID
	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("更新活动失败: 获取当前账户ID异常: %v, activity_id=%d", err, req.ActivityId)
		return nil, err
	}

	// 查询活动信息
	activity, err := s.repo.GetActivityByID(s.repo.DB, req.ActivityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		log.Error("更新活动失败: 查询活动异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
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

	if canErr := canUpdateActivity(activity); canErr != nil {
		return nil, canErr
	}

	// 解析时间
	if req.StartTime != "" {
		startTime, err := time.Parse("2006-01-02 15:04:05", req.StartTime)
		if err != nil {
			log.Error("更新活动失败: 解析开始时间异常: %v, activity_id=%d account_id=%d start_time=%s", err, req.ActivityId, accountID, req.StartTime)
			return nil, errors.New("开始时间格式错误")
		}
		activity.StartTime = startTime
	}
	if req.EndTime != "" {
		endTime, err := time.Parse("2006-01-02 15:04:05", req.EndTime)
		if err != nil {
			log.Error("更新活动失败: 解析结束时间异常: %v, activity_id=%d account_id=%d end_time=%s", err, req.ActivityId, accountID, req.EndTime)
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
	if isRequestFieldProvided(s.c, "maxPeople") {
		if req.MaxPeople < 0 {
			return nil, errors.New("最大招募人数不能为负数")
		}
		// 检查是否会导致报名人数超过新设定的最大人数
		if req.MaxPeople > 0 && activity.CurrentPeople > req.MaxPeople {
			return nil, errors.New("当前报名人数超过新设定的最大人数")
		}
		activity.MaxPeople = req.MaxPeople
	}

	if err := s.repo.UpdateActivity(s.repo.DB, activity); err != nil {
		log.Error("更新活动失败: 更新活动异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
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
		ActorID:     accountID,
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

	log.Info("更新活动成功: activity_id=%d org_id=%d account_id=%d", activity.ID, activity.OrgID, accountID)
	return &api.UpdateActivityResponse{
		Message: "更新活动成功",
	}, nil
}

// DeleteActivity 删除活动
func (s *ActivityService) DeleteActivity(req *api.DeleteActivityRequest) (*api.DeleteActivityResponse, error) {
	// 获取当前账号ID
	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("删除活动失败: 获取当前账户ID异常: %v, activity_id=%d", err, req.ActivityId)
		return nil, err
	}

	// 查询活动信息
	activity, err := s.repo.GetActivityByID(s.repo.DB, req.ActivityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		log.Error("删除活动失败: 查询活动异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
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

	// 校验活动状态
	if activity.Status == model.ActivityStatusFinished {
		return nil, errors.New("已结束的活动不能删除")
	}

	// 检查是否有已报名的志愿者
	if activity.CurrentPeople > 0 {
		// 可以选择允许删除或拒绝
		// 这里选择允许删除，记录日志
	}

	if err := s.repo.DeleteActivity(s.repo.DB, req.ActivityId); err != nil {
		log.Error("删除活动失败: 删除活动异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
		return nil, err
	}

	log.Info("删除活动成功: activity_id=%d org_id=%d account_id=%d", req.ActivityId, activity.OrgID, accountID)
	return &api.DeleteActivityResponse{
		Message: "删除活动成功",
	}, nil
}

// CancelActivity 取消活动
func (s *ActivityService) CancelActivity(req *api.CancelActivityRequest) (*api.CancelActivityResponse, error) {
	// 获取当前账号ID
	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("取消活动失败: 获取当前账户ID异常: %v, activity_id=%d", err, req.ActivityId)
		return nil, err
	}

	// 查询活动信息
	activity, err := s.repo.GetActivityByID(s.repo.DB, req.ActivityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		log.Error("取消活动失败: 查询活动异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
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

	if canErr := canCancelActivity(activity); canErr != nil {
		return nil, canErr
	}

	if err := s.repo.CancelActivity(s.repo.DB, req.ActivityId); err != nil {
		log.Error("取消活动失败: 更新活动状态异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
		return nil, err
	}

	PublishNotificationEvent(NotificationEvent{
		EventType:   model.NotificationEventActivityCanceled,
		BizType:     model.NotificationBizTypeActivity,
		BizID:       activity.ID,
		SourceOrgID: activity.OrgID,
		ActorID:     accountID,
		CreatedAt:   time.Now(),
		Payload: map[string]any{
			"activityTitle": activity.Title,
		},
		DedupeKey: fmt.Sprintf("activity.canceled:%d", activity.ID),
	})

	log.Info("取消活动成功: activity_id=%d org_id=%d account_id=%d", req.ActivityId, activity.OrgID, accountID)
	return &api.CancelActivityResponse{
		Message: "取消活动成功",
	}, nil
}

// FinishActivity 完结活动
func (s *ActivityService) FinishActivity(req *api.FinishActivityRequest) (*api.FinishActivityResponse, error) {
	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("完结活动失败: 获取当前账户ID异常: %v, activity_id=%d", err, req.ActivityId)
		return nil, err
	}

	activity, err := s.ensureActivityOperableByCurrentOrg(req.ActivityId, accountID)
	if err != nil {
		log.Error("完结活动失败: 校验活动归属异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
		return nil, err
	}
	if canErr := canFinishActivity(activity); canErr != nil {
		return nil, canErr
	}

	if err := s.repo.FinishActivity(s.repo.DB, req.ActivityId); err != nil {
		log.Error("完结活动失败: 更新活动状态异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
		return nil, err
	}

	log.Info("完结活动成功: activity_id=%d account_id=%d", req.ActivityId, accountID)
	return &api.FinishActivityResponse{Message: "完结活动成功"}, nil
}

// ========== 组织端：签到签退码管理 ==========

// GenerateAttendanceCodes 生成签到码/签退码（组织侧，初次生成同时生成两个码）
func (s *ActivityService) GenerateAttendanceCodes(req *api.GenerateAttendanceCodesRequest) (*api.GenerateAttendanceCodesResponse, error) {
	if req.ActivityId <= 0 {
		return nil, errors.New("活动ID不能为空")
	}
	if req.CheckInValidMinutes < 0 || req.CheckOutValidMinutes < 0 {
		return nil, errors.New("有效时长不能为负数")
	}

	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("生成签到签退码失败: 获取当前账户ID异常: %v, activity_id=%d", err, req.ActivityId)
		return nil, err
	}

	activity, err := s.ensureActivityOperableByCurrentOrg(req.ActivityId, accountID)
	if err != nil {
		log.Error("生成签到签退码失败: 校验活动归属异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
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
		log.Error("生成签到签退码失败: 生成签到码异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
		return nil, err
	}
	checkOutCode, checkOutCodeHash, checkOutExpireAt, err := generateAttendanceCodeForWrite(now, req.CheckOutValidMinutes)
	if err != nil {
		log.Error("生成签到签退码失败: 生成签退码异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
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
	if err := s.repo.UpdateActivityAttendanceCodeByID(s.repo.DB, req.ActivityId, updates); err != nil {
		log.Error("生成签到签退码失败: 更新活动码字段异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
		return nil, err
	}

	codeInfo, err := s.repo.GetActivityByID(s.repo.DB, req.ActivityId)
	if err != nil {
		log.Error("生成签到签退码失败: 查询活动码字段异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
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

	log.Info("生成签到签退码成功: activity_id=%d account_id=%d version=%d", req.ActivityId, accountID, resp.AttendanceCodeVersion)
	return resp, nil
}

// ResetAttendanceCode 重置签到码/签退码（组织侧，单次仅重置一种码）
func (s *ActivityService) ResetAttendanceCode(req *api.ResetAttendanceCodeRequest) (*api.ResetAttendanceCodeResponse, error) {
	if req.ActivityId <= 0 {
		return nil, errors.New("活动ID不能为空")
	}
	if !model.IsValidAttendanceCodeType(req.CodeType) {
		return nil, errors.New("重置码类型不合法")
	}
	if req.ValidMinutes < 0 {
		return nil, errors.New("有效时长不能为负数")
	}

	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("重置签到签退码失败: 获取当前账户ID异常: %v, activity_id=%d", err, req.ActivityId)
		return nil, err
	}

	activity, err := s.ensureActivityOperableByCurrentOrg(req.ActivityId, accountID)
	if err != nil {
		log.Error("重置签到签退码失败: 校验活动归属异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
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
		log.Error("重置签到签退码失败: 生成随机码异常: %v, activity_id=%d account_id=%d code_type=%d", err, req.ActivityId, accountID, req.CodeType)
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

	if err := s.repo.UpdateActivityAttendanceCodeByID(s.repo.DB, req.ActivityId, updates); err != nil {
		log.Error("重置签到签退码失败: 更新活动码字段异常: %v, activity_id=%d account_id=%d code_type=%d", err, req.ActivityId, accountID, req.CodeType)
		return nil, err
	}

	codeInfo, err := s.repo.GetActivityByID(s.repo.DB, req.ActivityId)
	if err != nil {
		log.Error("重置签到签退码失败: 查询活动码字段异常: %v, activity_id=%d account_id=%d code_type=%d", err, req.ActivityId, accountID, req.CodeType)
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

	log.Info("重置签到签退码成功: activity_id=%d account_id=%d code_type=%d version=%d", req.ActivityId, accountID, req.CodeType, resp.AttendanceCodeVersion)
	return resp, nil
}

// GetActivityAttendanceCodes 查询活动签到码/签退码（组织侧）
func (s *ActivityService) GetActivityAttendanceCodes(req *api.GetActivityAttendanceCodesRequest) (*api.GetActivityAttendanceCodesResponse, error) {
	if req.ActivityId <= 0 {
		return nil, errors.New("活动ID不能为空")
	}

	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("查询活动签到签退码失败: 获取当前账户ID异常: %v, activity_id=%d", err, req.ActivityId)
		return nil, err
	}

	activity, err := s.ensureActivityOperableByCurrentOrg(req.ActivityId, accountID)
	if err != nil {
		log.Error("查询活动签到签退码失败: 校验活动归属异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
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

// ========== 志愿者端：签到签退 ==========

// ActivityCheckIn 活动签到（志愿者侧）
func (s *ActivityService) ActivityCheckIn(req *api.ActivityCheckInRequest) (*api.ActivityCheckInResponse, error) {
	if req.ActivityId <= 0 {
		return nil, errors.New("活动ID不能为空")
	}
	if strings.TrimSpace(req.CheckInCode) == "" {
		return nil, errors.New("签到码不能为空")
	}

	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("活动签到失败: 获取当前账户ID异常: %v, activity_id=%d", err, req.ActivityId)
		return nil, err
	}
	volunteerID, err := s.getVolunteerIDByAccountID(accountID)
	if err != nil {
		log.Error("活动签到失败: 查询志愿者身份异常: %v, account_id=%d", err, accountID)
		return nil, err
	}

	var checkInTime time.Time
	err = s.withTransaction(func(tx *gorm.DB) error {
		activity, activityErr := s.repo.GetActivityByIDForUpdate(tx, req.ActivityId)
		if activityErr != nil {
			if errors.Is(activityErr, gorm.ErrRecordNotFound) {
				return errors.New("活动不存在")
			}
			return activityErr
		}
		if activity.Status == model.ActivityStatusCanceled {
			return errors.New("已取消活动不允许签到")
		}
		if codeErr := validateAttendanceCodeForActivity(activity, req.CheckInCode, model.AttendanceCodeTypeCheckIn); codeErr != nil {
			return codeErr
		}

		signup, err := s.repo.GetSignupForUpdate(tx, req.ActivityId, volunteerID)
		if err != nil {
			return err
		}
		if canErr := canCheckIn(activity, signup); canErr != nil {
			return canErr
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
		log.Error("活动签到失败: %v, activity_id=%d volunteer_id=%d account_id=%d", err, req.ActivityId, volunteerID, accountID)
		return nil, err
	}

	log.Info("活动签到成功: activity_id=%d volunteer_id=%d account_id=%d", req.ActivityId, volunteerID, accountID)
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

	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("活动签退失败: 获取当前账户ID异常: %v, activity_id=%d", err, req.ActivityId)
		return nil, err
	}
	volunteerID, err := s.getVolunteerIDByAccountID(accountID)
	if err != nil {
		log.Error("活动签退失败: 查询志愿者身份异常: %v, account_id=%d", err, accountID)
		return nil, err
	}

	activity, err := s.repo.GetActivityByID(s.repo.DB, req.ActivityId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("活动不存在")
		}
		log.Error("活动签退失败: 查询活动异常: %v, activity_id=%d account_id=%d", err, req.ActivityId, accountID)
		return nil, err
	}
	if activity.Status == model.ActivityStatusCanceled {
		return nil, errors.New("已取消活动不允许签退")
	}
	if err := validateAttendanceCodeForActivity(activity, req.CheckOutCode, model.AttendanceCodeTypeCheckOut); err != nil {
		log.Error("活动签退失败: 校验签退码异常: %v, activity_id=%d account_id=%d volunteer_id=%d", err, req.ActivityId, accountID, volunteerID)
		return nil, err
	}

	var checkOutTime time.Time
	var grantedHours float64
	// 事务外预检查：尽早失败，减少不必要的行锁持有时间。
	// Fast-fail before entering transaction to shorten lock hold time.
	preSignup, err := s.repo.GetSignup(s.repo.DB, req.ActivityId, volunteerID)
	if err != nil {
		log.Error("活动签退失败: 预检查报名记录异常: %v, activity_id=%d account_id=%d volunteer_id=%d", err, req.ActivityId, accountID, volunteerID)
		return nil, err
	}
	now := time.Now()
	if canErr := canCheckOut(activity, preSignup, now, volunteerCheckoutEarliestWindow); canErr != nil {
		return nil, canErr
	}

	// Re-check under row lock to prevent races between pre-check and final write.
	err = s.withTransaction(func(tx *gorm.DB) error {
		lockedActivity, activityErr := s.repo.GetActivityByIDForUpdate(tx, req.ActivityId)
		if activityErr != nil {
			if errors.Is(activityErr, gorm.ErrRecordNotFound) {
				return errors.New("活动不存在")
			}
			return activityErr
		}
		if lockedActivity.Status == model.ActivityStatusCanceled {
			return errors.New("已取消活动不允许签退")
		}
		if codeErr := validateAttendanceCodeForActivity(lockedActivity, req.CheckOutCode, model.AttendanceCodeTypeCheckOut); codeErr != nil {
			return codeErr
		}

		// 事务内二次校验：防止预检查与加锁之间的并发状态变化。
		signup, err := s.repo.GetSignupForUpdate(tx, req.ActivityId, volunteerID)
		if err != nil {
			return err
		}
		if canErr := canCheckOut(lockedActivity, signup, time.Now(), volunteerCheckoutEarliestWindow); canErr != nil {
			return canErr
		}

		// Idempotent return for repeated checkout requests.
		if signup.CheckOutStatus == model.ActivityCheckOutDone {
			if signup.CheckOutTime != nil {
				checkOutTime = *signup.CheckOutTime
			}
			grantedHours = signup.GrantedHours
			return nil
		}

		currentTime := time.Now()
		if currentTime.Before(*signup.CheckInTime) {
			currentTime = *signup.CheckInTime
		}
		checkOutTime = currentTime
		nextVersion := signup.WorkHourVersion + 1
		idempotencyKey := fmt.Sprintf("checkout:%d:%d", signup.ID, nextVersion)
		grantedHours, err = s.settleSignupWorkHours(
			tx,
			lockedActivity,
			signup,
			*signup.CheckInTime,
			checkOutTime,
			accountID,
			idempotencyKey,
			"签到签退自动结算",
			false,
		)
		return err
	})
	if err != nil {
		log.Error("活动签退失败: %v, activity_id=%d volunteer_id=%d account_id=%d", err, req.ActivityId, volunteerID, accountID)
		return nil, err
	}
	if grantedHours > 0 {
		s.publishWorkHourNotification(model.NotificationEventWorkHourGranted, preSignup.ID, accountID, grantedHours, "")
	}

	log.Info("活动签退成功: activity_id=%d volunteer_id=%d account_id=%d granted_hours=%.2f", req.ActivityId, volunteerID, accountID, grantedHours)
	return &api.ActivityCheckOutResponse{
		Success:      true,
		CheckOutTime: util.FormatDateTimeOrEmpty(checkOutTime),
		GrantedHours: grantedHours,
	}, nil
}

// ========== 组织端：签到签退补录 ==========

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

	accountID, err := s.currentAccountID()
	if err != nil {
		log.Error("活动补录失败: 获取当前账户ID异常: %v, activity_id=%d volunteer_id=%d", err, req.ActivityId, req.VolunteerId)
		return nil, err
	}

	activity, err := s.ensureActivityOperableByCurrentOrg(req.ActivityId, accountID)
	if err != nil {
		log.Error("活动补录失败: 校验活动归属异常: %v, activity_id=%d volunteer_id=%d account_id=%d", err, req.ActivityId, req.VolunteerId, accountID)
		return nil, err
	}
	if canErr := canSupplementAttendance(activity, nil); canErr != nil {
		return nil, canErr
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
		lockedActivity, activityErr := s.repo.GetActivityByIDForUpdate(tx, req.ActivityId)
		if activityErr != nil {
			if errors.Is(activityErr, gorm.ErrRecordNotFound) {
				return errors.New("活动不存在")
			}
			return activityErr
		}

		signup, err := s.repo.GetSignupForUpdate(tx, req.ActivityId, req.VolunteerId)
		if err != nil {
			return err
		}
		if canErr := canSupplementAttendance(lockedActivity, signup); canErr != nil {
			return canErr
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
			lockedActivity,
			signup,
			finalCheckIn,
			finalCheckOut,
			accountID,
			idempotencyKey,
			reason,
			true,
		)
		return err
	})
	if err != nil {
		log.Error("活动补录失败: %v, activity_id=%d volunteer_id=%d account_id=%d", err, req.ActivityId, req.VolunteerId, accountID)
		return nil, err
	}
	if grantedHours > 0 {
		signup, signupErr := s.repo.GetSignup(s.repo.DB, req.ActivityId, req.VolunteerId)
		if signupErr == nil && signup != nil {
			s.publishWorkHourNotification(model.NotificationEventWorkHourGranted, signup.ID, accountID, grantedHours, reason)
		}
	}

	log.Info("活动补录成功: activity_id=%d volunteer_id=%d account_id=%d granted_hours=%.2f", req.ActivityId, req.VolunteerId, accountID, grantedHours)
	return &api.ActivitySupplementAttendanceResponse{
		Success:      true,
		CheckInTime:  util.FormatDateTimeOrEmpty(finalCheckIn),
		CheckOutTime: util.FormatDateTimeOrEmpty(finalCheckOut),
		GrantedHours: grantedHours,
	}, nil
}
