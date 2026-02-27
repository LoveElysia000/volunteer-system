package service

import (
	"context"
	"encoding/json"
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

type ApprovalHandler func(*gorm.DB, *model.AuditRecord) error

// PendingAuditList 查询统一待审核列表（按目标类型筛选）。
func (s *AuditService) PendingAuditList(req *api.PendingAuditListRequest) (*api.PendingAuditListResponse, error) {
	if req == nil {
		log.Warn("统一待审核列表查询失败: 请求为空")
		return nil, errors.New("请求不能为空")
	}

	// 仅组织管理员可查看统一待审核列表。
	if _, err := s.getAuditOperatorID(); err != nil {
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

	targetType, err := normalizeAuditTargetType(req.TargetType)
	if err != nil {
		log.Warn("统一待审核列表查询失败: target_type参数无效: %v", err)
		return nil, err
	}
	statuses, err := normalizeAuditStatuses(req.Status)
	if err != nil {
		log.Warn("统一待审核列表查询失败: status参数无效: %v", err)
		return nil, err
	}

	queryMap := map[string]any{
		"target_type = ?": targetType,
		"status in (?)":   statuses,
	}

	keyword := strings.TrimSpace(req.Keyword)
	resp := &api.PendingAuditListResponse{
		List: make([]*api.PendingAuditItem, 0),
	}

	if keyword != "" {
		matchedIDs, matchErr := s.searchPendingAuditRecordIDsByKeyword(targetType, statuses, keyword)
		if matchErr != nil {
			log.Error("统一待审核列表查询失败: 关键词匹配异常: %v, target_type=%d keyword=%q", matchErr, targetType, keyword)
			return nil, matchErr
		}
		if len(matchedIDs) == 0 {
			return resp, nil
		}
		queryMap["id in (?)"] = matchedIDs
	}

	offset := pageSize * (page - 1)
	records, total, listErr := s.repo.GetAuditRecordsList(s.repo.DB, queryMap, pageSize, offset)
	if listErr != nil {
		log.Error("统一待审核列表查询失败: 查询审核记录异常: %v, page=%d page_size=%d", listErr, page, pageSize)
		return nil, listErr
	}

	if total == 0 {
		return resp, nil
	}
	resp.Total = int32(total)
	resp.List = s.buildPendingAuditItems(records)
	return resp, nil
}

// normalizeAuditTargetType 校验并返回审核目标类型。
func normalizeAuditTargetType(input int32) (int32, error) {
	if !model.IsValidAuditTargetType(input) {
		return 0, errors.New("审核目标类型不合法")
	}
	return input, nil
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

// buildPendingAuditItems 将审核记录列表转换为统一待审核返回项。
func (s *AuditService) buildPendingAuditItems(records []*model.AuditRecord) []*api.PendingAuditItem {
	items := make([]*api.PendingAuditItem, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		title, subTitle := s.resolvePendingAuditTitle(record)
		items = append(items, &api.PendingAuditItem{
			Id:         record.ID,
			TargetType: record.TargetType,
			TargetId:   record.TargetID,
			Title:      title,
			SubTitle:   subTitle,
			CreatorId:  record.CreatorID,
			CreatedAt:  record.CreatedAt.Format(util.DateTimeLayout),
		})
	}
	return items
}

// searchPendingAuditRecordIDsByKeyword 通过标题/副标题匹配获取审核记录主键ID集合。
func (s *AuditService) searchPendingAuditRecordIDsByKeyword(targetType int32, statuses []int32, keyword string) ([]int64, error) {
	queryMap := map[string]any{
		"target_type = ?": targetType,
		"status in (?)":   statuses,
	}
	records, _, err := s.repo.GetAuditRecordsList(s.repo.DB, queryMap, 0, 0)
	if err != nil {
		return nil, err
	}

	matchedIDs := make([]int64, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		title, subTitle := s.resolvePendingAuditTitle(record)
		if matchAuditKeyword(title, subTitle, keyword) {
			matchedIDs = append(matchedIDs, record.ID)
		}
	}
	return matchedIDs, nil
}

// matchAuditKeyword 判断标题与副标题是否命中关键词。
func matchAuditKeyword(title, subTitle, keyword string) bool {
	trimmed := strings.TrimSpace(keyword)
	if trimmed == "" {
		return true
	}
	keywordLower := strings.ToLower(trimmed)
	return strings.Contains(strings.ToLower(title), keywordLower) ||
		strings.Contains(strings.ToLower(subTitle), keywordLower)
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

	subTitle := "志愿者资料审核"
	scene, data, isEnvelope, err := parseAuditSnapshotEnvelope(record.NewContent)
	if err == nil && isEnvelope {
		switch scene {
		case model.AuditSceneVolunteerRealNameVerify:
			subTitle = "志愿者实名认证"
		case model.AuditSceneVolunteerProfileUpdate:
			subTitle = "志愿者资料变更"
		}

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

// AuditApproval 审核通过指定记录并执行对应审核目标的通过逻辑。
func (s *AuditService) AuditApproval(req *api.AuditApprovalRequest) (*api.AuditApprovalResponse, error) {
	var resp api.AuditApprovalResponse
	if req == nil {
		log.Warn("审核通过失败: 请求为空")
		return nil, errors.New("请求不能为空")
	}
	if req.Id <= 0 {
		log.Warn("审核通过失败: 审核记录ID为空")
		return nil, errors.New("审核记录ID不能为空")
	}
	record, err := s.repo.GetAuditRecordByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("审核通过失败: 审核记录不存在, record_id=%d", req.Id)
			return nil, errors.New("审核记录不存在")
		}
		log.Error("审核通过失败: 查询审核记录异常: %v, record_id=%d", err, req.Id)
		return nil, err
	}

	if err := ensureAuditRecordPending(record); err != nil {
		log.Warn("审核通过失败: 审核记录不可处理, record_id=%d status=%d audit_result=%d", record.ID, record.Status, record.AuditResult)
		return nil, err
	}

	auditorID, err := s.getAuditOperatorID()
	if err != nil {
		log.Warn("审核通过失败: 获取审核人失败, record_id=%d err=%v", record.ID, err)
		return nil, err
	}

	auditHandlerMap := map[int32]ApprovalHandler{
		model.AuditTargetVolunteer: s.applyVolunteerAuditApproval,
		model.AuditTargetMember:    s.applyMemberAuditApproval,
		model.AuditTargetSignup:    s.applySignupAuditApproval,
	}
	reason := strings.TrimSpace(req.Reason)

	err = s.repo.DB.Transaction(func(tx *gorm.DB) error {
		handler, ok := auditHandlerMap[record.TargetType]
		if !ok {
			return errors.New("不支持的审核目标类型")
		}

		if err := handler(tx, record); err != nil {
			return err
		}

		updates := map[string]any{
			"auditor_id":    auditorID,
			"audit_result":  model.ResolveAuditResult(model.AuditStatusApproved),
			"reject_reason": reason,
			"audit_time":    time.Now(),
			"status":        model.AuditStatusApproved,
		}
		if record.TargetID > 0 {
			updates["target_id"] = record.TargetID
		}
		return s.repo.UpdateAuditRecordByID(tx, record.ID, updates)
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("审核通过失败: 审核目标不存在, record_id=%d", record.ID)
			return nil, errors.New("审核目标不存在")
		}
		log.Error("审核通过失败: 事务执行异常: %v, record_id=%d", err, record.ID)
		return nil, err
	}
	log.Info("审核通过成功: record_id=%d target_type=%d target_id=%d auditor_id=%d", record.ID, record.TargetType, record.TargetID, auditorID)

	return &resp, nil
}

// AuditRejection 驳回指定审核记录并落库驳回结果。
func (s *AuditService) AuditRejection(req *api.AuditRejectionRequest) (*api.AuditRejectionResponse, error) {
	var resp api.AuditRejectionResponse
	if req == nil {
		log.Warn("审核驳回失败: 请求为空")
		return nil, errors.New("请求不能为空")
	}
	if req.Id <= 0 {
		log.Warn("审核驳回失败: 审核记录ID为空")
		return nil, errors.New("审核记录ID不能为空")
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		log.Warn("审核驳回失败: 驳回原因为空, record_id=%d", req.Id)
		return nil, errors.New("驳回原因不能为空")
	}
	record, err := s.repo.GetAuditRecordByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("审核驳回失败: 审核记录不存在, record_id=%d", req.Id)
			return nil, errors.New("审核记录不存在")
		}
		log.Error("审核驳回失败: 查询审核记录异常: %v, record_id=%d", err, req.Id)
		return nil, err
	}

	if err := ensureAuditRecordPending(record); err != nil {
		log.Warn("审核驳回失败: 审核记录不可处理, record_id=%d status=%d audit_result=%d", record.ID, record.Status, record.AuditResult)
		return nil, err
	}

	auditorID, err := s.getAuditOperatorID()
	if err != nil {
		log.Warn("审核驳回失败: 获取审核人失败, record_id=%d err=%v", record.ID, err)
		return nil, err
	}

	err = s.repo.DB.Transaction(func(tx *gorm.DB) error {
		if err := s.applyAuditRejectionSideEffects(tx, record); err != nil {
			return err
		}

		updates := map[string]any{
			"auditor_id":    auditorID,
			"audit_result":  model.ResolveAuditResult(model.AuditStatusRejected),
			"reject_reason": reason,
			"audit_time":    time.Now(),
			"status":        model.AuditStatusRejected,
		}
		return s.repo.UpdateAuditRecordByID(tx, record.ID, updates)
	})
	if err != nil {
		log.Error("审核驳回失败: 事务执行异常: %v, record_id=%d", err, record.ID)
		return nil, err
	}
	log.Info("审核驳回成功: record_id=%d target_type=%d target_id=%d auditor_id=%d", record.ID, record.TargetType, record.TargetID, auditorID)

	return &resp, nil
}

// applyVolunteerAuditApproval 处理志愿者目标的审核通过并回写志愿者信息。
func (s *AuditService) applyVolunteerAuditApproval(tx *gorm.DB, record *model.AuditRecord) error {
	volunteer, err := s.repo.FindVolunteerByID(tx, record.TargetID)
	if err != nil {
		return err
	}

	updates := map[string]any{}
	if record.OperationType != model.OperationTypeUpdate {
		return errors.New("志愿者审核仅支持更新操作")
	}

	scene, err := resolveVolunteerUpdateAuditScene(record.NewContent)
	if err != nil {
		return err
	}

	switch scene {
	case model.AuditSceneVolunteerProfileUpdate:
		// 资料变更审核：按 new_content 回写主表。
		newPayload, parseErr := unmarshalVolunteerProfileChangeAuditPayload(record.NewContent)
		if parseErr != nil {
			return parseErr
		}
		updates, err = newPayload.BuildVolunteerUpdates()
		if err != nil {
			return err
		}
	case model.AuditSceneVolunteerRealNameVerify:
		newPayload, parseErr := unmarshalVolunteerRealNameVerifyAuditPayload(record.NewContent)
		if parseErr != nil {
			return parseErr
		}
		updates, err = newPayload.BuildVolunteerApprovalUpdates()
		if err != nil {
			return err
		}
		updates["audit_status"] = model.VolunteerAuditStatusApproved
	default:
		return errors.New("不支持的志愿者审核场景")
	}

	return s.repo.UpdateVolunteer(tx, volunteer.ID, updates)
}

// applyAuditRejectionSideEffects 处理审核驳回后的附加副作用。
func (s *AuditService) applyAuditRejectionSideEffects(tx *gorm.DB, record *model.AuditRecord) error {
	if record.TargetType != model.AuditTargetVolunteer || record.OperationType != model.OperationTypeUpdate {
		return nil
	}

	scene, err := resolveVolunteerUpdateAuditScene(record.NewContent)
	if err != nil {
		return err
	}
	if scene != model.AuditSceneVolunteerRealNameVerify {
		return nil
	}

	volunteer, err := s.repo.FindVolunteerByID(tx, record.TargetID)
	if err != nil {
		return err
	}
	return s.repo.UpdateVolunteer(tx, volunteer.ID, map[string]any{
		"audit_status": model.VolunteerAuditStatusRejected,
	})
}

// applyMemberAuditApproval 处理组织成员目标的审核通过逻辑。
func (s *AuditService) applyMemberAuditApproval(tx *gorm.DB, record *model.AuditRecord) error {
	var member model.OrgMember
	if strings.TrimSpace(record.NewContent) != "" {
		if err := json.Unmarshal([]byte(record.NewContent), &member); err != nil {
			return err
		}
	}

	switch record.OperationType {
	case model.OperationTypeCreate:
		now := time.Now()
		if member.OrgID <= 0 || member.VolunteerID <= 0 {
			return errors.New("成员关系快照无效")
		}

		member.ID = 0
		member.Status = model.MemberStatusActive
		if member.AppliedAt.IsZero() {
			member.AppliedAt = now
		}
		if member.JoinedAt == nil {
			member.JoinedAt = &now
		}
		if err := s.repo.CreateMembership(tx, &member); err != nil {
			return err
		}
		record.TargetID = member.ID
		return nil

	case model.OperationTypeUpdate:
		// Prefer audit target_id as the canonical target.
		memberID := record.TargetID
		if memberID <= 0 {
			// Backward-compatibility for historical records lacking target_id.
			memberID = member.ID
		}
		if memberID <= 0 {
			return errors.New("目标ID不能为空")
		}
		if record.TargetID > 0 && member.ID > 0 && member.ID != record.TargetID {
			return errors.New("目标ID不一致")
		}

		updates := map[string]any{
			"status": model.MemberStatusActive,
		}
		if member.OrgID > 0 {
			updates["org_id"] = member.OrgID
		}
		if member.VolunteerID > 0 {
			updates["volunteer_id"] = member.VolunteerID
		}
		if member.Role > 0 {
			updates["role"] = member.Role
		}
		if member.Status > 0 {
			updates["status"] = member.Status
		}
		if !member.AppliedAt.IsZero() {
			updates["applied_at"] = member.AppliedAt
		}
		if member.JoinedAt != nil {
			updates["joined_at"] = member.JoinedAt
		}
		if _, ok := updates["joined_at"]; !ok {
			now := time.Now()
			updates["joined_at"] = &now
		}

		if err := s.repo.UpdateMembershipFields(tx, memberID, updates); err != nil {
			return err
		}
		record.TargetID = memberID
		return nil

	case model.OperationTypeDelete:
		if record.TargetID <= 0 {
			return nil
		}
		return s.repo.UpdateMembershipFields(tx, record.TargetID, map[string]any{
			"status": model.MemberStatusLeft,
		})

	default:
		return nil
	}
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

		needIncrementPeople := false
		signup, err := s.repo.GetSignup(tx, signupSnapshot.ActivityID, signupSnapshot.VolunteerID)
		if err != nil {
			return err
		}

		if signup == nil {
			signup = &model.ActivitySignup{
				ActivityID:  signupSnapshot.ActivityID,
				VolunteerID: signupSnapshot.VolunteerID,
				Status:      model.ActivitySignupStatusSuccess,
			}
			if err := s.repo.CreateSignup(tx, signup); err != nil {
				return err
			}
			needIncrementPeople = true
		} else if signup.Status != model.ActivitySignupStatusSuccess {
			if err := s.repo.UpdateActivitySignupStatusByID(tx, signup.ID, model.ActivitySignupStatusSuccess); err != nil {
				return err
			}
			needIncrementPeople = true
		}

		if needIncrementPeople {
			if err := s.repo.IncrementActivityPeople(tx, signupSnapshot.ActivityID); err != nil {
				return err
			}
		}
		record.TargetID = signup.ID
		return nil
	}

	signup, err := s.repo.GetActivitySignupByID(tx, record.TargetID)
	if err != nil {
		return err
	}
	return s.repo.UpdateActivitySignupStatusByID(tx, signup.ID, model.ActivitySignupStatusSuccess)
}

// AuditRecordDetail 查询并返回单条审核记录详情。
func (s *AuditService) AuditRecordDetail(req *api.AuditRecordDetailRequest) (*api.AuditRecordDetailResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if req.Id <= 0 {
		return nil, errors.New("审核记录ID不能为空")
	}

	record, err := s.repo.GetAuditRecordByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("审核记录不存在")
		}
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

// getAuditOperatorID 获取审核操作人 ID 并校验其审核权限。
func (s *AuditService) getAuditOperatorID() (int64, error) {
	auditorID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Warn("获取审核人失败: 无法从上下文获取用户ID, err=%v", err)
		return 0, err
	}
	if auditorID <= 0 {
		log.Warn("获取审核人失败: 用户ID无效, user_id=%d", auditorID)
		return 0, errors.New("审核人无效")
	}

	account, err := s.repo.FindByID(s.repo.DB, auditorID)
	if err != nil {
		log.Error("获取审核人失败: 查询账号异常, user_id=%d err=%v", auditorID, err)
		return 0, err
	}
	if account.IdentityType != model.RegisterTypeOrganizationCode {
		log.Warn("获取审核人失败: 身份无权限, user_id=%d identity_type=%d", auditorID, account.IdentityType)
		return 0, errors.New("无权限执行审核")
	}

	return auditorID, nil
}

// ensureAuditRecordPending 校验审核记录是否处于待处理状态。
func ensureAuditRecordPending(record *model.AuditRecord) error {
	if !model.IsValidAuditTargetType(record.TargetType) {
		return errors.New("审核目标类型不合法")
	}
	if record.TargetID <= 0 && record.OperationType != model.OperationTypeCreate {
		return errors.New("目标ID不能为空")
	}
	if record.Status != model.AuditStatusPending || model.IsValidAuditResult(record.AuditResult) {
		return errors.New("审核记录已处理")
	}
	return nil
}
