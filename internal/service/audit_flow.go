package service

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
	"volunteer-system/pkg/util"

	"gorm.io/gorm"
)

type ApprovalHandler func(*gorm.DB, *model.AuditRecord) error

const (
	auditBatchActionApprove  int32 = 1
	auditBatchActionReject   int32 = 2
	maxAuditBatchDecisionIDs       = 500
)

var errAuditRecordNotFound = errors.New("audit record not found")

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

	auditorID, err := s.getAuditOperatorID()
	if err != nil {
		log.Warn("审核通过失败: 获取审核人失败, record_id=%d err=%v", req.Id, err)
		return nil, err
	}

	auditHandlerMap := map[int32]ApprovalHandler{
		model.AuditTargetVolunteer: s.applyVolunteerAuditApproval,
		model.AuditTargetMember:    s.applyMemberAuditApproval,
		model.AuditTargetSignup:    s.applySignupAuditApproval,
	}
	reason := strings.TrimSpace(req.Reason)
	var approvedRecord *model.AuditRecord

	err = s.repo.DB.Transaction(func(tx *gorm.DB) error {
		record, lockErr := s.repo.GetAuditRecordByIDForUpdate(tx, req.Id)
		if lockErr != nil {
			if errors.Is(lockErr, gorm.ErrRecordNotFound) {
				return errAuditRecordNotFound
			}
			return lockErr
		}
		if pendingErr := ensureAuditRecordPending(record); pendingErr != nil {
			return pendingErr
		}
		if permissionErr := s.requireAuditReviewPermission(auditorID, record); permissionErr != nil {
			return permissionErr
		}

		handler, ok := auditHandlerMap[record.TargetType]
		if !ok {
			return errors.New("不支持的审核目标类型")
		}

		if err := handler(tx, record); err != nil {
			return err
		}

		updates := map[string]any{
			"auditor_id":    auditorID,
			"audit_result":  model.AuditResultByStatus[model.AuditStatusApproved],
			"reject_reason": reason,
			"audit_time":    time.Now(),
			"status":        model.AuditStatusApproved,
		}
		if record.TargetID > 0 {
			updates["target_id"] = record.TargetID
		}
		if err := s.repo.UpdateAuditRecordByID(tx, record.ID, updates); err != nil {
			return err
		}
		approvedRecord = record
		return nil
	})
	if err != nil {
		if errors.Is(err, errAuditRecordNotFound) {
			log.Warn("审核通过失败: 审核记录不存在, record_id=%d", req.Id)
			return nil, errors.New("审核记录不存在")
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("审核通过失败: 审核目标不存在, record_id=%d", req.Id)
			return nil, errors.New("审核目标不存在")
		}
		log.Warn("审核通过失败: %v, record_id=%d", err, req.Id)
		return nil, err
	}
	log.Info("审核通过成功: record_id=%d target_type=%d target_id=%d auditor_id=%d", approvedRecord.ID, approvedRecord.TargetType, approvedRecord.TargetID, auditorID)
	s.handleAuditApprovedSideEffects(approvedRecord, auditorID)

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
	auditorID, err := s.getAuditOperatorID()
	if err != nil {
		log.Warn("审核驳回失败: 获取审核人失败, record_id=%d err=%v", req.Id, err)
		return nil, err
	}
	var rejectedRecord *model.AuditRecord

	err = s.repo.DB.Transaction(func(tx *gorm.DB) error {
		record, lockErr := s.repo.GetAuditRecordByIDForUpdate(tx, req.Id)
		if lockErr != nil {
			if errors.Is(lockErr, gorm.ErrRecordNotFound) {
				return errAuditRecordNotFound
			}
			return lockErr
		}
		if pendingErr := ensureAuditRecordPending(record); pendingErr != nil {
			return pendingErr
		}
		if permissionErr := s.requireAuditReviewPermission(auditorID, record); permissionErr != nil {
			return permissionErr
		}
		if err := s.applyAuditRejectionSideEffects(tx, record); err != nil {
			return err
		}

		updates := map[string]any{
			"auditor_id":    auditorID,
			"audit_result":  model.AuditResultByStatus[model.AuditStatusRejected],
			"reject_reason": reason,
			"audit_time":    time.Now(),
			"status":        model.AuditStatusRejected,
		}
		if record.TargetID > 0 {
			updates["target_id"] = record.TargetID
		}
		if err := s.repo.UpdateAuditRecordByID(tx, record.ID, updates); err != nil {
			return err
		}
		rejectedRecord = record
		return nil
	})
	if err != nil {
		if errors.Is(err, errAuditRecordNotFound) {
			log.Warn("审核驳回失败: 审核记录不存在, record_id=%d", req.Id)
			return nil, errors.New("审核记录不存在")
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("审核驳回失败: 审核目标不存在, record_id=%d", req.Id)
			return nil, errors.New("审核目标不存在")
		}
		log.Warn("审核驳回失败: %v, record_id=%d", err, req.Id)
		return nil, err
	}
	log.Info("审核驳回成功: record_id=%d target_type=%d target_id=%d auditor_id=%d", rejectedRecord.ID, rejectedRecord.TargetType, rejectedRecord.TargetID, auditorID)
	s.handleAuditRejectedSideEffects(rejectedRecord, auditorID, reason)

	return &resp, nil
}

// AuditBatchDecision executes approval/rejection decisions in batch and returns partial success result.
func (s *AuditService) AuditBatchDecision(req *api.AuditBatchDecisionRequest) (*api.AuditBatchDecisionResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	ids := util.UniquePositiveInt64(req.Ids)
	if len(ids) == 0 {
		return nil, errors.New("审核记录ID不能为空")
	}
	if len(ids) > maxAuditBatchDecisionIDs {
		return nil, fmt.Errorf("单次最多处理 %d 条审核记录", maxAuditBatchDecisionIDs)
	}
	if req.Action != auditBatchActionApprove && req.Action != auditBatchActionReject {
		return nil, errors.New("批量审核动作不合法")
	}
	if req.Action == auditBatchActionReject && strings.TrimSpace(req.Reason) == "" {
		return nil, errors.New("驳回原因不能为空")
	}

	successCount := int32(0)
	failedIDs := make([]int64, 0)
	for _, id := range ids {
		var err error
		switch req.Action {
		case auditBatchActionApprove:
			_, err = s.AuditApproval(&api.AuditApprovalRequest{
				Id:     id,
				Reason: req.Reason,
			})
		case auditBatchActionReject:
			_, err = s.AuditRejection(&api.AuditRejectionRequest{
				Id:     id,
				Reason: req.Reason,
			})
		}
		if err != nil {
			failedIDs = append(failedIDs, id)
			continue
		}
		successCount++
	}

	return &api.AuditBatchDecisionResponse{
		SuccessCount: successCount,
		FailedIds:    failedIDs,
	}, nil
}

// ensureAuditRecordPending 校验审核记录是否处于待处理状态。
func ensureAuditRecordPending(record *model.AuditRecord) error {
	if record == nil {
		return errors.New("审核记录不存在")
	}
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
