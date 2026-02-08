package repository

import (
	"time"
	"volunteer-system/internal/model"

	"gorm.io/gorm"
)

type PendingVolunteerJoinOrgAuditTarget struct {
	TargetType int32     `gorm:"column:target_type"`
	TargetID   int64     `gorm:"column:target_id"`
	Status     int32     `gorm:"column:status"`
	Title      string    `gorm:"column:title"`
	SubTitle   string    `gorm:"column:sub_title"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

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

// UpdateAuditRecordByID updates one audit record by id.
func (r *Repository) UpdateAuditRecordByID(db *gorm.DB, id int64, updates map[string]any) error {
	return db.WithContext(r.ctx).
		Model(&model.AuditRecord{}).
		Where("id = ?", id).
		Updates(updates).Error
}
