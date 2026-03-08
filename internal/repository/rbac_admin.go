package repository

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type RBACRole struct {
	ID          int64     `gorm:"column:id"`
	RoleCode    string    `gorm:"column:role_code"`
	RoleName    string    `gorm:"column:role_name"`
	Description string    `gorm:"column:description"`
	Status      int32     `gorm:"column:status"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

type RBACPermission struct {
	ID          int64     `gorm:"column:id"`
	Resource    string    `gorm:"column:resource"`
	Action      string    `gorm:"column:action"`
	Description string    `gorm:"column:description"`
	Status      int32     `gorm:"column:status"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

type RBACRolePermissionItem struct {
	RoleID       int64  `gorm:"column:role_id"`
	PermissionID int64  `gorm:"column:permission_id"`
	Resource     string `gorm:"column:resource"`
	Action       string `gorm:"column:action"`
	Description  string `gorm:"column:description"`
}

type RBACAccountRoleBinding struct {
	ID        int64      `gorm:"column:id"`
	AccountID int64      `gorm:"column:account_id"`
	RoleID    int64      `gorm:"column:role_id"`
	RoleCode  string     `gorm:"column:role_code"`
	RoleName  string     `gorm:"column:role_name"`
	ScopeType string     `gorm:"column:scope_type"`
	ScopeID   int64      `gorm:"column:scope_id"`
	Status    int32      `gorm:"column:status"`
	GrantedBy int64      `gorm:"column:granted_by"`
	ExpiresAt *time.Time `gorm:"column:expires_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

type RBACPermissionScope struct {
	Resource  string `gorm:"column:resource"`
	Action    string `gorm:"column:action"`
	ScopeType string `gorm:"column:scope_type"`
	ScopeID   int64  `gorm:"column:scope_id"`
}

func (r *Repository) rbacCtx() context.Context {
	if r.ctx != nil {
		return r.ctx
	}
	return context.Background()
}

func (r *Repository) CreateRBACRole(db *gorm.DB, role *RBACRole) error {
	return db.WithContext(r.rbacCtx()).Table("rbac_roles").Create(role).Error
}

func (r *Repository) UpdateRBACRoleByID(db *gorm.DB, roleID int64, updates map[string]any) error {
	return db.WithContext(r.rbacCtx()).Table("rbac_roles").Where("id = ?", roleID).Updates(updates).Error
}

func (r *Repository) GetRBACRoleByID(db *gorm.DB, roleID int64) (*RBACRole, error) {
	var role RBACRole
	if err := db.WithContext(r.rbacCtx()).Table("rbac_roles").Where("id = ?", roleID).Take(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *Repository) GetRBACRoleByCode(db *gorm.DB, roleCode string) (*RBACRole, error) {
	var role RBACRole
	if err := db.WithContext(r.rbacCtx()).Table("rbac_roles").Where("role_code = ?", roleCode).Take(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *Repository) ListRBACRoles(db *gorm.DB, keyword string, includeDisabled bool, limit, offset int) ([]*RBACRole, int64, error) {
	rows := make([]*RBACRole, 0)
	base := db.WithContext(r.rbacCtx()).Table("rbac_roles")
	if keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where("(role_code LIKE ? OR role_name LIKE ?)", like, like)
	}
	if !includeDisabled {
		base = base.Where("status = ?", 1)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return rows, 0, nil
	}

	if limit > 0 {
		base = base.Limit(limit)
	}
	if offset > 0 {
		base = base.Offset(offset)
	}
	if err := base.Order("id ASC").Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) ListRBACPermissions(db *gorm.DB, keyword string, onlyEnabled bool) ([]*RBACPermission, error) {
	rows := make([]*RBACPermission, 0)
	query := db.WithContext(r.rbacCtx()).Table("rbac_permissions")
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("(resource LIKE ? OR action LIKE ? OR description LIKE ?)", like, like, like)
	}
	if onlyEnabled {
		query = query.Where("status = ?", 1)
	}
	if err := query.Order("resource ASC, action ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *Repository) GetRBACPermissionsByIDs(db *gorm.DB, ids []int64) ([]*RBACPermission, error) {
	rows := make([]*RBACPermission, 0)
	if len(ids) == 0 {
		return rows, nil
	}
	if err := db.WithContext(r.rbacCtx()).Table("rbac_permissions").Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *Repository) ListRBACRolePermissions(db *gorm.DB, roleID int64) ([]*RBACRolePermissionItem, error) {
	rows := make([]*RBACRolePermissionItem, 0)
	if roleID <= 0 {
		return rows, nil
	}

	err := db.WithContext(r.rbacCtx()).
		Table("rbac_role_permissions rp").
		Select("rp.role_id, rp.permission_id, p.resource, p.action, p.description").
		Joins("JOIN rbac_permissions p ON p.id = rp.permission_id").
		Where("rp.role_id = ?", roleID).
		Order("p.resource ASC, p.action ASC, p.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *Repository) ReplaceRBACRolePermissions(tx *gorm.DB, roleID int64, permissionIDs []int64) error {
	if err := tx.WithContext(r.rbacCtx()).Table("rbac_role_permissions").Where("role_id = ?", roleID).Delete(nil).Error; err != nil {
		return err
	}
	if len(permissionIDs) == 0 {
		return nil
	}

	rows := make([]map[string]any, 0, len(permissionIDs))
	for _, permissionID := range permissionIDs {
		rows = append(rows, map[string]any{
			"role_id":       roleID,
			"permission_id": permissionID,
		})
	}
	return tx.WithContext(r.rbacCtx()).
		Table("rbac_role_permissions").
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "role_id"}, {Name: "permission_id"}},
			DoNothing: true,
		}).
		Create(&rows).Error
}

func (r *Repository) UpsertRBACAccountRoleBinding(
	tx *gorm.DB,
	accountID, roleID int64,
	scopeType string,
	scopeID int64,
	status int32,
	grantedBy int64,
	expiresAt *time.Time,
) error {
	payload := map[string]any{
		"account_id": accountID,
		"role_id":    roleID,
		"scope_type": scopeType,
		"scope_id":   scopeID,
		"status":     status,
		"granted_by": grantedBy,
		"expires_at": expiresAt,
	}
	return tx.WithContext(r.rbacCtx()).
		Table("rbac_account_roles").
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "account_id"},
				{Name: "role_id"},
				{Name: "scope_type"},
				{Name: "scope_id"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"status":     status,
				"granted_by": grantedBy,
				"expires_at": expiresAt,
				"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
			}),
		}).
		Create(payload).Error
}

func (r *Repository) GetRBACAccountRoleBindingByID(db *gorm.DB, id int64) (*RBACAccountRoleBinding, error) {
	var row RBACAccountRoleBinding
	err := db.WithContext(r.rbacCtx()).
		Table("rbac_account_roles ar").
		Select("ar.*, r.role_code, r.role_name").
		Joins("JOIN rbac_roles r ON r.id = ar.role_id").
		Where("ar.id = ?", id).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) UpdateRBACAccountRoleBindingByID(tx *gorm.DB, id int64, updates map[string]any) error {
	return tx.WithContext(r.rbacCtx()).Table("rbac_account_roles").Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) ListRBACAccountRoleBindings(
	db *gorm.DB,
	accountID int64,
	scopeType string,
	scopeID int64,
	onlyActive bool,
	limit, offset int,
) ([]*RBACAccountRoleBinding, int64, error) {
	rows := make([]*RBACAccountRoleBinding, 0)
	base := db.WithContext(r.rbacCtx()).
		Table("rbac_account_roles ar").
		Select("ar.*, r.role_code, r.role_name").
		Joins("JOIN rbac_roles r ON r.id = ar.role_id")
	if accountID > 0 {
		base = base.Where("ar.account_id = ?", accountID)
	}
	if scopeType != "" {
		base = base.Where("ar.scope_type = ?", scopeType)
	}
	if scopeID > 0 || scopeType == "global" {
		base = base.Where("ar.scope_id = ?", scopeID)
	}
	if onlyActive {
		base = base.Where("ar.status = ?", 1).
			Where("r.status = ?", 1).
			Where("(ar.expires_at IS NULL OR ar.expires_at > CURRENT_TIMESTAMP)")
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return rows, 0, nil
	}

	if limit > 0 {
		base = base.Limit(limit)
	}
	if offset > 0 {
		base = base.Offset(offset)
	}
	if err := base.Order("ar.id DESC").Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (r *Repository) CountActiveGlobalBindingsByRoleCode(db *gorm.DB, roleCode string) (int64, error) {
	var count int64
	err := db.WithContext(r.rbacCtx()).
		Table("rbac_account_roles ar").
		Joins("JOIN rbac_roles r ON r.id = ar.role_id").
		Where("r.role_code = ?", roleCode).
		Where("ar.scope_type = ? AND ar.scope_id = ?", "global", 0).
		Where("ar.status = ?", 1).
		Where("r.status = ?", 1).
		Where("(ar.expires_at IS NULL OR ar.expires_at > CURRENT_TIMESTAMP)").
		Count(&count).Error
	return count, err
}

func (r *Repository) HasActiveRBACBindingByRoleCodeAndScope(
	db *gorm.DB,
	accountID int64,
	roleCode string,
	scopeType string,
	scopeID int64,
) (bool, error) {
	if accountID <= 0 || roleCode == "" || scopeType == "" {
		return false, nil
	}

	var count int64
	err := db.WithContext(r.rbacCtx()).
		Table("rbac_account_roles ar").
		Joins("JOIN rbac_roles r ON r.id = ar.role_id").
		Where("ar.account_id = ?", accountID).
		Where("r.role_code = ?", roleCode).
		Where("ar.scope_type = ? AND ar.scope_id = ?", scopeType, scopeID).
		Where("ar.status = ?", 1).
		Where("r.status = ?", 1).
		Where("(ar.expires_at IS NULL OR ar.expires_at > CURRENT_TIMESTAMP)").
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Repository) CreateRBACChangeLog(tx *gorm.DB, payload map[string]any) error {
	return tx.WithContext(r.rbacCtx()).Table("rbac_change_logs").Create(payload).Error
}

func (r *Repository) ListAccountPermissionScopes(db *gorm.DB, accountID int64) ([]*RBACPermissionScope, error) {
	rows := make([]*RBACPermissionScope, 0)
	if accountID <= 0 {
		return rows, nil
	}
	err := db.WithContext(r.rbacCtx()).
		Table("rbac_account_roles ar").
		Select("DISTINCT p.resource, p.action, ar.scope_type, ar.scope_id").
		Joins("JOIN rbac_roles r ON r.id = ar.role_id").
		Joins("JOIN rbac_role_permissions rp ON rp.role_id = r.id").
		Joins("JOIN rbac_permissions p ON p.id = rp.permission_id").
		Where("ar.account_id = ?", accountID).
		Where("ar.status = ?", 1).
		Where("r.status = ?", 1).
		Where("p.status = ?", 1).
		Where("(ar.expires_at IS NULL OR ar.expires_at > CURRENT_TIMESTAMP)").
		Order("p.resource ASC, p.action ASC, ar.scope_type ASC, ar.scope_id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
