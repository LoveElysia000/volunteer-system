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

func hasOrganizationPermission(organizations []*model.Organization, organizationID int64) bool {
	for _, org := range organizations {
		if org != nil && org.ID == organizationID {
			return true
		}
	}
	return false
}

// VolunteerJoinOrganization submits a join request for an organization.
func (s *MembershipService) VolunteerJoinOrganization(req *api.VolunteerJoinRequest) (*api.VolunteerJoinResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if req.OrganizationId <= 0 {
		return nil, errors.New("组织ID不能为空")
	}

	// 仅允许志愿者本人发起加入组织申请。
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("提交加入组织申请失败: 获取当前用户失败: %v", err)
		return nil, err
	}

	currentVolunteer, err := s.repo.FindVolunteerByAccountID(s.repo.DB, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("仅志愿者可执行该操作")
		}
		log.Error("提交加入组织申请失败: 查询当前志愿者异常: %v, user_id=%d", err, userID)
		return nil, err
	}

	if req.VolunteerId > 0 && req.VolunteerId != currentVolunteer.ID {
		return nil, errors.New("无权操作该志愿者")
	}
	req.VolunteerId = currentVolunteer.ID

	if req.VolunteerId <= 0 {
		return nil, errors.New("志愿者ID不能为空")
	}

	// Validate organization exists.
	organization, err := s.repo.GetOrganizationByID(s.repo.DB, req.OrganizationId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error("提交加入组织申请失败: 组织不存在, organization_id=%d volunteer_id=%d", req.OrganizationId, req.VolunteerId)
			return nil, errors.New("组织不存在")
		}
		log.Error("提交加入组织申请失败: 查询组织异常: %v, organization_id=%d volunteer_id=%d", err, req.OrganizationId, req.VolunteerId)
		return nil, err
	}

	// Validate volunteer exists.
	volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, req.VolunteerId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error("提交加入组织申请失败: 志愿者不存在, organization_id=%d volunteer_id=%d", req.OrganizationId, req.VolunteerId)
			return nil, errors.New("志愿者不存在")
		}
		log.Error("提交加入组织申请失败: 查询志愿者异常: %v, organization_id=%d volunteer_id=%d", err, req.OrganizationId, req.VolunteerId)
		return nil, err
	}

	orgID := organization.ID
	volunteerID := volunteer.ID

	existing, err := s.repo.FindMembershipByOrgAndVolunteer(s.repo.DB, orgID, volunteerID)
	if err != nil {
		log.Error("提交加入组织申请失败: 查询成员关系异常: %v, organization_id=%d volunteer_id=%d", err, orgID, volunteerID)
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("成员关系已存在或正在审核中")
	}

	hasPendingCreateAudit, err := s.hasPendingMemberCreateAudit(s.repo.DB, orgID, volunteerID)
	if err != nil {
		log.Error("提交加入组织申请失败: 查询待审核记录异常: %v, organization_id=%d volunteer_id=%d", err, orgID, volunteerID)
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
		log.Error("提交加入组织申请失败: 序列化新成员快照异常: %v, organization_id=%d volunteer_id=%d", err, orgID, volunteerID)
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
		log.Error("提交加入组织申请失败: 创建审核记录异常: %v, organization_id=%d volunteer_id=%d", err, orgID, volunteerID)
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
		log.Error("查询待审核创建成员记录失败: %v, organization_id=%d volunteer_id=%d", err, orgID, volunteerID)
		return false, err
	}

	for _, record := range records {
		if record == nil {
			continue
		}

		var member model.OrgMember
		if err := json.Unmarshal([]byte(record.NewContent), &member); err != nil {
			log.Error("解析待审核成员记录快照失败: record_id=%d err=%v", record.ID, err)
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

	// 仅允许志愿者本人发起退出组织申请。
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("提交退出组织申请失败: 获取当前用户失败: %v, membership_id=%d", err, req.MembershipId)
		return nil, err
	}
	currentVolunteer, err := s.repo.FindVolunteerByAccountID(s.repo.DB, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("仅志愿者可执行该操作")
		}
		log.Error("提交退出组织申请失败: 查询当前志愿者异常: %v, user_id=%d, membership_id=%d", err, userID, req.MembershipId)
		return nil, err
	}

	member, err := s.repo.GetMembershipByID(s.repo.DB, req.MembershipId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error("提交退出组织申请失败: 成员关系不存在, membership_id=%d", req.MembershipId)
			return nil, errors.New("成员关系不存在")
		}
		log.Error("提交退出组织申请失败: 查询成员关系异常: %v, membership_id=%d", err, req.MembershipId)
		return nil, err
	}

	if member.VolunteerID != currentVolunteer.ID {
		return nil, errors.New("无权操作该成员关系")
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
		log.Error("提交退出组织申请失败: 查询待审核记录异常: %v, membership_id=%d", err, member.ID)
		return nil, err
	}
	if len(records) > 0 {
		return nil, errors.New("该成员关系已有待审核申请")
	}

	oldContent, err := json.Marshal(member)
	if err != nil {
		log.Error("提交退出组织申请失败: 序列化旧成员快照异常: %v, membership_id=%d", err, member.ID)
		return nil, err
	}

	newMember := *member
	newMember.Status = model.MemberStatusLeft
	newContent, err := json.Marshal(&newMember)
	if err != nil {
		log.Error("提交退出组织申请失败: 序列化新成员快照异常: %v, membership_id=%d", err, member.ID)
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
		log.Error("提交退出组织申请失败: 创建审核记录异常: %v, membership_id=%d", err, member.ID)
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
		log.Error("查询组织成员列表失败: 获取当前用户失败: %v, organization_id=%d", err, req.OrganizationId)
		return nil, err
	}
	organizations, err := s.repo.FindOrganizationByAccountID(s.repo.DB, userID)
	if err != nil {
		log.Error("查询组织成员列表失败: 查询组织异常: %v, organization_id=%d user_id=%d", err, req.OrganizationId, userID)
		return nil, err
	}
	if !hasOrganizationPermission(organizations, req.OrganizationId) {
		return nil, errors.New("无权操作该组织")
	}

	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	members, total, err := s.repo.GetOrganizationMembers(s.repo.DB, req.OrganizationId, req.Status, req.Role, req.Keyword, pageSize, offset)
	if err != nil {
		log.Error("查询组织成员列表失败: 查询成员数据异常: %v, organization_id=%d page=%d page_size=%d", err, req.OrganizationId, req.Page, req.PageSize)
		return nil, err
	}

	organization, err := s.repo.GetOrganizationByID(s.repo.DB, req.OrganizationId)
	if err != nil {
		log.Error("查询组织成员列表失败: 查询组织信息异常: %v, organization_id=%d", err, req.OrganizationId)
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
			log.Error("查询组织成员列表失败: 批量查询志愿者异常: %v, organization_id=%d volunteer_count=%d", err, req.OrganizationId, len(volunteerIDs))
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
		log.Error("查询志愿者组织列表失败: 获取当前用户失败: %v, volunteer_id=%d", err, req.VolunteerId)
		return nil, err
	}
	volunteer, err := s.repo.FindVolunteerByAccountID(s.repo.DB, userID)
	if err != nil || volunteer == nil || volunteer.ID != req.VolunteerId {
		if err != nil {
			log.Error("查询志愿者组织列表失败: 查询志愿者异常: %v, volunteer_id=%d user_id=%d", err, req.VolunteerId, userID)
		}
		return nil, errors.New("无权操作该志愿者")
	}

	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	list, total, err := s.repo.GetVolunteerOrganizations(s.repo.DB, req.VolunteerId, req.Status, pageSize, offset)
	if err != nil {
		log.Error("查询志愿者组织列表失败: 查询成员关系异常: %v, volunteer_id=%d page=%d page_size=%d", err, req.VolunteerId, req.Page, req.PageSize)
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
			log.Error("查询志愿者组织列表失败: 批量查询组织异常: %v, volunteer_id=%d org_count=%d", err, req.VolunteerId, len(orgIDs))
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
	if req.AccountId <= 0 {
		return nil, errors.New("组织管理者ID不能为空")
	}
	if req.MembershipId <= 0 {
		return nil, errors.New("成员关系ID不能为空")
	}
	if req.Status <= 0 {
		return nil, errors.New("状态不能为空")
	}
	// 组织管理者可直接修改成员状态，但不能直接置为待审核。
	if req.Status != model.MemberStatusActive &&
		req.Status != model.MemberStatusRejected &&
		req.Status != model.MemberStatusLeft {
		return nil, errors.New("状态值不合法")
	}

	member, err := s.repo.GetMembershipByID(s.repo.DB, req.MembershipId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error("更新成员状态失败: 成员关系不存在, membership_id=%d", req.MembershipId)
			return nil, errors.New("成员关系不存在")
		}
		log.Error("更新成员状态失败: 查询成员关系异常: %v, membership_id=%d", err, req.MembershipId)
		return nil, err
	}

	// Permission: only organization owner for the membership.
	organization, err := s.repo.GetOrganizationByID(s.repo.DB, member.OrgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error("更新成员状态失败: 组织不存在, membership_id=%d org_id=%d", req.MembershipId, member.OrgID)
			return nil, errors.New("组织不存在")
		}
		log.Error("更新成员状态失败: 查询组织异常: %v, membership_id=%d org_id=%d account_id=%d", err, req.MembershipId, member.OrgID, req.AccountId)
		return nil, err
	}
	if organization.AccountID != req.AccountId {
		return nil, errors.New("无权操作该组织")
	}

	updates := map[string]any{
		"status": req.Status,
	}
	if req.Status == model.MemberStatusActive && member.Status != model.MemberStatusActive {
		now := time.Now()
		updates["joined_at"] = &now
	}

	if err := s.repo.UpdateMembershipFields(s.repo.DB, member.ID, updates); err != nil {
		log.Error("更新成员状态失败: 更新成员关系异常: %v, membership_id=%d status=%d", err, member.ID, req.Status)
		return nil, err
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
			log.Error("查询成员统计失败: 获取当前用户失败: %v", err)
			return nil, err
		}
		organizations, err := s.repo.FindOrganizationByAccountID(s.repo.DB, userID)
		if err != nil {
			log.Error("查询成员统计失败: 查询组织异常: %v, user_id=%d", err, userID)
			return nil, err
		}
		if len(organizations) == 0 {
			return nil, errors.New("组织ID不能为空")
		}
		orgID = organizations[0].ID
	} else {
		userID, err := middleware.GetUserIDInt(s.c)
		if err != nil {
			log.Error("查询成员统计失败: 获取当前用户失败: %v, organization_id=%d", err, orgID)
			return nil, err
		}
		organizations, err := s.repo.FindOrganizationByAccountID(s.repo.DB, userID)
		if err != nil {
			log.Error("查询成员统计失败: 查询组织异常: %v, organization_id=%d user_id=%d", err, orgID, userID)
			return nil, err
		}
		if !hasOrganizationPermission(organizations, orgID) {
			return nil, errors.New("无权操作该组织")
		}
	}

	statusCounts, total, err := s.repo.GetMembershipStatusCounts(s.repo.DB, orgID)
	if err != nil {
		log.Error("查询成员统计失败: 查询统计数据异常: %v, organization_id=%d", err, orgID)
		return nil, err
	}

	resp := &api.MembershipStatsResponse{
		PendingCount:   statusCounts[model.MemberStatusPending],
		ActiveCount:    statusCounts[model.MemberStatusActive],
		InactiveCount:  statusCounts[model.MemberStatusLeft],
		SuspendedCount: statusCounts[model.MemberStatusRejected],
		TotalCount:     total,
	}

	return resp, nil
}
