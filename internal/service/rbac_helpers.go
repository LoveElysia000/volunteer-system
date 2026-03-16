package service

import (
	"context"
	"errors"
	"fmt"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

func (s *Service) hasPermissionByScope(accountID int64, scopeType string, scopeID int64, resource, action string) (bool, error) {
	if s == nil || s.repo == nil || s.repo.DB == nil {
		return false, errors.New("repository not initialized")
	}
	return s.repo.HasPermissionByScope(s.repo.DB, accountID, scopeType, scopeID, resource, action)
}

func (s *Service) requireOrgPermission(accountID, orgID int64, resource, action string) error {
	if accountID <= 0 {
		return errUnauthorized("未登录或认证无效")
	}
	if orgID <= 0 {
		return errBadRequest("组织ID不能为空")
	}

	allowed, err := s.hasPermissionByScope(accountID, model.RBACScopeGlobal, 0, resource, action)
	if err != nil {
		return err
	}
	if allowed {
		return nil
	}

	allowed, err = s.hasPermissionByScope(accountID, model.RBACScopeOrg, orgID, resource, action)
	if err != nil {
		return err
	}
	if allowed {
		return nil
	}

	return errForbidden("无权操作该组织")
}

func (s *Service) requireGlobalPermission(accountID int64, resource, action string) error {
	if accountID <= 0 {
		return errUnauthorized("未登录或认证无效")
	}

	allowed, err := s.hasPermissionByScope(accountID, model.RBACScopeGlobal, 0, resource, action)
	if err != nil {
		return err
	}
	if allowed {
		return nil
	}

	return errForbidden("无权执行该操作")
}

func (s *Service) ensureDefaultRBACBinding(db *gorm.DB, account *model.SysAccount) error {
	if account == nil || account.ID <= 0 {
		return nil
	}

	switch account.IdentityType {
	case model.RegisterTypeVolunteerCode:
		return s.ensureRoleBindingIfMissing(db, account.ID, model.RBACRoleVolunteer, model.RBACScopeGlobal, 0)
	case model.RegisterTypeOrganizationCode:
		orgs, err := s.repo.FindOrganizationByAccountID(db, account.ID)
		if err != nil {
			return err
		}
		if len(orgs) == 0 {
			log.Warn("默认角色自愈跳过: account_id=%d 无组织记录", account.ID)
			return nil
		}
		for _, org := range orgs {
			if org == nil || org.ID <= 0 {
				continue
			}
			if err := s.ensureRoleBindingIfMissing(db, account.ID, model.RBACRoleOrgOwner, model.RBACScopeOrg, org.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Service) ensureRoleBindingIfMissing(
	db *gorm.DB,
	accountID int64,
	roleCode string,
	scopeType string,
	scopeID int64,
) error {
	bound, err := s.repo.HasActiveRBACBindingByRoleCodeAndScope(db, accountID, roleCode, scopeType, scopeID)
	if err != nil {
		return err
	}
	if bound {
		return nil
	}

	role, err := s.repo.GetRBACRoleByCode(db, roleCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("默认角色自愈跳过: role_code=%s 未初始化", roleCode)
			return nil
		}
		return err
	}
	if role == nil || role.Status != 1 {
		log.Warn("默认角色自愈跳过: role_code=%s 不可用", roleCode)
		return nil
	}

	return s.repo.UpsertRBACAccountRoleBinding(
		db,
		accountID,
		role.ID,
		scopeType,
		scopeID,
		1,
		accountID,
		nil,
	)
}

func (s *Service) bindDefaultRole(db *gorm.DB, accountID int64, roleCode, scopeType string, scopeID int64) error {
	role, err := s.repo.GetRBACRoleByCode(db, roleCode)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("默认角色未初始化: %s", roleCode)
		}
		return err
	}
	if role == nil {
		return fmt.Errorf("默认角色未初始化: %s", roleCode)
	}
	if role.Status != 1 {
		return fmt.Errorf("默认角色已禁用: %s", roleCode)
	}
	return s.repo.UpsertRBACAccountRoleBinding(
		db,
		accountID,
		role.ID,
		scopeType,
		scopeID,
		1,
		accountID,
		nil,
	)
}

type RBACReconcileService struct {
	Service
}

func NewRBACReconcileService(ctx context.Context, c *app.RequestContext) *RBACReconcileService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &RBACReconcileService{
		Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
}

// ReconcileMissingDefaultRoles scans active accounts and ensures default RBAC bindings.
// It returns the number of accounts processed without errors.
func (s *RBACReconcileService) ReconcileMissingDefaultRoles(limit, offset int) (int, error) {
	if limit <= 0 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	accounts, err := s.repo.ListActiveAccounts(s.repo.DB, limit, offset)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, account := range accounts {
		if account == nil || account.ID <= 0 {
			continue
		}
		if err := s.ensureDefaultRBACBinding(s.repo.DB, account); err != nil {
			log.Warn("RBAC回补失败: account_id=%d, err=%v", account.ID, err)
			continue
		}
		processed++
	}

	return processed, nil
}
