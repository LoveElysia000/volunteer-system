package repository

import (
	"errors"
	"volunteer-system/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateAuditRecord inserts an audit record.
func (r *Repository) CreateAuditRecord(db *gorm.DB, record *model.AuditRecord) error {
	return db.WithContext(r.ctx).Create(record).Error
}

func (r *Repository) GetAuditRecordsList(db *gorm.DB, queryMap map[string]any, limit, offset int32) ([]*model.AuditRecord, int64, error) {
	var total int64
	var list []*model.AuditRecord

	query := db.WithContext(r.ctx).Model(&model.AuditRecord{})
	for key, value := range queryMap {
		query = query.Where(key, value)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return list, 0, nil
	}

	if limit > 0 {
		query = query.Limit(int(limit))
	}
	if offset > 0 {
		query = query.Offset(int(offset))
	}

	if err := query.Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ExistsPendingMemberAuditByTargetID checks if a pending member audit exists for target id.
func (r *Repository) ExistsPendingMemberAuditByTargetID(db *gorm.DB, targetID int64) (bool, error) {
	if targetID <= 0 {
		return false, nil
	}

	var id int64
	err := db.WithContext(r.ctx).
		Model(&model.AuditRecord{}).
		Select("id").
		Where("target_type = ? AND target_id = ? AND status = ?",
			model.AuditTargetMember, targetID, model.AuditStatusPending).
		Limit(1).
		Take(&id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ExistsPendingMemberCreateAuditBySnapshot checks pending member-create audit via snapshot keys.
func (r *Repository) ExistsPendingMemberCreateAuditBySnapshot(
	db *gorm.DB,
	orgID, volunteerID, creatorID int64,
) (bool, error) {
	if orgID <= 0 || volunteerID <= 0 || creatorID <= 0 {
		return false, nil
	}

	var id int64
	err := db.WithContext(r.ctx).
		Model(&model.AuditRecord{}).
		Select("id").
		Where("target_type = ? AND operation_type = ? AND status = ? AND creator_id = ? AND target_id = ?",
			model.AuditTargetMember, model.OperationTypeCreate, model.AuditStatusPending, creatorID, 0).
		Where(
			"COALESCE(JSON_EXTRACT(new_content, '$.org_id'), 0) = ? AND COALESCE(JSON_EXTRACT(new_content, '$.volunteer_id'), 0) = ?",
			orgID, volunteerID,
		).
		Limit(1).
		Take(&id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetAuditRecordByID finds one audit record by id.
func (r *Repository) GetAuditRecordByID(db *gorm.DB, id int64) (*model.AuditRecord, error) {
	var record model.AuditRecord
	if err := db.WithContext(r.ctx).
		Model(&model.AuditRecord{}).
		Where("id = ?", id).
		First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// GetAuditRecordByIDForUpdate finds one audit record by id and locks the row for update.
func (r *Repository) GetAuditRecordByIDForUpdate(db *gorm.DB, id int64) (*model.AuditRecord, error) {
	var record model.AuditRecord
	if err := db.WithContext(r.ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&model.AuditRecord{}).
		Where("id = ?", id).
		First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// UpdateAuditRecordByID updates one audit record by id.
func (r *Repository) UpdateAuditRecordByID(db *gorm.DB, id int64, updates map[string]any) error {
	return db.WithContext(r.ctx).
		Model(&model.AuditRecord{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// ListSignupRejectAuditRecords 批量查询报名审核驳回记录（按 audit_time/id 倒序）。
func (r *Repository) ListSignupRejectAuditRecords(db *gorm.DB, signupIDs []int64) ([]*model.AuditRecord, error) {
	if len(signupIDs) == 0 {
		return []*model.AuditRecord{}, nil
	}

	var records []*model.AuditRecord
	err := db.WithContext(r.ctx).
		Model(&model.AuditRecord{}).
		Where("target_type = ? AND target_id IN ? AND status = ? AND reject_reason <> ''",
			model.AuditTargetSignup, signupIDs, model.AuditStatusRejected).
		Order("audit_time DESC, id DESC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}
