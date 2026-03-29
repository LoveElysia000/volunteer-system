package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

type AuditService struct {
	Service
}

// NewAuditService 创建审核服务并初始化上下文与仓储依赖。
func NewAuditService(ctx context.Context, c *app.RequestContext) *AuditService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &AuditService{
		Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
}

const (
	defaultAuditSLAHours int32 = 24
)

// ===== 审核端：查询能力 =====

// PendingAuditList 查询统一待审核列表（按目标类型筛选）。
func (s *AuditService) PendingAuditList(req *api.PendingAuditListRequest) (*api.PendingAuditListResponse, error) {
	if req == nil {
		log.Warn("统一待审核列表查询失败: 请求为空")
		return nil, errors.New("请求不能为空")
	}

	operatorID, err := s.getAuditOperatorID()
	if err != nil {
		log.Warn("统一待审核列表查询失败: 权限校验失败: %v", err)
		return nil, err
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	targetTypes, err := normalizeAuditTargetTypes(req)
	if err != nil {
		log.Warn("统一待审核列表查询失败: target_types参数无效: %v", err)
		return nil, err
	}
	statuses, err := normalizeAuditStatuses(req.Status)
	if err != nil {
		log.Warn("统一待审核列表查询失败: status参数无效: %v", err)
		return nil, err
	}
	slaHours := req.SlaHours
	if slaHours <= 0 {
		slaHours = defaultAuditSLAHours
	}
	queryMap := map[string]any{
		"target_type IN ?": targetTypes,
		"status in (?)":    statuses,
	}
	if strings.TrimSpace(req.CreatedFrom) != "" {
		from, parseErr := util.ParseDateFilterBound(strings.TrimSpace(req.CreatedFrom), false)
		if parseErr != nil {
			return nil, errors.New("createdFrom 时间格式错误")
		}
		queryMap["created_at >= ?"] = from
	}
	if strings.TrimSpace(req.CreatedTo) != "" {
		to, parseErr := util.ParseDateFilterBound(strings.TrimSpace(req.CreatedTo), true)
		if parseErr != nil {
			return nil, errors.New("createdTo 时间格式错误")
		}
		queryMap["created_at <= ?"] = to
	}

	resp := &api.PendingAuditListResponse{
		List: make([]*api.PendingAuditItem, 0),
	}

	records, _, listErr := s.repo.GetAuditRecordsList(s.repo.DB, queryMap, 0, 0)
	if listErr != nil {
		log.Error("统一待审核列表查询失败: 查询审核记录异常: %v, page=%d page_size=%d", listErr, page, pageSize)
		return nil, listErr
	}

	if len(records) == 0 {
		return resp, nil
	}

	authorizedRecords := make([]*model.AuditRecord, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		if err := s.requireAuditReviewPermission(operatorID, record); err != nil {
			continue
		}
		authorizedRecords = append(authorizedRecords, record)
	}
	if len(authorizedRecords) == 0 {
		return resp, nil
	}

	filteredItems := filterPendingAuditItemsByKeyword(
		s.buildPendingAuditItems(authorizedRecords, slaHours),
		req.Keyword,
	)
	resp.Total = int32(len(filteredItems))
	start := int((page - 1) * pageSize)
	if start >= len(filteredItems) {
		resp.List = []*api.PendingAuditItem{}
		return resp, nil
	}
	end := start + int(pageSize)
	if end > len(filteredItems) {
		end = len(filteredItems)
	}
	resp.List = filteredItems[start:end]
	return resp, nil
}

// ----- 审核端私有能力：列表参数与展示组装 -----

// normalizeAuditTargetTypes 校验并返回审核目标类型集合。
func normalizeAuditTargetTypes(req *api.PendingAuditListRequest) ([]int32, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}

	inputs := make([]int32, 0, len(req.TargetTypes))
	inputs = append(inputs, req.TargetTypes...)
	if len(inputs) == 0 {
		return []int32{
			model.AuditTargetMember,
			model.AuditTargetSignup,
			model.AuditTargetVolunteer,
			model.AuditTargetOrg,
		}, nil
	}

	result := make([]int32, 0, len(inputs))
	seen := make(map[int32]struct{}, len(inputs))
	for _, input := range inputs {
		if !model.IsValidAuditTargetType(input) {
			return nil, errors.New("审核目标类型不合法")
		}
		if _, ok := seen[input]; ok {
			continue
		}
		seen[input] = struct{}{}
		result = append(result, input)
	}
	return result, nil
}

// normalizeAuditStatuses 归一化审核状态筛选条件（空值默认待审核）。
func normalizeAuditStatuses(input []int32) ([]int32, error) {
	if len(input) == 0 {
		return []int32{model.AuditStatusPending}, nil
	}

	result := make([]int32, 0, len(input))
	seen := make(map[int32]struct{}, len(input))
	for _, status := range input {
		if status != model.AuditStatusPending &&
			status != model.AuditStatusApproved &&
			status != model.AuditStatusRejected {
			return nil, errors.New("审核状态不合法")
		}
		if _, ok := seen[status]; ok {
			continue
		}
		seen[status] = struct{}{}
		result = append(result, status)
	}
	return result, nil
}

func filterPendingAuditItemsByKeyword(items []*api.PendingAuditItem, keyword string) []*api.PendingAuditItem {
	trimmedKeyword := strings.TrimSpace(keyword)
	if trimmedKeyword == "" {
		return items
	}

	filtered := make([]*api.PendingAuditItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		if strings.Contains(item.Title, trimmedKeyword) || strings.Contains(item.SubTitle, trimmedKeyword) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// buildPendingAuditItems 将审核记录列表转换为统一待审核返回项。
func (s *AuditService) buildPendingAuditItems(records []*model.AuditRecord, slaHours int32) []*api.PendingAuditItem {
	items := make([]*api.PendingAuditItem, 0, len(records))
	threshold := time.Duration(slaHours) * time.Hour
	now := time.Now()
	for _, record := range records {
		if record == nil {
			continue
		}
		title, subTitle := s.resolvePendingAuditTitle(record)
		isOverdue := false
		if threshold > 0 && record.Status == model.AuditStatusPending {
			isOverdue = now.Sub(record.CreatedAt) > threshold
		}
		items = append(items, &api.PendingAuditItem{
			Id:         record.ID,
			TargetType: record.TargetType,
			TargetId:   record.TargetID,
			Title:      title,
			SubTitle:   subTitle,
			CreatorId:  record.CreatorID,
			CreatedAt:  record.CreatedAt.Format(util.DateTimeLayout),
			IsOverdue:  isOverdue,
		})
	}
	return items
}

// resolvePendingAuditTitle 按审核目标类型分发标题解析逻辑。
func (s *AuditService) resolvePendingAuditTitle(record *model.AuditRecord) (string, string) {
	switch record.TargetType {
	case model.AuditTargetMember:
		return s.resolveMemberAuditTitle(record)
	case model.AuditTargetSignup:
		return s.resolveSignupAuditTitle(record)
	case model.AuditTargetVolunteer:
		return s.resolveVolunteerAuditTitle(record)
	case model.AuditTargetOrg:
		return s.resolveOrganizationAuditTitle(record)
	default:
		return "审核记录", "未知审核类型"
	}
}

// resolveVolunteerAuditTitle 构建志愿者审核项的标题与副标题。
func (s *AuditService) resolveVolunteerAuditTitle(record *model.AuditRecord) (string, string) {
	title := ""
	if record.TargetID > 0 {
		volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, record.TargetID)
		if err == nil && volunteer != nil {
			title = volunteer.RealName
		}
	}

	subTitle := "志愿者实名认证"
	scene, data, isEnvelope, err := parseAuditSnapshotEnvelope(record.NewContent)
	if err == nil && isEnvelope {
		if title == "" && scene == model.AuditSceneVolunteerRealNameVerify {
			var payload VolunteerRealNameVerifyAuditPayload
			if unmarshalErr := json.Unmarshal(data, &payload); unmarshalErr == nil {
				title = strings.TrimSpace(payload.RealName)
			}
		}
	}

	if title == "" {
		title = "志愿者审核"
	}
	return title, subTitle
}

// resolveOrganizationAuditTitle 构建组织审核项的标题与副标题。
func (s *AuditService) resolveOrganizationAuditTitle(record *model.AuditRecord) (string, string) {
	title := ""
	if record.TargetID > 0 {
		organization, err := s.repo.GetOrganizationByID(s.repo.DB, record.TargetID)
		if err == nil && organization != nil {
			title = organization.OrgName
		}
	}
	if title == "" {
		title = "组织审核"
	}
	return title, "组织资质审核"
}

// resolveMemberAuditTitle 构建加入组织审核项的标题与副标题。
func (s *AuditService) resolveMemberAuditTitle(record *model.AuditRecord) (string, string) {
	var member model.OrgMember
	memberID := record.TargetID
	if strings.TrimSpace(record.NewContent) != "" {
		_ = json.Unmarshal([]byte(record.NewContent), &member)
	}
	if memberID > 0 && (member.VolunteerID <= 0 || member.OrgID <= 0) {
		memberEntity, err := s.repo.GetMembershipByID(s.repo.DB, memberID)
		if err == nil && memberEntity != nil {
			member = *memberEntity
		}
	}

	title := "加入组织审核"
	if member.VolunteerID > 0 {
		volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, member.VolunteerID)
		if err == nil && volunteer != nil && strings.TrimSpace(volunteer.RealName) != "" {
			title = volunteer.RealName
		}
	}

	subTitle := "组织信息"
	if member.OrgID > 0 {
		organization, err := s.repo.GetOrganizationByID(s.repo.DB, member.OrgID)
		if err == nil && organization != nil && strings.TrimSpace(organization.OrgName) != "" {
			subTitle = organization.OrgName
		}
	}
	return title, subTitle
}

// resolveSignupAuditTitle 构建活动报名审核项的标题与副标题。
func (s *AuditService) resolveSignupAuditTitle(record *model.AuditRecord) (string, string) {
	var signup model.ActivitySignup
	if strings.TrimSpace(record.NewContent) != "" {
		_ = json.Unmarshal([]byte(record.NewContent), &signup)
	}
	if record.TargetID > 0 && (signup.ActivityID <= 0 || signup.VolunteerID <= 0) {
		signupEntity, err := s.repo.GetActivitySignupByID(s.repo.DB, record.TargetID)
		if err == nil && signupEntity != nil {
			signup = *signupEntity
		}
	}

	title := "活动报名审核"
	if signup.VolunteerID > 0 {
		volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, signup.VolunteerID)
		if err == nil && volunteer != nil && strings.TrimSpace(volunteer.RealName) != "" {
			title = volunteer.RealName
		}
	}

	subTitle := "活动信息"
	if signup.ActivityID > 0 {
		activity, err := s.repo.GetActivityByID(s.repo.DB, signup.ActivityID)
		if err == nil && activity != nil && strings.TrimSpace(activity.Title) != "" {
			subTitle = activity.Title
		}
	}
	return title, subTitle
}

// ===== 审核端：审核结果落库与副作用 =====

// applyVolunteerAuditApproval 处理志愿者目标的审核通过并回写志愿者信息。
func (s *AuditService) applyVolunteerAuditApproval(tx *gorm.DB, record *model.AuditRecord) error {
	volunteer, err := s.repo.FindVolunteerByID(tx, record.TargetID)
	if err != nil {
		return err
	}

	if record.OperationType != model.OperationTypeUpdate {
		return errors.New("志愿者审核仅支持更新操作")
	}

	scene, err := resolveVolunteerUpdateAuditScene(record.NewContent)
	if err != nil {
		return err
	}

	switch scene {
	case model.AuditSceneVolunteerRealNameVerify:
		newPayload, parseErr := unmarshalVolunteerRealNameVerifyAuditPayload(record.NewContent)
		if parseErr != nil {
			return parseErr
		}
		updates, err := newPayload.BuildVolunteerApprovalUpdates()
		if err != nil {
			return err
		}
		updates["audit_status"] = model.VolunteerAuditStatusApproved
		return s.repo.UpdateVolunteer(tx, volunteer.ID, updates)
	default:
		return errors.New("不支持的志愿者审核场景")
	}
}

// applyAuditRejectionSideEffects 处理审核驳回后的附加副作用。
func (s *AuditService) applyAuditRejectionSideEffects(tx *gorm.DB, record *model.AuditRecord) error {
	if record.TargetType == model.AuditTargetVolunteer && record.OperationType == model.OperationTypeUpdate {
		scene, err := resolveVolunteerUpdateAuditScene(record.NewContent)
		if err != nil {
			return err
		}
		if scene == model.AuditSceneVolunteerRealNameVerify {
			volunteer, err := s.repo.FindVolunteerByID(tx, record.TargetID)
			if err != nil {
				return err
			}
			if err := s.repo.UpdateVolunteer(tx, volunteer.ID, map[string]any{
				"audit_status": model.VolunteerAuditStatusRejected,
			}); err != nil {
				return err
			}
		}
	}
	return s.applySignupAuditRejection(tx, record)
}

// applySignupAuditRejection materializes or updates signup status for rejected signup audits.
func (s *AuditService) applySignupAuditRejection(tx *gorm.DB, record *model.AuditRecord) error {
	if record.TargetType != model.AuditTargetSignup {
		return nil
	}

	if record.OperationType == model.OperationTypeCreate && record.TargetID <= 0 {
		if strings.TrimSpace(record.NewContent) == "" {
			return errors.New("报名快照无效")
		}
		var signupSnapshot model.ActivitySignup
		if err := json.Unmarshal([]byte(record.NewContent), &signupSnapshot); err != nil {
			return err
		}
		if signupSnapshot.ActivityID <= 0 || signupSnapshot.VolunteerID <= 0 {
			return errors.New("报名快照无效")
		}

		signup, err := s.repo.GetSignup(tx, signupSnapshot.ActivityID, signupSnapshot.VolunteerID)
		if err != nil {
			return err
		}
		decision, err := resolveSignupTransition(signupTransitionReject, signup)
		if err != nil {
			return err
		}
		if decision.apply {
			if decision.createIfMissing {
				signup = &model.ActivitySignup{
					ActivityID:  signupSnapshot.ActivityID,
					VolunteerID: signupSnapshot.VolunteerID,
					Status:      decision.toStatus,
				}
				if err := s.repo.CreateSignup(tx, signup); err != nil {
					return err
				}
			} else {
				if signup == nil {
					return errors.New("报名记录不存在")
				}
				if err := s.repo.UpdateActivitySignupStatusByID(tx, signup.ID, decision.toStatus); err != nil {
					return err
				}
			}
		}

		if signup != nil {
			record.TargetID = signup.ID
		}
		return nil
	}

	if record.TargetID <= 0 {
		return nil
	}

	signup, err := s.repo.GetActivitySignupByID(tx, record.TargetID)
	if err != nil {
		return err
	}
	decision, err := resolveSignupTransition(signupTransitionReject, signup)
	if err != nil {
		return err
	}
	if !decision.apply {
		return nil
	}
	return s.repo.UpdateActivitySignupStatusByID(tx, signup.ID, decision.toStatus)
}

// applySignupAuditApproval 处理活动报名目标的审核通过并同步报名状态。
func (s *AuditService) applySignupAuditApproval(tx *gorm.DB, record *model.AuditRecord) error {
	if record.OperationType == model.OperationTypeCreate && record.TargetID <= 0 {
		if strings.TrimSpace(record.NewContent) == "" {
			return errors.New("报名快照无效")
		}

		var signupSnapshot model.ActivitySignup
		if err := json.Unmarshal([]byte(record.NewContent), &signupSnapshot); err != nil {
			return err
		}
		if signupSnapshot.ActivityID <= 0 || signupSnapshot.VolunteerID <= 0 {
			return errors.New("报名快照无效")
		}
		activity, err := s.repo.GetActivityByIDForUpdate(tx, signupSnapshot.ActivityID)
		if err != nil {
			return err
		}
		if activity.Status != model.ActivityStatusRecruiting {
			return errors.New("活动已结束或已取消")
		}

		signup, err := s.repo.GetSignup(tx, signupSnapshot.ActivityID, signupSnapshot.VolunteerID)
		if err != nil {
			return err
		}
		decision, err := resolveSignupTransition(signupTransitionApprove, signup)
		if err != nil {
			return err
		}
		if decision.peopleDelta > 0 && activity.MaxPeople > 0 && activity.CurrentPeople >= activity.MaxPeople {
			return errors.New("活动名额已满")
		}
		if decision.apply {
			if decision.createIfMissing {
				signup = &model.ActivitySignup{
					ActivityID:  signupSnapshot.ActivityID,
					VolunteerID: signupSnapshot.VolunteerID,
					Status:      decision.toStatus,
				}
				if err := s.repo.CreateSignup(tx, signup); err != nil {
					return err
				}
			} else {
				if signup == nil {
					return errors.New("报名记录不存在")
				}
				if err := s.repo.UpdateActivitySignupStatusByID(tx, signup.ID, decision.toStatus); err != nil {
					return err
				}
			}
		}
		if decision.peopleDelta > 0 {
			if err := s.repo.IncrementActivityPeople(tx, signupSnapshot.ActivityID); err != nil {
				return err
			}
		}
		if signup == nil {
			return errors.New("报名记录不存在")
		}
		record.TargetID = signup.ID
		return nil
	}

	signup, err := s.repo.GetActivitySignupByID(tx, record.TargetID)
	if err != nil {
		return err
	}
	activity, err := s.repo.GetActivityByIDForUpdate(tx, signup.ActivityID)
	if err != nil {
		return err
	}
	if activity.Status != model.ActivityStatusRecruiting {
		return errors.New("活动已结束或已取消")
	}
	decision, err := resolveSignupTransition(signupTransitionApprove, signup)
	if err != nil {
		return err
	}
	if !decision.apply {
		return nil
	}
	if decision.peopleDelta > 0 && activity.MaxPeople > 0 && activity.CurrentPeople >= activity.MaxPeople {
		return errors.New("活动名额已满")
	}
	if err := s.repo.UpdateActivitySignupStatusByID(tx, signup.ID, decision.toStatus); err != nil {
		return err
	}
	if decision.peopleDelta > 0 {
		return s.repo.IncrementActivityPeople(tx, signup.ActivityID)
	}
	return nil
}

func (s *AuditService) handleAuditApprovedSideEffects(record *model.AuditRecord, auditorID int64) {
	if record == nil || record.TargetID <= 0 {
		return
	}
	if record.TargetType == model.AuditTargetSignup {
		signup, err := s.repo.GetActivitySignupByID(s.repo.DB, record.TargetID)
		if err != nil || signup == nil || signup.Status != model.ActivitySignupStatusSuccess {
			return
		}
		s.publishSignupApprovedNotification(signup.ID, auditorID)
		return
	}
	if record.TargetType != model.AuditTargetMember {
		return
	}

	member, err := s.repo.GetMembershipByID(s.repo.DB, record.TargetID)
	if err != nil || member == nil {
		log.Error("审核通过副作用失败: 查询成员关系异常: %v, record_id=%d target_id=%d", err, record.ID, record.TargetID)
		return
	}

	volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, member.VolunteerID)
	if err != nil || volunteer == nil {
		log.Error("审核通过副作用失败: 查询志愿者异常: %v, record_id=%d target_id=%d volunteer_id=%d", err, record.ID, record.TargetID, member.VolunteerID)
		return
	}
	if volunteer.AccountID <= 0 {
		log.Warn("审核通过副作用跳过: 志愿者账号ID无效, record_id=%d target_id=%d volunteer_id=%d", record.ID, record.TargetID, member.VolunteerID)
		return
	}

	if record.OperationType == model.OperationTypeDelete || member.Status == model.MemberStatusLeft {
		rows, archiveErr := s.repo.ArchiveNotificationInboxByReceiverAndOrg(
			s.repo.DB,
			volunteer.AccountID,
			member.OrgID,
			model.NotificationArchiveReasonLeftOrg,
			time.Now(),
		)
		if archiveErr != nil {
			log.Error("审核通过副作用失败: 归档通知异常: %v, record_id=%d target_id=%d receiver_id=%d org_id=%d",
				archiveErr, record.ID, record.TargetID, volunteer.AccountID, member.OrgID)
			return
		}
		log.Info("审核通过副作用完成: 归档组织通知成功, record_id=%d target_id=%d receiver_id=%d org_id=%d archived_rows=%d",
			record.ID, record.TargetID, volunteer.AccountID, member.OrgID, rows)
		return
	}

	// 仅“加入组织创建审核通过”发入组通知，避免更新类审核误触发。
	if record.OperationType != model.OperationTypeCreate {
		return
	}

	if member.Status != model.MemberStatusActive {
		return
	}

	orgName := ""
	organization, orgErr := s.repo.GetOrganizationByID(s.repo.DB, member.OrgID)
	if orgErr == nil && organization != nil {
		orgName = organization.OrgName
	}

	PublishNotificationEvent(NotificationEvent{
		EventType:   model.NotificationEventMemberJoinApproved,
		BizType:     model.NotificationBizTypeMembership,
		BizID:       member.ID,
		SourceOrgID: member.OrgID,
		ActorID:     auditorID,
		CreatedAt:   time.Now(),
		Payload: map[string]any{
			"receiverAccountID": volunteer.AccountID,
			"organizationName":  orgName,
			"volunteerName":     volunteer.RealName,
		},
		DedupeKey: fmt.Sprintf("member.join.approved:%d", member.ID),
	})
}

func (s *AuditService) handleAuditRejectedSideEffects(record *model.AuditRecord, auditorID int64, reason string) {
	if record == nil {
		return
	}
	if record.TargetType != model.AuditTargetSignup || record.TargetID <= 0 {
		return
	}

	signup, err := s.repo.GetActivitySignupByID(s.repo.DB, record.TargetID)
	if err != nil || signup == nil {
		return
	}
	if signup.Status != model.ActivitySignupStatusRejected {
		return
	}
	activity, err := s.repo.GetActivityByID(s.repo.DB, signup.ActivityID)
	if err != nil || activity == nil {
		return
	}
	volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, signup.VolunteerID)
	if err != nil || volunteer == nil || volunteer.AccountID <= 0 {
		return
	}

	PublishNotificationEvent(NotificationEvent{
		EventType:   model.NotificationEventSignupRejected,
		BizType:     model.NotificationBizTypeActivity,
		BizID:       signup.ID,
		SourceOrgID: activity.OrgID,
		ActorID:     auditorID,
		CreatedAt:   time.Now(),
		Payload: map[string]any{
			"receiverAccountID": volunteer.AccountID,
			"activityTitle":     activity.Title,
			"reason":            reason,
		},
		DedupeKey: fmt.Sprintf("signup.rejected:%d:%d", signup.ID, record.ID),
	})
}

// ===== 审核端：详情查询 =====

// AuditRecordDetail 查询并返回单条审核记录详情。
func (s *AuditService) AuditRecordDetail(req *api.AuditRecordDetailRequest) (*api.AuditRecordDetailResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if req.Id <= 0 {
		return nil, errors.New("审核记录ID不能为空")
	}

	operatorID, err := s.getAuditOperatorID()
	if err != nil {
		return nil, err
	}
	record, err := s.repo.GetAuditRecordByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("审核记录不存在")
		}
		return nil, err
	}
	if err := s.requireAuditReviewPermission(operatorID, record); err != nil {
		return nil, err
	}

	auditTime := ""
	if !record.AuditTime.IsZero() {
		auditTime = record.AuditTime.Format(util.DateTimeLayout)
	}
	createdAt := ""
	if !record.CreatedAt.IsZero() {
		createdAt = record.CreatedAt.Format(util.DateTimeLayout)
	}

	return &api.AuditRecordDetailResponse{
		Record: &api.AuditRecordDetail{
			Id:           record.ID,
			TargetType:   record.TargetType,
			TargetId:     record.TargetID,
			AuditorId:    record.AuditorID,
			Status:       record.Status,
			OldContent:   record.OldContent,
			NewContent:   record.NewContent,
			AuditResult:  record.AuditResult,
			RejectReason: record.RejectReason,
			AuditTime:    auditTime,
			CreatedAt:    createdAt,
		},
	}, nil
}

// ----- 审核端私有能力：鉴权与作用域解析 -----

// getAuditOperatorID 获取审核操作人 ID 并校验其审核权限。
func (s *AuditService) getAuditOperatorID() (int64, error) {
	auditorAccountID, err := s.currentAccountID()
	if err != nil {
		log.Warn("获取审核人失败: 无法从上下文获取账户ID, err=%v", err)
		return 0, err
	}
	if auditorAccountID <= 0 {
		log.Warn("获取审核人失败: 账户ID无效, account_id=%d", auditorAccountID)
		return 0, errors.New("审核人无效")
	}

	account, err := s.repo.FindByID(s.repo.DB, auditorAccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("获取审核人失败: 账号不存在, account_id=%d", auditorAccountID)
			return 0, errors.New("账号不存在")
		}
		log.Error("获取审核人失败: 查询账号异常, account_id=%d err=%v", auditorAccountID, err)
		return 0, err
	}
	if account.Status != model.SysAccountNormal {
		return 0, errors.New("账号状态异常")
	}

	hasGlobal, err := s.hasPermissionByScope(
		auditorAccountID,
		model.RBACScopeGlobal,
		0,
		model.PermissionResourceAudit,
		model.PermissionActionReview,
	)
	if err != nil {
		return 0, err
	}
	if hasGlobal {
		return auditorAccountID, nil
	}

	hasAnyOrg, err := s.repo.HasAnyOrgPermission(
		s.repo.DB,
		auditorAccountID,
		model.PermissionResourceAudit,
		model.PermissionActionReview,
	)
	if err != nil {
		return 0, err
	}
	if !hasAnyOrg {
		return 0, errors.New("无权限执行审核")
	}

	return auditorAccountID, nil
}

func (s *AuditService) requireAuditReviewPermission(operatorID int64, record *model.AuditRecord) error {
	orgID, err := s.resolveAuditRecordOrgID(record)
	if err != nil {
		return err
	}
	if orgID > 0 {
		return s.requireOrgPermission(
			operatorID,
			orgID,
			model.PermissionResourceAudit,
			model.PermissionActionReview,
		)
	}
	return s.requireGlobalPermission(
		operatorID,
		model.PermissionResourceAudit,
		model.PermissionActionReview,
	)
}

func (s *AuditService) resolveAuditRecordOrgID(record *model.AuditRecord) (int64, error) {
	if record == nil {
		return 0, errors.New("审核记录不能为空")
	}

	switch record.TargetType {
	case model.AuditTargetOrg:
		return record.TargetID, nil

	case model.AuditTargetMember:
		if record.TargetID > 0 {
			member, err := s.repo.GetMembershipByID(s.repo.DB, record.TargetID)
			if err == nil && member != nil {
				return member.OrgID, nil
			}
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, err
			}
		}
		var snapshot model.OrgMember
		if strings.TrimSpace(record.NewContent) != "" && json.Unmarshal([]byte(record.NewContent), &snapshot) == nil {
			return snapshot.OrgID, nil
		}
		return 0, nil

	case model.AuditTargetSignup:
		activityID := int64(0)
		if record.TargetID > 0 {
			signup, err := s.repo.GetActivitySignupByID(s.repo.DB, record.TargetID)
			if err == nil && signup != nil {
				activityID = signup.ActivityID
			}
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, err
			}
		}
		if activityID <= 0 {
			var snapshot model.ActivitySignup
			if strings.TrimSpace(record.NewContent) != "" && json.Unmarshal([]byte(record.NewContent), &snapshot) == nil {
				activityID = snapshot.ActivityID
			}
		}
		if activityID <= 0 {
			return 0, nil
		}
		activity, err := s.repo.GetActivityByID(s.repo.DB, activityID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return 0, nil
			}
			return 0, err
		}
		return activity.OrgID, nil

	default:
		return 0, nil
	}
}
