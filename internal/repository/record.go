package repository

import (
	"time"
	"volunteer-system/internal/model"

	"gorm.io/gorm"
)

// CreateRecord 创建志愿者积分/工时变动流水。
func (r *Repository) CreateRecord(db *gorm.DB, record *model.Record) error {
	return db.WithContext(r.ctx).Create(record).Error
}

// SumRecordAmountByTypeAndTime 按类型和时间窗口聚合志愿者流水值。
func (r *Repository) SumRecordAmountByTypeAndTime(db *gorm.DB, volunteerID int64, recordType string, start, end time.Time) (float64, error) {
	type sumResult struct {
		Amount float64 `gorm:"column:amount"`
	}

	var result sumResult
	err := db.WithContext(r.ctx).
		Model(&model.Record{}).
		Select("COALESCE(SUM(amount), 0) AS amount").
		Where("volunteer_id = ?", volunteerID).
		Where("type = ?", recordType).
		Where("create_time >= ? AND create_time < ?", start, end).
		Scan(&result).Error
	if err != nil {
		return 0, err
	}
	return result.Amount, nil
}
