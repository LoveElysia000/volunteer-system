package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

type MembershipService struct {
	Service
}

func NewMembershipService(ctx context.Context, c *app.RequestContext) *MembershipService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &MembershipService{
		Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
}

// VolunteerJoinOrganization submits a join request for an organization.
func (s *MembershipService) VolunteerJoinOrganization(req *api.VolunteerJoinRequest) (*api.VolunteerJoinResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if req.OrganizationId <= 0 {
		return nil, errors.New("组织ID不能为空")
	}

	// If current user is a volunteer, enforce volunteerId match.
	if userID, err := middleware.GetUserIDInt(s.c); err == nil {
		if volunteer, err := s.repo.FindVolunteerByAccountID(s.repo.DB, userID); err == nil && volunteer != nil {
			if req.VolunteerId > 0 && req.VolunteerId != volunteer.ID {
				return nil, errors.New("无权操作该志愿者")
			}
			req.VolunteerId = volunteer.ID
		}
	}
	if req.VolunteerId <= 0 {
		return nil, errors.New("志愿者ID不能为空")
	}

	// Validate organization exists.
	organization, err := s.repo.GetOrganizationByID(s.repo.DB, req.OrganizationId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("组织不存在")
		}
		return nil, err
	}

	// Validate volunteer exists.
	volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, req.VolunteerId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("志愿者不存在")
		}
		return nil, err
	}

	orgID := organization.ID
	volunteerID := volunteer.ID

	existing, err := s.repo.FindMembershipByOrgAndVolunteer(s.repo.DB, orgID, volunteerID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("成员关系已存在或正在审核中")
	}

	hasPendingCreateAudit, err := s.hasPendingMemberCreateAudit(s.repo.DB, orgID, volunteerID)
	if err != nil {
		return nil, err
	}
	if hasPendingCreateAudit {
		return nil, errors.New("成员关系已存在或正在审核中")
	}

	now := time.Now()
	newMember := &model.OrgMember{
		OrgID:       orgID,
		VolunteerID: volunteerID,
		Role:        model.MemberRoleMember,
		Status:      model.MemberStatusActive,
		AppliedAt:   now,
	}

	newContent, err := json.Marshal(newMember)
	if err != nil {
		return nil, err
	}

	record := &model.AuditRecord{
		TargetType:    model.AuditTargetMember,
		TargetID:      0,
		AuditorID:     0,
		OldContent:    "{}",
		NewContent:    string(newContent),
		AuditResult:   0,
		RejectReason:  "",
		AuditTime:     now,
		OperationType: model.OperationTypeCreate,
		Status:        model.AuditStatusPending,
	}
	if err := s.repo.CreateAuditRecord(s.repo.DB, record); err != nil {
		return nil, err
	}

	return &api.VolunteerJoinResponse{
		Status:  model.MemberStatusPending,
		Message: "application submitted",
	}, nil
}

func (s *MembershipService) hasPendingMemberCreateAudit(db *gorm.DB, orgID, volunteerID int64) (bool, error) {
	queryMap := map[string]any{
		"target_type = ?":    model.AuditTargetMember,
		"operation_type = ?": model.OperationTypeCreate,
		"status = ?":         model.AuditStatusPending,
	}
	records, _, err := s.repo.GetAuditRecordsList(db, queryMap, 0, 0)
	if err != nil {
		return false, err
	}

	for _, record := range records {
		if record == nil {
			continue
		}

		var member model.OrgMember
		if err := json.Unmarshal([]byte(record.NewContent), &member); err != nil {
			continue
		}

		if member.OrgID == orgID && member.VolunteerID == volunteerID {
			return true, nil
		}
	}

	return false, nil
}

// VolunteerLeaveOrganization submits a leave request for an organization.
func (s *MembershipService) VolunteerLeaveOrganization(req *api.VolunteerLeaveRequest) (*api.VolunteerLeaveResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}

	if req.MembershipId <= 0 {
		return nil, errors.New("成员关系ID不能为空")
	}

	member, err := s.repo.GetMembershipByID(s.repo.DB, req.MembershipId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("成员关系不存在")
		}
		return nil, err
	}

	// If current user is a volunteer, enforce ownership.
	if userID, err := middleware.GetUserIDInt(s.c); err == nil {
		volunteer, err := s.repo.FindVolunteerByAccountID(s.repo.DB, userID)
		if err == nil && volunteer != nil && member.VolunteerID != volunteer.ID {
			return nil, errors.New("无权操作该成员关系")
		}
	}

	if member.Status == model.MemberStatusLeft {
		return nil, errors.New("该成员已退出组织")
	}

	queryMap := map[string]any{
		"target_type = ?": model.AuditTargetMember,
		"target_id = ?":   member.ID,
		"status = ?":      model.AuditStatusPending,
	}
	records, _, err := s.repo.GetAuditRecordsList(s.repo.DB, queryMap, 1, 0)
	if err != nil {
		return nil, err
	}
	if len(records) > 0 {
		return nil, errors.New("该成员关系已有待审核申请")
	}

	oldContent, err := json.Marshal(member)
	if err != nil {
		return nil, err
	}

	newMember := *member
	newMember.Status = model.MemberStatusLeft
	newContent, err := json.Marshal(&newMember)
	if err != nil {
		return nil, err
	}

	record := &model.AuditRecord{
		TargetType:    model.AuditTargetMember,
		TargetID:      member.ID,
		AuditorID:     0,
		OldContent:    string(oldContent),
		NewContent:    string(newContent),
		AuditResult:   0,
		RejectReason:  "",
		AuditTime:     time.Now(),
		OperationType: model.OperationTypeDelete,
		Status:        model.AuditStatusPending,
	}
	if err := s.repo.CreateAuditRecord(s.repo.DB, record); err != nil {
		return nil, err
	}

	return &api.VolunteerLeaveResponse{
		Message: "application submitted",
	}, nil
}

// GetOrganizationMembers returns members for an organization.
func (s *MembershipService) GetOrganizationMembers(req *api.OrganizationMembersRequest) (*api.OrganizationMembersResponse, error) {
	if req.OrganizationId <= 0 {
		return nil, errors.New("组织ID不能为空")
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// Permission: only organization owner.
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}
	org, err := s.repo.FindOrganizationByAccountID(s.repo.DB, userID)
	if err != nil || org == nil || org.ID != req.OrganizationId {
		return nil, errors.New("无权操作该组织")
	}

	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	members, total, err := s.repo.GetOrganizationMembers(s.repo.DB, req.OrganizationId, req.Status, req.Role, req.Keyword, pageSize, offset)
	if err != nil {
		return nil, err
	}

	organization, err := s.repo.GetOrganizationByID(s.repo.DB, req.OrganizationId)
	if err != nil {
		return nil, err
	}

	volunteerNameMap := make(map[int64]string)
	if len(members) > 0 {
		volunteerIDs := make([]int64, 0, len(members))
		seen := make(map[int64]struct{}, len(members))
		for _, m := range members {
			if _, exists := seen[m.VolunteerID]; exists {
				continue
			}
			seen[m.VolunteerID] = struct{}{}
			volunteerIDs = append(volunteerIDs, m.VolunteerID)
		}

		volunteers, err := s.repo.GetVolunteersByIDs(s.repo.DB, volunteerIDs)
		if err != nil {
			return nil, err
		}
		for _, volunteer := range volunteers {
			volunteerNameMap[volunteer.ID] = volunteer.RealName
		}
	}

	resp := &api.OrganizationMembersResponse{
		Total: int32(total),
		List:  make([]*api.MemberInfo, 0, len(members)),
	}

	for _, m := range members {
		item := &api.MemberInfo{
			MembershipId:     m.ID,
			VolunteerId:      m.VolunteerID,
			VolunteerName:    volunteerNameMap[m.VolunteerID],
			VolunteerCode:    "",
			OrganizationId:   m.OrgID,
			OrganizationName: organization.OrgName,
			Status:           m.Status,
			Role:             m.Role,
			Position:         "",
			Motivation:       "",
			ExpectedHours:    "",
			JoinDate:         util.FormatJoinDate(m.JoinedAt, m.AppliedAt),
			ReviewDate:       "",
			ReviewComment:    "",
			LeaveDate:        "",
			LeaveReason:      "",
			CreatedAt:        m.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:        m.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		resp.List = append(resp.List, item)
	}

	return resp, nil
}

// GetVolunteerOrganizations returns organizations joined by a volunteer.
func (s *MembershipService) GetVolunteerOrganizations(req *api.VolunteerOrganizationsRequest) (*api.VolunteerOrganizationsResponse, error) {
	if req.VolunteerId <= 0 {
		return nil, errors.New("志愿者ID不能为空")
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// Permission: volunteer can only access own memberships.
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}
	volunteer, err := s.repo.FindVolunteerByAccountID(s.repo.DB, userID)
	if err != nil || volunteer == nil || volunteer.ID != req.VolunteerId {
		return nil, errors.New("无权操作该志愿者")
	}

	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	list, total, err := s.repo.GetVolunteerOrganizations(s.repo.DB, req.VolunteerId, req.Status, pageSize, offset)
	if err != nil {
		return nil, err
	}

	orgInfoMap := make(map[int64]*model.Organization)
	if len(list) > 0 {
		orgIDs := make([]int64, 0, len(list))
		seen := make(map[int64]struct{}, len(list))
		for _, member := range list {
			if _, exists := seen[member.OrgID]; exists {
				continue
			}
			seen[member.OrgID] = struct{}{}
			orgIDs = append(orgIDs, member.OrgID)
		}

		organizations, err := s.repo.GetOrganizationsByIDs(s.repo.DB, orgIDs)
		if err != nil {
			return nil, err
		}
		for _, org := range organizations {
			orgInfoMap[org.ID] = org
		}
	}

	resp := &api.VolunteerOrganizationsResponse{
		Total: int32(total),
		List:  make([]*api.OrganizationMemberInfo, 0, len(list)),
	}

	for _, m := range list {
		organizationName := ""
		organizationCode := ""
		if org, ok := orgInfoMap[m.OrgID]; ok && org != nil {
			organizationName = org.OrgName
			organizationCode = org.LicenseCode
		}

		item := &api.OrganizationMemberInfo{
			MembershipId:     m.ID,
			OrganizationId:   m.OrgID,
			OrganizationName: organizationName,
			OrganizationCode: organizationCode,
			Status:           m.Status,
			Role:             m.Role,
			Position:         "",
			JoinDate:         util.FormatJoinDate(m.JoinedAt, m.AppliedAt),
			ReviewDate:       "",
			ReviewComment:    "",
			CreatedAt:        m.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:        m.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		resp.List = append(resp.List, item)
	}

	return resp, nil
}

// UpdateMemberStatus updates membership status by organization owner.
func (s *MembershipService) UpdateMemberStatus(req *api.MemberStatusUpdateRequest) (*api.MemberStatusUpdateResponse, error) {
	if req.MembershipId <= 0 {
		return nil, errors.New("成员关系ID不能为空")
	}
	if req.Status <= 0 {
		return nil, errors.New("状态不能为空")
	}
	if req.Status < model.MemberStatusPending || req.Status > model.MemberStatusLeft {
		return nil, errors.New("状态值不合法")
	}

	member, err := s.repo.GetMembershipByID(s.repo.DB, req.MembershipId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("成员关系不存在")
		}
		return nil, err
	}

	// Permission: only organization owner for the membership.
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}
	org, err := s.repo.FindOrganizationByAccountID(s.repo.DB, userID)
	if err != nil || org == nil || org.ID != member.OrgID {
		return nil, errors.New("无权操作该组织")
	}

	updates := map[string]any{
		"status": req.Status,
	}
	if req.Status == model.MemberStatusActive {
		now := time.Now()
		updates["joined_at"] = &now
	}

	if err := s.repo.UpdateMembershipFields(s.repo.DB, member.ID, updates); err != nil {
		return nil, err
	}

	// Save audit record if reviewComment provided.
	if req.ReviewComment != "" {
		auditStatus := model.AuditStatusRejected
		if req.Status == model.MemberStatusActive {
			auditStatus = model.AuditStatusApproved
		}

		oldContent, err := json.Marshal(member)
		if err != nil {
			return nil, err
		}

		newMember := *member
		newMember.Status = req.Status
		if joinedAt, ok := updates["joined_at"].(*time.Time); ok {
			newMember.JoinedAt = joinedAt
		}

		newContent, err := json.Marshal(&newMember)
		if err != nil {
			return nil, err
		}

		record := &model.AuditRecord{
			TargetType:    model.AuditTargetMember,
			TargetID:      member.ID,
			AuditorID:     userID,
			OldContent:    string(oldContent),
			NewContent:    string(newContent),
			AuditResult:   model.ResolveAuditResult(auditStatus),
			RejectReason:  req.ReviewComment,
			AuditTime:     time.Now(),
			OperationType: model.OperationTypeUpdate,
			Status:        auditStatus,
		}
		_ = s.repo.CreateAuditRecord(s.repo.DB, record)
	}

	return &api.MemberStatusUpdateResponse{
		Message: "status updated",
	}, nil
}

// MembershipStats returns summary counts.
func (s *MembershipService) MembershipStats(req *api.MembershipStatsRequest) (*api.MembershipStatsResponse, error) {
	orgID := req.OrganizationId
	if orgID <= 0 {
		// Default to current organization if possible.
		userID, err := middleware.GetUserIDInt(s.c)
		if err != nil {
			return nil, err
		}
		org, err := s.repo.FindOrganizationByAccountID(s.repo.DB, userID)
		if err != nil || org == nil {
			return nil, errors.New("组织ID不能为空")
		}
		orgID = org.ID
	} else {
		userID, err := middleware.GetUserIDInt(s.c)
		if err != nil {
			return nil, err
		}
		org, err := s.repo.FindOrganizationByAccountID(s.repo.DB, userID)
		if err != nil || org == nil || org.ID != orgID {
			return nil, errors.New("无权操作该组织")
		}
	}

	statusCounts, total, err := s.repo.GetMembershipStatusCounts(s.repo.DB, orgID)
	if err != nil {
		return nil, err
	}

	resp := &api.MembershipStatsResponse{
		PendingCount:   int32(statusCounts[model.MemberStatusPending]),
		ActiveCount:    int32(statusCounts[model.MemberStatusActive]),
		InactiveCount:  int32(statusCounts[model.MemberStatusLeft]),
		SuspendedCount: int32(statusCounts[model.MemberStatusRejected]),
		TotalCount:     int32(total),
	}

	return resp, nil
}
