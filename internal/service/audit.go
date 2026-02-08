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

type ApprovalHandler func(*gorm.DB, *model.AuditRecord) (int32, int32, error)

// VolunteerJoinOrgAuditList returns pending audits for volunteer join organization requests.
func (s *AuditService) VolunteerJoinOrgAuditList(req *api.PendingVolunteerJoinOrgAuditListRequest) (*api.PendingVolunteerJoinOrgAuditListResponse, error) {
	var resp api.PendingVolunteerJoinOrgAuditListResponse
	auditType := model.AuditTypeVolunteerJoinOrganization

	auditMap := map[string]any{
		"target_type = ?": auditType,
	}

	if req.Status != nil {
		auditMap["status in (?)"] = req.Status
	}

	// Search by keyword, skip if keyword is empty after trim.
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		targetIDs, err := s.repo.FindVolunteerIDsByKeyword(s.repo.DB, keyword)
		if err != nil {
			return nil, err
		}
		if len(targetIDs) == 0 {
			return &resp, nil
		}
		auditMap["target_id in (?)"] = targetIDs
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	offset := req.PageSize * (req.Page - 1)
	auditRecords, total, err := s.repo.GetAuditRecordsList(s.repo.DB, auditMap, req.PageSize, offset)
	if err != nil {
		return nil, err
	}

	if total == 0 {
		return &resp, nil
	}

	for _, record := range auditRecords {
		if record.TargetType != auditType {
			continue
		}

		member, err := s.repo.GetMembershipByID(s.repo.DB, record.TargetID)
		if err != nil {
			return nil, err
		}

		volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, member.VolunteerID)
		if err != nil {
			return nil, err
		}

		resp.List = append(resp.List, &api.PendingVolunteerJoinOrgAuditItem{
			TargetId:  record.TargetID,
			Status:    member.Status,
			Title:     volunteer.RealName,
			SubTitle:  member.TableName(),
			CreatedAt: record.CreatedAt.Format(util.DateTimeLayout),
		})
	}
	resp.Total = int32(total)
	return &resp, nil
}

// AuditApproval approves one audit target.
func (s *AuditService) AuditApproval(req *api.AuditApprovalRequest) (*api.AuditApprovalResponse, error) {
	var resp api.AuditApprovalResponse
	if req == nil {
		return nil, errors.New("request is required")
	}
	if req.Id <= 0 {
		return nil, errors.New("id is required")
	}

	record, err := s.repo.GetAuditRecordByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("audit record not found")
		}
		return nil, err
	}

	if !model.IsValidAuditTargetType(record.TargetType) {
		return nil, errors.New("targetType is invalid")
	}
	if record.TargetID <= 0 {
		return nil, errors.New("targetId is required")
	}
	if model.IsValidAuditResult(record.AuditResult) {
		return nil, errors.New("audit record already processed")
	}
	auditorID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}
	if auditorID <= 0 {
		return nil, errors.New("invalid auditor")
	}
	account, err := s.repo.FindByID(s.repo.DB, auditorID)
	if err != nil {
		return nil, err
	}
	if account.IdentityType != model.RegisterTypeOrganizationCode {
		return nil, errors.New("permission denied")
	}

	auditHandlerMap := map[int32]ApprovalHandler{
		model.AuditTargetVolunteer: s.applyVolunteerAuditApproval,
		model.AuditTargetOrg:       s.applyOrgAuditApproval,
		model.AuditTargetMember:    s.applyMemberAuditApproval,
		model.AuditTargetSignup:    s.applySignupAuditApproval,
	}
	err = s.repo.Transaction(func(tx *gorm.DB) error {
		handler, ok := auditHandlerMap[record.TargetType]
		if !ok {
			return errors.New("unsupported targetType")
		}

		oldStatus, newStatus, err := handler(tx, record)
		if err != nil {
			return err
		}
		updates := map[string]any{
			"auditor_id":    auditorID,
			"old_status":    oldStatus,
			"new_status":    newStatus,
			"audit_result":  model.AuditResultPass,
			"reject_reason": "",
			"audit_time":    time.Now(),
		}

		return s.repo.UpdateAuditRecordByID(tx, record.ID, updates)
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("target not found")
		}
		return nil, err
	}

	return &resp, nil
}

// AuditRejection rejects one audit target.
func (s *AuditService) AuditRejection(req *api.AuditRejectionRequest) (*api.AuditRejectionResponse, error) {
	var resp api.AuditRejectionResponse
	if req == nil {
		return nil, errors.New("request is required")
	}
	if req.Id <= 0 {
		return nil, errors.New("id is required")
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, errors.New("reject reason is required")
	}

	record, err := s.repo.GetAuditRecordByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("audit record not found")
		}
		return nil, err
	}

	if !model.IsValidAuditTargetType(record.TargetType) {
		return nil, errors.New("targetType is invalid")
	}
	if record.TargetID <= 0 {
		return nil, errors.New("targetId is required")
	}

	auditorID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}
	if auditorID <= 0 {
		return nil, errors.New("invalid auditor")
	}
	account, err := s.repo.FindByID(s.repo.DB, auditorID)
	if err != nil {
		return nil, err
	}
	if account.IdentityType != model.RegisterTypeOrganizationCode {
		return nil, errors.New("permission denied")
	}

	updates := map[string]any{
		"auditor_id":    auditorID,
		"new_status":    model.ResolveAuditStatus(model.AuditResultReject),
		"audit_result":  model.AuditResultReject,
		"reject_reason": reason,
		"audit_time":    time.Now(),
	}
	if err := s.repo.UpdateAuditRecordByID(s.repo.DB, record.ID, updates); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *AuditService) applyVolunteerAuditApproval(tx *gorm.DB, record *model.AuditRecord) (int32, int32, error) {
	volunteer, err := s.repo.FindVolunteerByID(tx, record.TargetID)
	if err != nil {
		return 0, 0, err
	}

	oldStatus := volunteer.AuditStatus
	newStatus := model.AuditStatusApproved
	if err := s.repo.UpdateVolunteer(tx, volunteer.ID, map[string]any{
		"audit_status": newStatus,
	}); err != nil {
		return 0, 0, err
	}

	return oldStatus, newStatus, nil
}

func (s *AuditService) applyOrgAuditApproval(tx *gorm.DB, record *model.AuditRecord) (int32, int32, error) {
	organization, err := s.repo.GetOrganizationByID(tx, record.TargetID)
	if err != nil {
		return 0, 0, err
	}

	oldStatus := organization.AuditStatus
	newStatus := model.AuditStatusApproved
	if err := s.repo.UpdateOrganization(tx, organization.ID, map[string]any{
		"audit_status": newStatus,
	}); err != nil {
		return 0, 0, err
	}

	return oldStatus, newStatus, nil
}

func (s *AuditService) applyMemberAuditApproval(tx *gorm.DB, record *model.AuditRecord) (int32, int32, error) {
	var member model.OrgMember
	if err := json.Unmarshal([]byte(record.NewContent), &member); err != nil {
		return 0, 0, err
	}

	switch record.OperationType {
	case model.OperationTypeCreate:
		oldStatus := record.OldStatus
		newStatus := model.MemberStatusActive
		now := time.Now()

		member.ID = 0
		member.Status = newStatus
		if member.JoinedAt == nil {
			member.JoinedAt = &now
		}
		if err := s.repo.CreateMembership(tx, &member); err != nil {
			return 0, 0, err
		}
		return oldStatus, newStatus, nil
	case model.OperationTypeUpdate:
		memberID := member.ID
		if memberID <= 0 {
			memberID = record.TargetID
		}

		oldStatus := record.OldStatus
		newStatus := record.NewStatus
		if member.Status > 0 {
			newStatus = member.Status
		}

		updates := map[string]any{}
		if member.OrgID > 0 {
			updates["org_id = ?"] = member.OrgID
		}
		if member.VolunteerID > 0 {
			updates["volunteer_id = ?"] = member.VolunteerID
		}
		if member.Role > 0 {
			updates["role = ?"] = member.Role
		}
		if member.Status > 0 {
			updates["status = ?"] = member.Status
		}
		if !member.AppliedAt.IsZero() {
			updates["applied_at = ?"] = member.AppliedAt
		}
		if member.JoinedAt != nil {
			updates["joined_at = ?"] = member.JoinedAt
		}

		if len(updates) > 0 {
			if err := s.repo.UpdateMembershipFields(tx, memberID, updates); err != nil {
				return 0, 0, err
			}
		}
		return oldStatus, newStatus, nil
	default:
		return record.OldStatus, record.NewStatus, nil
	}
}

func (s *AuditService) applySignupAuditApproval(tx *gorm.DB, record *model.AuditRecord) (int32, int32, error) {
	signup, err := s.repo.GetActivitySignupByID(tx, record.TargetID)
	if err != nil {
		return 0, 0, err
	}

	oldStatus := signup.Status
	newStatus := model.AuditStatusApproved
	if err := s.repo.UpdateActivitySignupStatusByID(tx, signup.ID, newStatus); err != nil {
		return 0, 0, err
	}

	return oldStatus, newStatus, nil
}

// AuditRecordDetail returns one audit record.
func (s *AuditService) AuditRecordDetail(req *api.AuditRecordDetailRequest) (*api.AuditRecordDetailResponse, error) {
	if req.Id <= 0 {
		return nil, errors.New("id is required")
	}

	record, err := s.repo.GetAuditRecordByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("audit record not found")
		}
		return nil, err
	}

	var oldStatus int32
	if record.OldStatus != 0 {
		oldStatus = record.OldStatus
	}

	return &api.AuditRecordDetailResponse{
		Record: &api.AuditRecordDetail{
			Id:           record.ID,
			TargetType:   record.TargetType,
			TargetId:     record.TargetID,
			AuditorId:    record.AuditorID,
			OldStatus:    oldStatus,
			NewStatus:    record.NewStatus,
			OldContent:   derefString(record.OldContent),
			NewContent:   derefString(record.NewContent),
			AuditResult:  record.AuditResult,
			RejectReason: derefString(record.RejectReason),
			AuditTime:    record.AuditTime.Format(util.DateTimeLayout),
			CreatedAt:    record.CreatedAt.Format(util.DateTimeLayout),
		},
	}, nil
}

func derefString(v string) string {
	if v == "" {
		return ""
	}
	return v
}
