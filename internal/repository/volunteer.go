package repository

import (
	"volunteer-system/internal/model"

	"gorm.io/gorm"
)

// FindVolunteerByAccountID 根据账户ID查找志愿者
func (r *Repository) FindVolunteerByAccountID(db *gorm.DB, accountID int64) (*model.Volunteer, error) {
	var volunteer model.Volunteer
	err := db.WithContext(r.ctx).Model(&volunteer).Where("account_id = ?", accountID).First(&volunteer).Error
	if err != nil {
		return nil, err
	}
	return &volunteer, nil
}

// FindVolunteerByID 根据ID查找志愿者
func (r *Repository) FindVolunteerByID(db *gorm.DB, id int64) (*model.Volunteer, error) {
	var volunteer model.Volunteer
	err := db.WithContext(r.ctx).Model(&volunteer).Where("id = ?", id).First(&volunteer).Error
	if err != nil {
		return nil, err
	}
	return &volunteer, nil
}

// GetVolunteersByIDs returns volunteers by ids.
func (r *Repository) GetVolunteersByIDs(db *gorm.DB, ids []int64) ([]*model.Volunteer, error) {
	volunteers := make([]*model.Volunteer, 0)
	if len(ids) == 0 {
		return volunteers, nil
	}

	if err := db.WithContext(r.ctx).Model(&model.Volunteer{}).Where("id IN ?", ids).Find(&volunteers).Error; err != nil {
		return nil, err
	}

	return volunteers, nil
}

// CreateVolunteer 创建志愿者档案
func (r *Repository) CreateVolunteer(db *gorm.DB, volunteer *model.Volunteer) error {
	err := db.WithContext(r.ctx).Create(volunteer).Error
	if err != nil {
		return err
	}
	return nil
}

// UpdateVolunteer 更新志愿者档案
func (r *Repository) UpdateVolunteer(db *gorm.DB, id int64, updates map[string]interface{}) error {
	err := db.WithContext(r.ctx).Model(&model.Volunteer{}).Where("id = ?", id).Updates(updates).Error
	if err != nil {
		return err
	}
	return nil
}

// GetVolunteerList 获取志愿者列表（组织管理员端，仅所管理组织的正式成员）
func (r *Repository) GetVolunteerList(db *gorm.DB, orgIDs []int64, queryMap map[string]any, limit, offset int) ([]*model.Volunteer, int64, error) {
	var volunteers []*model.Volunteer
	var total int64

	if len(orgIDs) == 0 {
		return volunteers, 0, nil
	}

	// 创建 base session，通过组织成员关系限制组织范围
	baseSession := db.WithContext(r.ctx).
		Table("volunteers as v").
		Joins("INNER JOIN org_members as m ON m.volunteer_id = v.id").
		Where("m.org_id IN ?", orgIDs).
		Where("m.status = ?", model.MemberStatusActive)

	// 循环处理所有查询条件
	for key, value := range queryMap {
		// 状态筛选请使用 volunteers 别名，例如 "v.status = ?"
		baseSession = baseSession.Where(key, value)
	}

	// 使用去重后的志愿者ID获取总数，避免同一志愿者在多个组织时重复计数
	err := baseSession.Distinct("v.id").Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		// 如果没有数据，直接返回空结果
		return volunteers, 0, nil
	}
	// 使用去重后的志愿者数据分页查询，避免同一志愿者重复出现
	querySession := baseSession.Distinct().Select("v.*")
	err = querySession.Offset(offset).Limit(limit).
		Order("v.created_at DESC").
		Find(&volunteers).Error
	if err != nil {
		return nil, 0, err
	}

	return volunteers, total, nil
}

// FindVolunteerIDsByKeyword 根据关键字查询志愿者ID列表
func (r *Repository) FindVolunteerIDsByKeyword(db *gorm.DB, keyword string) ([]int64, error) {
	var volunteerIDs []int64
	if err := db.WithContext(r.ctx).Model(&model.Volunteer{}).
		Where("real_name LIKE ?", "%"+keyword+"%").
		Pluck("id", &volunteerIDs).Error; err != nil {
		return nil, err
	}
	return volunteerIDs, nil
}
