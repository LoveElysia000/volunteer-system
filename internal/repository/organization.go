package repository

import (
	"volunteer-system/internal/model"

	"gorm.io/gorm"
)

// GetOrganizationByID 根据组织ID查找组织
func (r *Repository) GetOrganizationByID(db *gorm.DB, orgID int64) (*model.Organization, error) {
	var organization model.Organization
	err := db.WithContext(r.ctx).Model(&model.Organization{}).Where("id = ?", orgID).First(&organization).Error
	if err != nil {
		return nil, err
	}
	return &organization, nil
}

// GetOrganizationsByIDs returns organizations by ids.
func (r *Repository) GetOrganizationsByIDs(db *gorm.DB, ids []int64) ([]*model.Organization, error) {
	organizations := make([]*model.Organization, 0)
	if len(ids) == 0 {
		return organizations, nil
	}

	if err := db.WithContext(r.ctx).Model(&model.Organization{}).Where("id IN ?", ids).Find(&organizations).Error; err != nil {
		return nil, err
	}

	return organizations, nil
}

// FindOrganizationByAccountID 根据账户ID查找组织
func (r *Repository) FindOrganizationByAccountID(db *gorm.DB, accountID int64) (*model.Organization, error) {
	var organization model.Organization
	err := db.WithContext(r.ctx).Model(&model.Organization{}).Where("account_id = ?", accountID).First(&organization).Error
	if err != nil {
		return nil, err
	}
	return &organization, nil
}

// CreateOrganization 创建组织档案
func (r *Repository) CreateOrganization(db *gorm.DB, org *model.Organization) error {
	err := db.WithContext(r.ctx).Create(org).Error
	if err != nil {
		return err
	}
	return nil
}

// UpdateOrganization 更新组织信息
func (r *Repository) UpdateOrganization(db *gorm.DB, orgID int64, updates map[string]any) error {
	err := db.WithContext(r.ctx).Model(&model.Organization{}).Where("id = ?", orgID).Updates(updates).Error
	if err != nil {
		return err
	}
	return nil
}

// DeleteOrganization 删除组织
func (r *Repository) DeleteOrganization(db *gorm.DB, orgID int64) error {
	err := db.WithContext(r.ctx).Delete(&model.Organization{}, orgID).Error
	if err != nil {
		return err
	}
	return nil
}

// BulkDeleteOrganizations 批量删除组织
func (r *Repository) BulkDeleteOrganizations(db *gorm.DB, orgIDs []int64) (int64, int64, error) {
	result := db.WithContext(r.ctx).Delete(&model.Organization{}, orgIDs)
	if result.Error != nil {
		return 0, 0, result.Error
	}
	return result.RowsAffected, int64(len(orgIDs)) - result.RowsAffected, nil
}

// GetOrganizationList 获取组织列表
func (r *Repository) GetOrganizationList(db *gorm.DB, queryMap map[string]any, limit, offset int) ([]*model.Organization, int64, error) {
	var organizations []*model.Organization
	var total int64

	// 创建 base session，关联 sys_accounts 表获取账号状态
	baseSession := db.WithContext(r.ctx).
		Table("organizations as org").
		Joins("LEFT JOIN sys_accounts as sys ON org.account_id = sys.id")

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
		return organizations, 0, nil
	}

	// 使用 base session 分页查询数据
	querySession := baseSession.Select("org.*")
	err = querySession.Offset(offset).Limit(limit).
		Order("org.created_at DESC").
		Find(&organizations).Error
	if err != nil {
		return nil, 0, err
	}

	return organizations, total, nil
}

// FindOrganizationIDsByKeyword 根据关键字查找组织ID列表
func (r *Repository) FindOrganizationIDsByKeyword(db *gorm.DB, keyword string) ([]int64, error) {
	var ids []int64
	err := db.WithContext(r.ctx).Model(&model.Organization{}).
		Where("org_name LIKE ? OR contact_person LIKE ? OR contact_phone LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}
