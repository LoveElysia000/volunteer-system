package repository

import (
	"errors"
	"volunteer-system/internal/model"

	"gorm.io/gorm"
)

// FindNextLevelRuleByTotalHours 根据当前总工时查询下一等级规则。
func (r *Repository) FindNextLevelRuleByTotalHours(db *gorm.DB, totalHours float64) (*model.LevelRule, error) {
	var rule model.LevelRule
	err := db.WithContext(r.ctx).
		Model(&model.LevelRule{}).
		Where("threshold_hours > ?", totalHours).
		Order("threshold_hours ASC").
		First(&rule).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

// ResolveLevelIDByTotalHours 根据累计工时解析当前等级ID。
func (r *Repository) ResolveLevelIDByTotalHours(db *gorm.DB, totalHours float64) (int32, error) {
	type levelResult struct {
		Level int32 `gorm:"column:level"`
	}

	var result levelResult
	err := db.WithContext(r.ctx).
		Model(&model.LevelRule{}).
		Select("COALESCE(MAX(level), 1) AS level").
		Where("threshold_hours <= ?", totalHours).
		Scan(&result).Error
	if err != nil {
		return 0, err
	}
	if result.Level <= 0 {
		return 1, nil
	}
	return result.Level, nil
}
