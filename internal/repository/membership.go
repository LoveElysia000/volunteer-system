package repository

import (
	"errors"
	"volunteer-system/internal/model"

	"gorm.io/gorm"
)

// FindMembershipByOrgAndVolunteer finds membership by org and volunteer.
func (r *Repository) FindMembershipByOrgAndVolunteer(db *gorm.DB, orgID, volunteerID int64) (*model.OrgMember, error) {
	var member model.OrgMember
	err := db.WithContext(r.ctx).Model(&model.OrgMember{}).
		Where("org_id = ? AND volunteer_id = ?", orgID, volunteerID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &member, nil
}

// GetMembershipByID finds membership by id.
func (r *Repository) GetMembershipByID(db *gorm.DB, id int64) (*model.OrgMember, error) {
	var member model.OrgMember
	err := db.WithContext(r.ctx).Model(&model.OrgMember{}).Where("id = ?", id).First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	return &member, nil
}

// GetMembershipsByIDs returns memberships by ids.
func (r *Repository) GetMembershipsByIDs(db *gorm.DB, ids []int64) ([]*model.OrgMember, error) {
	members := make([]*model.OrgMember, 0)
	if len(ids) == 0 {
		return members, nil
	}

	if err := db.WithContext(r.ctx).Model(&model.OrgMember{}).Where("id IN ?", ids).Find(&members).Error; err != nil {
		return nil, err
	}

	return members, nil
}

// CreateMembership creates a membership record.
func (r *Repository) CreateMembership(db *gorm.DB, member *model.OrgMember) error {
	return db.WithContext(r.ctx).Create(member).Error
}

// UpdateMembershipFields updates membership fields by id.
func (r *Repository) UpdateMembershipFields(db *gorm.DB, id int64, updates map[string]any) error {
	return db.WithContext(r.ctx).Where("id = ?", id).Updates(updates).Error
}

// GetOrganizationMembers returns members for an organization with filters.
func (r *Repository) GetOrganizationMembers(db *gorm.DB, orgID int64, status, role int32, keyword string, limit, offset int) ([]*model.OrgMember, int64, error) {
	var members []*model.OrgMember
	var total int64

	base := db.WithContext(r.ctx).
		Table("org_members as m").
		Joins("LEFT JOIN volunteers v ON m.volunteer_id = v.id").
		Where("m.org_id = ?", orgID)

	if status > 0 {
		base = base.Where("m.status = ?", status)
	}
	if role > 0 {
		base = base.Where("m.role = ?", role)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where("v.real_name LIKE ? OR v.id_card LIKE ?", like, like)
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return members, 0, nil
	}

	err := base.Select("m.*").
		Order("m.created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&members).Error
	if err != nil {
		return nil, 0, err
	}

	return members, total, nil
}

// GetVolunteerOrganizations returns organizations joined by a volunteer.
func (r *Repository) GetVolunteerOrganizations(db *gorm.DB, volunteerID int64, status int32, limit, offset int) ([]*model.OrgMember, int64, error) {
	var members []*model.OrgMember
	var total int64

	base := db.WithContext(r.ctx).
		Table("org_members as m").
		Where("m.volunteer_id = ?", volunteerID)

	if status > 0 {
		base = base.Where("m.status = ?", status)
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return members, 0, nil
	}

	err := base.Select("m.*").
		Order("m.created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&members).Error
	if err != nil {
		return nil, 0, err
	}

	return members, total, nil
}

// GetMembershipStatusCounts returns counts by status for an organization.
func (r *Repository) GetMembershipStatusCounts(db *gorm.DB, orgID int64) (map[int32]int64, int64, error) {
	type statusCount struct {
		Status int32 `gorm:"column:status"`
		Count  int64 `gorm:"column:count"`
	}

	base := db.WithContext(r.ctx).Model(&model.OrgMember{})
	if orgID > 0 {
		base = base.Where("org_id = ?", orgID)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []statusCount
	if err := base.Select("status, COUNT(*) as count").Group("status").Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	result := make(map[int32]int64)
	for _, row := range rows {
		result[row.Status] = row.Count
	}

	return result, total, nil
}
