package repository

import (
	"volunteer-system/internal/model"

	"gorm.io/gorm"
)

// CreateWorkHourLog 创建工时流水
func (r *Repository) CreateWorkHourLog(db *gorm.DB, logItem *model.WorkHourLog) error {
	return db.WithContext(r.ctx).Create(logItem).Error
}

// GetWorkHourLogByID 根据ID获取工时流水
func (r *Repository) GetWorkHourLogByID(db *gorm.DB, id int64) (*model.WorkHourLog, error) {
	var logItem model.WorkHourLog
	if err := db.WithContext(r.ctx).Where("id = ?", id).First(&logItem).Error; err != nil {
		return nil, err
	}
	return &logItem, nil
}

// GetWorkHourLogByIdempotencyKey 根据幂等键获取工时流水
func (r *Repository) GetWorkHourLogByIdempotencyKey(db *gorm.DB, key string) (*model.WorkHourLog, error) {
	var logItem model.WorkHourLog
	err := db.WithContext(r.ctx).Where("idempotency_key = ?", key).First(&logItem).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &logItem, nil
}

// ListWorkHourLogs 查询工时流水
func (r *Repository) ListWorkHourLogs(db *gorm.DB, queryMap map[string]any, limit, offset int) ([]*model.WorkHourLog, int64, error) {
	var logs []*model.WorkHourLog
	var total int64

	query := db.WithContext(r.ctx).Model(&model.WorkHourLog{})
	for key, value := range queryMap {
		query = query.Where(key, value)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return logs, 0, nil
	}

	if err := query.Offset(offset).
		Limit(limit).
		Order("created_at DESC, id DESC").
		Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
