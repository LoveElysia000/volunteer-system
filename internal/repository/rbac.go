package repository

import (
	"context"

	"gorm.io/gorm"
)

// HasPermissionByScope checks whether account has resource/action permission under exact scope.
func (r *Repository) HasPermissionByScope(db *gorm.DB, accountID int64, scopeType string, scopeID int64, resource, action string) (bool, error) {
	if accountID <= 0 || scopeType == "" || resource == "" || action == "" {
		return false, nil
	}

	ctx := r.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	var count int64
	err := db.WithContext(ctx).
		Table("rbac_account_roles as ar").
		Joins("JOIN rbac_roles as r ON r.id = ar.role_id").
		Joins("JOIN rbac_role_permissions as rp ON rp.role_id = r.id").
		Joins("JOIN rbac_permissions as p ON p.id = rp.permission_id").
		Where("ar.account_id = ?", accountID).
		Where("ar.status = ?", 1).
		Where("r.status = ?", 1).
		Where("p.status = ?", 1).
		Where("ar.scope_type = ? AND ar.scope_id = ?", scopeType, scopeID).
		Where("(ar.expires_at IS NULL OR ar.expires_at > CURRENT_TIMESTAMP)").
		Where("p.resource = ? AND p.action = ?", resource, action).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// HasAnyOrgPermission checks whether account has resource/action permission in any org scope.
func (r *Repository) HasAnyOrgPermission(db *gorm.DB, accountID int64, resource, action string) (bool, error) {
	if accountID <= 0 || resource == "" || action == "" {
		return false, nil
	}

	ctx := r.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	var count int64
	err := db.WithContext(ctx).
		Table("rbac_account_roles as ar").
		Joins("JOIN rbac_roles as r ON r.id = ar.role_id").
		Joins("JOIN rbac_role_permissions as rp ON rp.role_id = r.id").
		Joins("JOIN rbac_permissions as p ON p.id = rp.permission_id").
		Where("ar.account_id = ?", accountID).
		Where("ar.status = ?", 1).
		Where("r.status = ?", 1).
		Where("p.status = ?", 1).
		Where("ar.scope_type = ?", "org").
		Where("(ar.expires_at IS NULL OR ar.expires_at > CURRENT_TIMESTAMP)").
		Where("p.resource = ? AND p.action = ?", resource, action).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListOrgScopeIDsByPermission returns distinct org scope ids that account can access for a permission.
func (r *Repository) ListOrgScopeIDsByPermission(
	db *gorm.DB,
	accountID int64,
	resource, action string,
	limit int,
) ([]int64, error) {
	if accountID <= 0 || resource == "" || action == "" {
		return []int64{}, nil
	}

	ctx := r.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	orgIDs := make([]int64, 0)
	query := db.WithContext(ctx).
		Table("rbac_account_roles as ar").
		Distinct("ar.scope_id").
		Joins("JOIN rbac_roles as r ON r.id = ar.role_id").
		Joins("JOIN rbac_role_permissions as rp ON rp.role_id = r.id").
		Joins("JOIN rbac_permissions as p ON p.id = rp.permission_id").
		Where("ar.account_id = ?", accountID).
		Where("ar.status = ?", 1).
		Where("r.status = ?", 1).
		Where("p.status = ?", 1).
		Where("ar.scope_type = ?", "org").
		Where("(ar.expires_at IS NULL OR ar.expires_at > CURRENT_TIMESTAMP)").
		Where("p.resource = ? AND p.action = ?", resource, action).
		Order("ar.scope_id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Pluck("ar.scope_id", &orgIDs).Error; err != nil {
		return nil, err
	}
	return orgIDs, nil
}
