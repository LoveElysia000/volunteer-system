package service

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
	"volunteer-system/internal/model"

	"gorm.io/gorm"
)

// applyMemberAuditApproval 处理组织成员目标的审核通过逻辑。
func (s *AuditService) applyMemberAuditApproval(tx *gorm.DB, record *model.AuditRecord) error {
	memberSnapshot, err := parseMemberAuditSnapshot(record.NewContent)
	if err != nil {
		return err
	}

	switch record.OperationType {
	case model.OperationTypeCreate:
		return s.applyMemberCreateAuditApproval(tx, record, memberSnapshot)
	case model.OperationTypeUpdate:
		return s.applyMemberUpdateAuditApproval(tx, record, memberSnapshot)
	case model.OperationTypeDelete:
		return s.applyMemberDeleteAuditApproval(tx, record, memberSnapshot)
	default:
		return nil
	}
}

func parseMemberAuditSnapshot(raw string) (*model.OrgMember, error) {
	snapshot := &model.OrgMember{}
	if strings.TrimSpace(raw) == "" {
		return snapshot, nil
	}
	if err := json.Unmarshal([]byte(raw), snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func resolveMemberAuditTargetID(record *model.AuditRecord, snapshot *model.OrgMember) (int64, error) {
	targetID := record.TargetID
	if targetID <= 0 && snapshot != nil {
		targetID = snapshot.ID
	}
	if targetID <= 0 {
		return 0, errors.New("目标ID不能为空")
	}
	if record.TargetID > 0 && snapshot != nil && snapshot.ID > 0 && snapshot.ID != record.TargetID {
		return 0, errors.New("目标ID不一致")
	}
	return targetID, nil
}

func ensureMemberSnapshotIdentity(current, snapshot *model.OrgMember) error {
	if current == nil || snapshot == nil {
		return nil
	}
	if snapshot.OrgID > 0 && snapshot.OrgID != current.OrgID {
		return errors.New("成员关系组织不一致")
	}
	if snapshot.VolunteerID > 0 && snapshot.VolunteerID != current.VolunteerID {
		return errors.New("成员关系志愿者不一致")
	}
	return nil
}

func buildMemberUpdateAuditApprovalUpdates(snapshot *model.OrgMember, targetStatus int32, now time.Time) map[string]any {
	leaveReason := ""
	if snapshot != nil {
		leaveReason = strings.TrimSpace(snapshot.LeaveReason)
	}
	updates := buildMemberStatusUpdates(targetStatus, leaveReason, now)
	if snapshot == nil {
		return updates
	}
	if snapshot.Role > 0 {
		updates["role"] = snapshot.Role
	}
	if !snapshot.AppliedAt.IsZero() {
		updates["applied_at"] = snapshot.AppliedAt
	}
	if snapshot.JoinedAt != nil {
		updates["joined_at"] = snapshot.JoinedAt
	}
	if snapshot.LeftAt != nil {
		updates["left_at"] = snapshot.LeftAt
	}
	if leaveReason != "" {
		updates["leave_reason"] = leaveReason
	}
	if targetStatus == model.MemberStatusActive && snapshot.JoinedAt == nil {
		updates["joined_at"] = &now
	}
	return updates
}

func buildMemberDeleteAuditApprovalUpdates(snapshot *model.OrgMember, now time.Time) map[string]any {
	leaveReason := ""
	if snapshot != nil {
		leaveReason = strings.TrimSpace(snapshot.LeaveReason)
	}
	updates := buildMemberStatusUpdates(model.MemberStatusLeft, leaveReason, now)
	if snapshot != nil && snapshot.LeftAt != nil {
		updates["left_at"] = snapshot.LeftAt
	}
	return updates
}

func (s *AuditService) applyMemberCreateAuditApproval(
	tx *gorm.DB,
	record *model.AuditRecord,
	memberSnapshot *model.OrgMember,
) error {
	now := time.Now()
	if memberSnapshot.OrgID <= 0 || memberSnapshot.VolunteerID <= 0 {
		return errors.New("成员关系快照无效")
	}

	memberSnapshot.ID = 0
	memberSnapshot.Status = model.MemberStatusActive
	if memberSnapshot.AppliedAt.IsZero() {
		memberSnapshot.AppliedAt = now
	}
	if memberSnapshot.JoinedAt == nil {
		memberSnapshot.JoinedAt = &now
	}
	if err := s.repo.CreateMembership(tx, memberSnapshot); err != nil {
		return err
	}
	record.TargetID = memberSnapshot.ID
	return nil
}

func (s *AuditService) applyMemberUpdateAuditApproval(
	tx *gorm.DB,
	record *model.AuditRecord,
	memberSnapshot *model.OrgMember,
) error {
	memberID, err := resolveMemberAuditTargetID(record, memberSnapshot)
	if err != nil {
		return err
	}
	current, err := s.repo.GetMembershipByIDForUpdate(tx, memberID)
	if err != nil {
		return err
	}
	if err := ensureMemberSnapshotIdentity(current, memberSnapshot); err != nil {
		return err
	}

	targetStatus := model.MemberStatusActive
	if memberSnapshot.Status > 0 {
		targetStatus = memberSnapshot.Status
	}
	if err := validateMemberTransition(flowAdminUpdate, current.Status, targetStatus); err != nil {
		return err
	}

	now := time.Now()
	updates := buildMemberUpdateAuditApprovalUpdates(memberSnapshot, targetStatus, now)
	if err := s.repo.UpdateMembershipFields(tx, memberID, updates); err != nil {
		return err
	}
	record.TargetID = memberID
	return nil
}

func (s *AuditService) applyMemberDeleteAuditApproval(
	tx *gorm.DB,
	record *model.AuditRecord,
	memberSnapshot *model.OrgMember,
) error {
	memberID, err := resolveMemberAuditTargetID(record, memberSnapshot)
	if err != nil {
		return err
	}
	current, err := s.repo.GetMembershipByIDForUpdate(tx, memberID)
	if err != nil {
		return err
	}
	if err := ensureMemberSnapshotIdentity(current, memberSnapshot); err != nil {
		return err
	}
	if err := validateMemberTransition(flowLeaveApply, current.Status, model.MemberStatusLeft); err != nil {
		return err
	}

	now := time.Now()
	updates := buildMemberDeleteAuditApprovalUpdates(memberSnapshot, now)
	if err := s.repo.UpdateMembershipFields(tx, memberID, updates); err != nil {
		return err
	}
	record.TargetID = memberID
	return nil
}
