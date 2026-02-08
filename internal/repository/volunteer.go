package repository

import (
	"time"
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

// UpdateVolunteerInfo 更新志愿者基本信息
func (r *Repository) UpdateVolunteerInfo(db *gorm.DB, id int64, realName string, gender int32, birthday *time.Time, avatarURL, introduction string) error {
	updates := make(map[string]interface{})
	if realName != "" {
		updates["real_name"] = realName
	}
	if gender >= 0 {
		updates["gender"] = gender
	}
	if birthday != nil {
		updates["birthday"] = birthday
	}
	if avatarURL != "" {
		updates["avatar_url"] = avatarURL
	}
	if introduction != "" {
		updates["introduction"] = introduction
	}
	if len(updates) == 0 {
		return nil
	}
	return r.UpdateVolunteer(db, id, updates)
}

// GetVolunteerList 获取志愿者列表（管理员端）
func (r *Repository) GetVolunteerList(db *gorm.DB, queryMap map[string]any, limit, offset int) ([]*model.Volunteer, int64, error) {
	var volunteers []*model.Volunteer
	var total int64

	// 创建 base session，关联 sys_accounts 表获取账号状态
	baseSession := db.WithContext(r.ctx).
		Table("volunteers as v").
		Joins("LEFT JOIN sys_accounts as sys ON v.account_id = sys.id")

	// 循环处理所有查询条件
	for key, value := range queryMap {
		baseSession = baseSession.Where(key, value)
	}

	// 使用 base session 获取总数
	err := baseSession.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		// 如果没有数据，直接返回空结果
		return volunteers, 0, nil
	}
	// 使用 base session 分页查询数据
	querySession := baseSession.Select("v.*")
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
