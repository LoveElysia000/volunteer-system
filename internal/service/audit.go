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

// VolunteerJoinOrgAuditList returns pending audits for volunteer join organization requests.
func (s *AuditService) VolunteerJoinOrgAuditList(req *api.PendingVolunteerJoinOrgAuditListRequest) (*api.PendingVolunteerJoinOrgAuditListResponse, error) {
	if req == nil {
		log.Warn("待审核列表查询失败: 请求为空")
		return nil, errors.New("请求不能为空")
	}

	resp := &api.PendingVolunteerJoinOrgAuditListResponse{}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	auditMap := map[string]any{
		"target_type = ?": model.AuditTypeVolunteerJoinOrganization,
	}
	if len(req.Status) > 0 {
		auditMap["status in (?)"] = req.Status
	}

	offset := req.PageSize * (req.Page - 1)
	auditRecords, total, err := s.repo.GetAuditRecordsList(s.repo.DB, auditMap, req.PageSize, offset)
	if err != nil {
		log.Error("待审核列表查询失败: %v, page=%d, pageSize=%d, status=%v", err, req.Page, req.PageSize, req.Status)
		return nil, err
	}
	if total == 0 {
		return resp, nil
	}

	items := make([]*api.PendingVolunteerJoinOrgAuditItem, 0, len(auditRecords))
	for _, record := range auditRecords {
		if record == nil {
			continue
		}
		if strings.TrimSpace(record.NewContent) == "" {
			continue
		}

		var snapshot model.OrgMember
		if err := json.Unmarshal([]byte(record.NewContent), &snapshot); err != nil {
			log.Warn("待审核记录快照解析失败: record_id=%d", record.ID)
			return nil, errors.New("成员关系快照无效")
		}
		if snapshot.VolunteerID <= 0 || snapshot.OrgID <= 0 {
			log.Warn("待审核记录快照字段无效: record_id=%d volunteer_id=%d org_id=%d", record.ID, snapshot.VolunteerID, snapshot.OrgID)
			return nil, errors.New("成员关系快照无效")
		}

		volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, snapshot.VolunteerID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Warn("待审核记录关联志愿者不存在: record_id=%d volunteer_id=%d", record.ID, snapshot.VolunteerID)
				return nil, errors.New("志愿者不存在")
			}
			log.Error("查询待审核记录关联志愿者失败: %v, record_id=%d volunteer_id=%d", err, record.ID, snapshot.VolunteerID)
			return nil, err
		}

		organization, err := s.repo.GetOrganizationByID(s.repo.DB, snapshot.OrgID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Warn("待审核记录关联组织不存在: record_id=%d org_id=%d", record.ID, snapshot.OrgID)
				return nil, errors.New("组织不存在")
			}
			log.Error("查询待审核记录关联组织失败: %v, record_id=%d org_id=%d", err, record.ID, snapshot.OrgID)
			return nil, err
		}

		targetID := record.TargetID
		if targetID <= 0 {
			targetID = snapshot.ID
		}
		items = append(items, &api.PendingVolunteerJoinOrgAuditItem{
			TargetId:  targetID,
			Status:    record.Status,
			Title:     volunteer.RealName,
			SubTitle:  organization.OrgName,
			CreatedAt: record.CreatedAt.Format(util.DateTimeLayout),
		})
	}

	resp.Total = int32(total)
	resp.List = items
	return resp, nil
}

// AuditApproval approves one audit target.
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

// AuditRejection rejects one audit target.
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

	updates := map[string]any{
		"auditor_id":    auditorID,
		"audit_result":  model.ResolveAuditResult(model.AuditStatusRejected),
		"reject_reason": reason,
		"audit_time":    time.Now(),
		"status":        model.AuditStatusRejected,
	}
	if err := s.repo.UpdateAuditRecordByID(s.repo.DB, record.ID, updates); err != nil {
		log.Error("审核驳回失败: 更新审核记录异常: %v, record_id=%d", err, record.ID)
		return nil, err
	}
	log.Info("审核驳回成功: record_id=%d target_type=%d target_id=%d auditor_id=%d", record.ID, record.TargetType, record.TargetID, auditorID)

	return &resp, nil
}

func (s *AuditService) applyVolunteerAuditApproval(tx *gorm.DB, record *model.AuditRecord) error {
	volunteer, err := s.repo.FindVolunteerByID(tx, record.TargetID)
	if err != nil {
		return err
	}
	return s.repo.UpdateVolunteer(tx, volunteer.ID, map[string]any{
		"audit_status": model.VolunteerAuditStatusApproved,
	})
}

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
		return s.repo.CreateMembership(tx, &member)

	case model.OperationTypeUpdate:
		memberID := member.ID
		if memberID <= 0 {
			memberID = record.TargetID
		}
		if memberID <= 0 {
			return errors.New("目标ID不能为空")
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

		return s.repo.UpdateMembershipFields(tx, memberID, updates)

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
		return nil
	}

	signup, err := s.repo.GetActivitySignupByID(tx, record.TargetID)
	if err != nil {
		return err
	}
	return s.repo.UpdateActivitySignupStatusByID(tx, signup.ID, model.ActivitySignupStatusSuccess)
}

// AuditRecordDetail returns one audit record.
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
