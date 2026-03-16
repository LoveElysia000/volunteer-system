package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type MembershipService struct {
	Service
}

const membershipApplicationSubmittedMessage = "application submitted"

// NewMembershipService 创建成员关系服务实例，并注入请求上下文与仓储依赖。
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

// ===== 志愿者端：申请加入/退出组织 =====

func buildPendingJoinResponse() *api.VolunteerJoinResponse {
	return &api.VolunteerJoinResponse{
		Status:  model.MemberStatusPending,
		Message: membershipApplicationSubmittedMessage,
	}
}

func (s *MembershipService) ensureJoinTargets(tx *gorm.DB, orgID, volunteerID int64) error {
	if err := s.ensureJoinOrganizationExists(tx, orgID); err != nil {
		return err
	}
	return s.ensureJoinVolunteerExists(tx, volunteerID)
}

func (s *MembershipService) ensureJoinOrganizationExists(tx *gorm.DB, orgID int64) error {
	organization, err := s.repo.GetOrganizationByID(tx, orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("组织不存在")
		}
		return err
	}
	if organization == nil {
		return errors.New("组织不存在")
	}
	return nil
}

func (s *MembershipService) ensureJoinVolunteerExists(tx *gorm.DB, volunteerID int64) error {
	// 锁定志愿者，串行化同一志愿者的加入申请，防止并发重复提交。
	volunteer, err := s.repo.FindVolunteerByIDForUpdate(tx, volunteerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("志愿者不存在")
		}
		return err
	}
	if volunteer == nil {
		return errors.New("志愿者不存在")
	}
	return nil
}

func (s *MembershipService) handleExistingJoinMembership(tx *gorm.DB, existing *model.OrgMember, userID int64) (*api.VolunteerJoinResponse, error) {
	if existing == nil {
		return nil, nil
	}

	if err := validateMemberTransition(flowJoinReapply, existing.Status, model.MemberStatusActive); err != nil {
		return nil, err
	}
	return s.submitReapplyMembershipAudit(tx, existing, userID)
}

func (s *MembershipService) submitReapplyMembershipAudit(tx *gorm.DB, existing *model.OrgMember, userID int64) (*api.VolunteerJoinResponse, error) {
	hasPendingMemberAudit, err := s.hasPendingMemberAuditByTargetID(tx, existing.ID)
	if err != nil {
		return nil, err
	}
	if hasPendingMemberAudit {
		return nil, errors.New("成员关系已存在或正在审核中")
	}

	now := time.Now()
	oldMember := *existing
	newMember := oldMember
	newMember.Status = model.MemberStatusActive
	newMember.AppliedAt = now
	newMember.JoinedAt = nil
	newMember.LeftAt = nil
	newMember.LeaveReason = ""

	record, err := buildPendingAuditRecord(
		model.AuditTargetMember,
		model.OperationTypeUpdate,
		existing.ID,
		userID,
		&oldMember,
		&newMember,
		now,
	)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateAuditRecord(tx, record); err != nil {
		return nil, err
	}

	return buildPendingJoinResponse(), nil
}

func (s *MembershipService) submitCreateMembershipAudit(tx *gorm.DB, orgID, volunteerID, userID int64) (*api.VolunteerJoinResponse, error) {
	hasPendingCreateAudit, err := s.hasPendingMemberCreateAudit(tx, orgID, volunteerID, userID)
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
	record, err := buildPendingCreateAuditRecordByModel(
		model.AuditTargetMember,
		userID,
		newMember,
		now,
	)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateAuditRecord(tx, record); err != nil {
		return nil, err
	}
	return buildPendingJoinResponse(), nil
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

	orgID := req.OrganizationId
	volunteerID := req.VolunteerId
	var resp *api.VolunteerJoinResponse

	err = s.withTransaction(func(tx *gorm.DB) error {
		if err := s.ensureJoinTargets(tx, orgID, volunteerID); err != nil {
			return err
		}

		existing, existingErr := s.repo.FindMembershipByOrgAndVolunteer(tx, orgID, volunteerID)
		if existingErr != nil {
			return existingErr
		}
		existingResp, err := s.handleExistingJoinMembership(tx, existing, userID)
		if err != nil {
			return err
		}
		if existingResp != nil {
			resp = existingResp
			return nil
		}

		createResp, err := s.submitCreateMembershipAudit(tx, orgID, volunteerID, userID)
		if err != nil {
			return err
		}
		resp = createResp
		return nil
	})
	if err != nil {
		log.Error("提交加入组织申请失败: %v, organization_id=%d volunteer_id=%d user_id=%d", err, orgID, volunteerID, userID)
		return nil, err
	}
	return resp, nil
}

// hasPendingMemberCreateAudit 检查是否存在相同组织与志愿者的待审核新增成员申请。
func (s *MembershipService) hasPendingMemberCreateAudit(db *gorm.DB, orgID, volunteerID, creatorID int64) (bool, error) {
	exists, err := s.repo.ExistsPendingMemberCreateAuditBySnapshot(db, orgID, volunteerID, creatorID)
	if err != nil {
		log.Error("查询待审核创建成员记录失败: %v, organization_id=%d volunteer_id=%d", err, orgID, volunteerID)
		return false, err
	}
	return exists, nil
}

func (s *MembershipService) hasPendingMemberAuditByTargetID(db *gorm.DB, targetID int64) (bool, error) {
	return s.repo.ExistsPendingMemberAuditByTargetID(db, targetID)
}

func (s *MembershipService) submitLeaveMembershipAudit(
	tx *gorm.DB,
	membershipID, currentVolunteerID, userID int64,
	leaveReason string,
) (*api.VolunteerLeaveResponse, error) {
	member, err := s.repo.GetMembershipByIDForUpdate(tx, membershipID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("成员关系不存在")
		}
		return nil, err
	}
	if member.VolunteerID != currentVolunteerID {
		return nil, errors.New("无权操作该成员关系")
	}
	if err := validateMemberTransition(flowLeaveApply, member.Status, model.MemberStatusLeft); err != nil {
		return nil, err
	}

	hasPending, err := s.hasPendingMemberAuditByTargetID(tx, member.ID)
	if err != nil {
		return nil, err
	}
	if hasPending {
		return nil, errors.New("该成员关系已有待审核申请")
	}

	newMember := *member
	newMember.Status = model.MemberStatusLeft
	now := time.Now()
	newMember.LeftAt = &now
	newMember.LeaveReason = leaveReason

	record, err := buildPendingDeleteAuditRecordByModel(
		model.AuditTargetMember,
		member.ID,
		userID,
		member,
		&newMember,
		now,
	)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateAuditRecord(tx, record); err != nil {
		return nil, err
	}

	return &api.VolunteerLeaveResponse{
		Message: membershipApplicationSubmittedMessage,
	}, nil
}

// VolunteerLeaveOrganization submits a leave request for an organization.
func (s *MembershipService) VolunteerLeaveOrganization(req *api.VolunteerLeaveRequest) (*api.VolunteerLeaveResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}

	if req.MembershipId <= 0 {
		return nil, errors.New("成员关系ID不能为空")
	}
	leaveReason := strings.TrimSpace(req.Reason)

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

	var resp *api.VolunteerLeaveResponse
	err = s.withTransaction(func(tx *gorm.DB) error {
		txResp, txErr := s.submitLeaveMembershipAudit(tx, req.MembershipId, currentVolunteer.ID, userID, leaveReason)
		if txErr != nil {
			return txErr
		}
		resp = txResp
		return nil
	})
	if err != nil {
		log.Error("提交退出组织申请失败: %v, membership_id=%d user_id=%d", err, req.MembershipId, userID)
		return nil, err
	}
	return resp, nil
}

// GetVolunteerOrganizations returns organizations joined by a volunteer.
func (s *MembershipService) GetVolunteerOrganizations(req *api.VolunteerOrganizationsRequest) (*api.VolunteerOrganizationsResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
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
		for _, member := range list {
			if member == nil {
				continue
			}
			orgIDs = append(orgIDs, member.OrgID)
		}
		orgIDs = util.UniquePositiveInt64(orgIDs)

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
		if m == nil {
			continue
		}
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

// ===== 组织端：成员管理 =====

// GetOrganizationMembers returns members for an organization.
func (s *MembershipService) GetOrganizationMembers(req *api.OrganizationMembersRequest) (*api.OrganizationMembersResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
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
	if err := s.requireOrgPermission(
		userID,
		req.OrganizationId,
		model.PermissionResourceMembership,
		model.PermissionActionManage,
	); err != nil {
		return nil, err
	}

	pageSize := int(req.PageSize)
	offset := (int(req.Page) - 1) * pageSize
	members, total, err := s.repo.GetOrganizationMembers(s.repo.DB, req.OrganizationId, req.Status, req.Role, req.Keyword, pageSize, offset)
	if err != nil {
		log.Error("查询组织成员列表失败: 查询成员数据异常: %v, organization_id=%d page=%d page_size=%d", err, req.OrganizationId, req.Page, req.PageSize)
		return nil, err
	}

	volunteerIDs := make([]int64, 0, len(members))
	for _, member := range members {
		if member == nil {
			continue
		}
		volunteerIDs = append(volunteerIDs, member.VolunteerID)
	}
	volunteerIDs = util.UniquePositiveInt64(volunteerIDs)

	var (
		organization *model.Organization
		volunteers   []*model.Volunteer
	)
	group, _ := errgroup.WithContext(s.ctx)
	group.SetLimit(2)
	group.Go(func() error {
		org, queryErr := s.repo.GetOrganizationByID(s.repo.DB, req.OrganizationId)
		if queryErr != nil {
			log.Error("查询组织成员列表失败: 查询组织信息异常: %v, organization_id=%d", queryErr, req.OrganizationId)
			return queryErr
		}
		organization = org
		return nil
	})
	group.Go(func() error {
		volunteerList, queryErr := s.repo.GetVolunteersByIDs(s.repo.DB, volunteerIDs)
		if queryErr != nil {
			log.Error("查询组织成员列表失败: 批量查询志愿者异常: %v, organization_id=%d volunteer_count=%d", queryErr, req.OrganizationId, len(volunteerIDs))
			return queryErr
		}
		volunteers = volunteerList
		return nil
	})
	if waitErr := group.Wait(); waitErr != nil {
		return nil, waitErr
	}

	volunteerNameMap := make(map[int64]string, len(volunteers))
	for _, volunteer := range volunteers {
		if volunteer == nil {
			continue
		}
		volunteerNameMap[volunteer.ID] = volunteer.RealName
	}
	organizationName := ""
	if organization != nil {
		organizationName = organization.OrgName
	}

	resp := &api.OrganizationMembersResponse{
		Total: int32(total),
		List:  make([]*api.MemberInfo, 0, len(members)),
	}

	for _, m := range members {
		if m == nil {
			continue
		}
		leaveDate := ""
		if m.LeftAt != nil {
			leaveDate = m.LeftAt.Format("2006-01-02 15:04:05")
		}
		item := &api.MemberInfo{
			MembershipId:     m.ID,
			VolunteerId:      m.VolunteerID,
			VolunteerName:    volunteerNameMap[m.VolunteerID],
			VolunteerCode:    "",
			OrganizationId:   m.OrgID,
			OrganizationName: organizationName,
			Status:           m.Status,
			Role:             m.Role,
			Position:         "",
			Motivation:       "",
			ExpectedHours:    "",
			JoinDate:         util.FormatJoinDate(m.JoinedAt, m.AppliedAt),
			ReviewDate:       "",
			ReviewComment:    "",
			LeaveDate:        leaveDate,
			LeaveReason:      m.LeaveReason,
			CreatedAt:        m.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:        m.UpdatedAt.Format("2006-01-02 15:04:05"),
		}
		resp.List = append(resp.List, item)
	}

	return resp, nil
}

// ----- 组织端私有能力 -----
func (s *MembershipService) ensureMembershipManagePermission(operatorID, orgID int64) error {
	organization, err := s.repo.GetOrganizationByID(s.repo.DB, orgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Error("更新成员状态失败: 组织不存在, org_id=%d", orgID)
			return errors.New("组织不存在")
		}
		log.Error("更新成员状态失败: 查询组织异常: %v, org_id=%d operator_id=%d", err, orgID, operatorID)
		return err
	}
	return s.requireOrgPermission(
		operatorID,
		organization.ID,
		model.PermissionResourceMembership,
		model.PermissionActionManage,
	)
}

func buildMemberStatusUpdates(newStatus int32, reviewComment string, now time.Time) map[string]any {
	updates := map[string]any{
		"status": newStatus,
	}
	switch newStatus {
	case model.MemberStatusActive:
		updates["joined_at"] = &now
		updates["left_at"] = nil
		updates["leave_reason"] = ""
	case model.MemberStatusLeft:
		updates["left_at"] = &now
		updates["leave_reason"] = strings.TrimSpace(reviewComment)
	}
	return updates
}

func (s *MembershipService) updateMemberStatusInTx(
	tx *gorm.DB,
	membershipID int64,
	newStatus int32,
	reviewComment string,
	expectedOrgID int64,
) (*model.OrgMember, bool, error) {
	member, err := s.repo.GetMembershipByIDForUpdate(tx, membershipID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, errors.New("成员关系不存在")
		}
		return nil, false, err
	}
	if member.OrgID != expectedOrgID {
		return nil, false, errors.New("成员关系组织已变更，请刷新后重试")
	}
	if member.Status == newStatus {
		return nil, false, nil
	}
	if err := validateMemberTransition(flowAdminUpdate, member.Status, newStatus); err != nil {
		return nil, false, err
	}

	hasPending, err := s.hasPendingMemberAuditByTargetID(tx, member.ID)
	if err != nil {
		return nil, false, err
	}
	if hasPending {
		return nil, false, errors.New("该成员关系已有待审核申请")
	}

	previous := *member
	now := time.Now()
	updates := buildMemberStatusUpdates(newStatus, reviewComment, now)
	if err := s.repo.UpdateMembershipFields(tx, member.ID, updates); err != nil {
		return nil, false, err
	}
	return &previous, true, nil
}

// UpdateMemberStatus updates membership status by authorized operator.
func (s *MembershipService) UpdateMemberStatus(req *api.MemberStatusUpdateRequest) (*api.MemberStatusUpdateResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	operatorID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("更新成员状态失败: 获取当前用户失败: %v", err)
		return nil, err
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
	if err := s.ensureMembershipManagePermission(operatorID, member.OrgID); err != nil {
		return nil, err
	}

	var sideEffectMember *model.OrgMember
	err = s.withTransaction(func(tx *gorm.DB) error {
		previous, changed, txErr := s.updateMemberStatusInTx(
			tx,
			req.MembershipId,
			req.Status,
			req.ReviewComment,
			member.OrgID,
		)
		if txErr != nil {
			return txErr
		}
		if changed {
			sideEffectMember = previous
		}
		return nil
	})
	if err != nil {
		log.Error("更新成员状态失败: %v, membership_id=%d operator_id=%d target_status=%d",
			err, req.MembershipId, operatorID, req.Status)
		return nil, err
	}
	if sideEffectMember != nil {
		s.handleMemberStatusSideEffects(sideEffectMember, req.Status, operatorID)
	}

	return &api.MemberStatusUpdateResponse{
		Message: "status updated",
	}, nil
}

func (s *MembershipService) handleMemberStatusSideEffects(member *model.OrgMember, newStatus int32, operatorID int64) {
	if member == nil || member.Status == newStatus {
		return
	}

	volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, member.VolunteerID)
	if err != nil || volunteer == nil {
		log.Error("成员状态副作用失败: 查询志愿者异常: %v, membership_id=%d volunteer_id=%d", err, member.ID, member.VolunteerID)
		return
	}
	if volunteer.AccountID <= 0 {
		log.Warn("成员状态副作用跳过: 志愿者账号ID无效, membership_id=%d volunteer_id=%d", member.ID, member.VolunteerID)
		return
	}

	switch newStatus {
	case model.MemberStatusActive:
		// 成员转为 active 时发“加入组织通过”通知给成员本人。
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
			ActorID:     operatorID,
			CreatedAt:   time.Now(),
			Payload: map[string]any{
				"receiverAccountID": volunteer.AccountID,
				"organizationName":  orgName,
				"volunteerName":     volunteer.RealName,
			},
			DedupeKey: fmt.Sprintf("member.join.approved:%d", member.ID),
		})

	case model.MemberStatusLeft:
		// 成员退出组织后，将该组织来源通知归档而非物理删除。
		rows, archiveErr := s.repo.ArchiveNotificationInboxByReceiverAndOrg(
			s.repo.DB,
			volunteer.AccountID,
			member.OrgID,
			model.NotificationArchiveReasonLeftOrg,
			time.Now(),
		)
		if archiveErr != nil {
			log.Error("成员状态副作用失败: 归档通知异常: %v, membership_id=%d receiver_id=%d org_id=%d",
				archiveErr, member.ID, volunteer.AccountID, member.OrgID)
			return
		}
		log.Info("成员状态副作用完成: 归档组织通知成功, membership_id=%d receiver_id=%d org_id=%d archived_rows=%d",
			member.ID, volunteer.AccountID, member.OrgID, rows)
	}
}

// MembershipStats returns summary counts.
func (s *MembershipService) MembershipStats(req *api.MembershipStatsRequest) (*api.MembershipStatsResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("查询成员统计失败: 获取当前用户失败: %v", err)
		return nil, err
	}

	orgID := req.OrganizationId
	if orgID <= 0 {
		// 在未显式传 organizationId 时，默认使用用户可管理的第一个组织作用域。
		orgIDs, err := s.repo.ListOrgScopeIDsByPermission(
			s.repo.DB,
			userID,
			model.PermissionResourceMembership,
			model.PermissionActionManage,
			1,
		)
		if err != nil {
			log.Error("查询成员统计失败: 查询RBAC组织作用域异常: %v, user_id=%d", err, userID)
			return nil, err
		}
		if len(orgIDs) == 0 {
			hasGlobal, perr := s.hasPermissionByScope(
				userID,
				model.RBACScopeGlobal,
				0,
				model.PermissionResourceMembership,
				model.PermissionActionManage,
			)
			if perr != nil {
				return nil, perr
			}
			if hasGlobal {
				return nil, errors.New("组织ID不能为空")
			}
			return nil, errors.New("无权操作该组织")
		}
		orgID = orgIDs[0]
	}

	if err := s.requireOrgPermission(
		userID,
		orgID,
		model.PermissionResourceMembership,
		model.PermissionActionManage,
	); err != nil {
		return nil, err
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
