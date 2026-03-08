package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

const (
	defaultRBACPageSize = 20
	maxRBACPageSize     = 100
)

var roleCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

type AuthzService struct {
	Service
}

func NewAuthzService(ctx context.Context, c *app.RequestContext) *AuthzService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &AuthzService{
		Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
}

func normalizeRBACPage(page, pageSize int32) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultRBACPageSize
	}
	if pageSize > maxRBACPageSize {
		pageSize = maxRBACPageSize
	}
	return int(page), int(pageSize)
}

func (s *AuthzService) requireRBACManageOperator() (int64, error) {
	operatorID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return 0, err
	}
	if err := s.requireGlobalPermission(
		operatorID,
		model.PermissionResourceRBAC,
		model.PermissionActionManage,
	); err != nil {
		return 0, err
	}
	return operatorID, nil
}

func (s *AuthzService) ListRoles(req *api.RoleListRequest) (*api.RoleListResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if _, err := s.requireRBACManageOperator(); err != nil {
		return nil, err
	}

	page, pageSize := normalizeRBACPage(req.Page, req.PageSize)
	rows, total, err := s.repo.ListRBACRoles(
		s.repo.DB,
		strings.TrimSpace(req.Keyword),
		req.IncludeDisabled,
		pageSize,
		(page-1)*pageSize,
	)
	if err != nil {
		return nil, err
	}
	resp := &api.RoleListResponse{
		Total: int32(total),
		List:  make([]*api.RoleInfo, 0, len(rows)),
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		resp.List = append(resp.List, &api.RoleInfo{
			Id:          row.ID,
			RoleCode:    row.RoleCode,
			RoleName:    row.RoleName,
			Description: row.Description,
			Status:      row.Status,
			CreatedAt:   util.FormatDateTimeOrEmpty(row.CreatedAt),
			UpdatedAt:   util.FormatDateTimeOrEmpty(row.UpdatedAt),
		})
	}
	return resp, nil
}

func (s *AuthzService) CreateRole(req *api.RoleCreateRequest) (*api.RoleCreateResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	operatorID, err := s.requireRBACManageOperator()
	if err != nil {
		return nil, err
	}

	roleCode := strings.TrimSpace(strings.ToLower(req.RoleCode))
	roleName := strings.TrimSpace(req.RoleName)
	if !roleCodePattern.MatchString(roleCode) {
		return nil, errors.New("角色编码格式不合法")
	}
	if roleName == "" {
		return nil, errors.New("角色名称不能为空")
	}

	existed, err := s.repo.GetRBACRoleByCode(s.repo.DB, roleCode)
	if err == nil && existed != nil {
		return nil, errors.New("角色编码已存在")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now()
	role := &repository.RBACRole{
		RoleCode:    roleCode,
		RoleName:    roleName,
		Description: strings.TrimSpace(req.Description),
		Status:      1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	err = s.withTransaction(func(tx *gorm.DB) error {
		if err := s.repo.CreateRBACRole(tx, role); err != nil {
			return err
		}
		return s.repo.CreateRBACChangeLog(tx, buildRBACChangeLogPayload(
			operatorID,
			0,
			role.ID,
			"global",
			0,
			"role.create",
			nil,
			role,
			"",
		))
	})
	if err != nil {
		return nil, err
	}

	return &api.RoleCreateResponse{Id: role.ID}, nil
}

func (s *AuthzService) UpdateRole(req *api.RoleUpdateRequest) (*api.RoleUpdateResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	operatorID, err := s.requireRBACManageOperator()
	if err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, errors.New("角色ID不能为空")
	}

	role, err := s.repo.GetRBACRoleByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("角色不存在")
		}
		return nil, err
	}

	updates := map[string]any{}
	if name := strings.TrimSpace(req.RoleName); name != "" {
		updates["role_name"] = name
	}
	if req.Description != "" {
		updates["description"] = strings.TrimSpace(req.Description)
	}
	if len(updates) == 0 {
		return nil, errors.New("没有可更新字段")
	}
	updates["updated_at"] = time.Now()

	before := *role
	err = s.withTransaction(func(tx *gorm.DB) error {
		if err := s.repo.UpdateRBACRoleByID(tx, role.ID, updates); err != nil {
			return err
		}
		after, err := s.repo.GetRBACRoleByID(tx, role.ID)
		if err != nil {
			return err
		}
		return s.repo.CreateRBACChangeLog(tx, buildRBACChangeLogPayload(
			operatorID,
			0,
			role.ID,
			"global",
			0,
			"role.update",
			before,
			after,
			"",
		))
	})
	if err != nil {
		return nil, err
	}
	return &api.RoleUpdateResponse{Message: "updated"}, nil
}

func (s *AuthzService) UpdateRoleStatus(req *api.RoleStatusUpdateRequest) (*api.RoleStatusUpdateResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	operatorID, err := s.requireRBACManageOperator()
	if err != nil {
		return nil, err
	}
	if req.Id <= 0 {
		return nil, errors.New("角色ID不能为空")
	}
	if req.Status != 0 && req.Status != 1 {
		return nil, errors.New("状态值不合法")
	}

	role, err := s.repo.GetRBACRoleByID(s.repo.DB, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("角色不存在")
		}
		return nil, err
	}
	if role.RoleCode == model.RBACRoleSuperAdmin && req.Status != 1 {
		return nil, errors.New("super_admin 角色不可禁用")
	}

	before := *role
	err = s.withTransaction(func(tx *gorm.DB) error {
		if err := s.repo.UpdateRBACRoleByID(tx, role.ID, map[string]any{
			"status":     req.Status,
			"updated_at": time.Now(),
		}); err != nil {
			return err
		}
		after, err := s.repo.GetRBACRoleByID(tx, role.ID)
		if err != nil {
			return err
		}
		return s.repo.CreateRBACChangeLog(tx, buildRBACChangeLogPayload(
			operatorID,
			0,
			role.ID,
			"global",
			0,
			"role.status.update",
			before,
			after,
			"",
		))
	})
	if err != nil {
		return nil, err
	}
	return &api.RoleStatusUpdateResponse{Message: "updated"}, nil
}

func (s *AuthzService) ListPermissions(req *api.PermissionListRequest) (*api.PermissionListResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if _, err := s.requireRBACManageOperator(); err != nil {
		return nil, err
	}

	rows, err := s.repo.ListRBACPermissions(
		s.repo.DB,
		strings.TrimSpace(req.Keyword),
		req.OnlyEnabled,
	)
	if err != nil {
		return nil, err
	}
	resp := &api.PermissionListResponse{
		List: make([]*api.PermissionInfo, 0, len(rows)),
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		resp.List = append(resp.List, &api.PermissionInfo{
			Id:          row.ID,
			Resource:    row.Resource,
			Action:      row.Action,
			Description: row.Description,
			Status:      row.Status,
		})
	}
	return resp, nil
}

func (s *AuthzService) GetRolePermissions(req *api.RolePermissionsRequest) (*api.RolePermissionsResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if _, err := s.requireRBACManageOperator(); err != nil {
		return nil, err
	}
	if req.RoleId <= 0 {
		return nil, errors.New("角色ID不能为空")
	}
	if _, err := s.repo.GetRBACRoleByID(s.repo.DB, req.RoleId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("角色不存在")
		}
		return nil, err
	}

	rows, err := s.repo.ListRBACRolePermissions(s.repo.DB, req.RoleId)
	if err != nil {
		return nil, err
	}
	resp := &api.RolePermissionsResponse{
		RoleId:      req.RoleId,
		Permissions: make([]*api.RolePermissionItem, 0, len(rows)),
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		resp.Permissions = append(resp.Permissions, &api.RolePermissionItem{
			PermissionId: row.PermissionID,
			Resource:     row.Resource,
			Action:       row.Action,
			Description:  row.Description,
		})
	}
	return resp, nil
}

func (s *AuthzService) SetRolePermissions(req *api.RolePermissionsSetRequest) (*api.RolePermissionsSetResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	operatorID, err := s.requireRBACManageOperator()
	if err != nil {
		return nil, err
	}
	if req.RoleId <= 0 {
		return nil, errors.New("角色ID不能为空")
	}

	role, err := s.repo.GetRBACRoleByID(s.repo.DB, req.RoleId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("角色不存在")
		}
		return nil, err
	}

	permissionIDs := util.UniquePositiveInt64(req.PermissionIds)
	permissions, err := s.repo.GetRBACPermissionsByIDs(s.repo.DB, permissionIDs)
	if err != nil {
		return nil, err
	}
	if len(permissionIDs) != len(permissions) {
		return nil, errors.New("存在无效权限ID")
	}

	containsRBACManage := false
	for _, permission := range permissions {
		if permission == nil {
			continue
		}
		if permission.Resource == model.PermissionResourceRBAC && permission.Action == model.PermissionActionManage {
			containsRBACManage = true
			break
		}
	}
	if containsRBACManage && role.RoleCode != model.RBACRoleSuperAdmin {
		return nil, errors.New("仅 super_admin 角色可拥有 rbac.manage")
	}
	if role.RoleCode == model.RBACRoleSuperAdmin {
		if !containsRBACManage {
			return nil, errors.New("super_admin 角色必须保留 rbac.manage")
		}
		if len(permissionIDs) != 1 {
			return nil, errors.New("super_admin 角色仅允许保留 rbac.manage")
		}
	}

	before, err := s.repo.ListRBACRolePermissions(s.repo.DB, role.ID)
	if err != nil {
		return nil, err
	}
	err = s.withTransaction(func(tx *gorm.DB) error {
		if err := s.repo.ReplaceRBACRolePermissions(tx, role.ID, permissionIDs); err != nil {
			return err
		}
		after, err := s.repo.ListRBACRolePermissions(tx, role.ID)
		if err != nil {
			return err
		}
		return s.repo.CreateRBACChangeLog(tx, buildRBACChangeLogPayload(
			operatorID,
			0,
			role.ID,
			"global",
			0,
			"role.permissions.set",
			before,
			after,
			"",
		))
	})
	if err != nil {
		return nil, err
	}
	return &api.RolePermissionsSetResponse{Message: "updated"}, nil
}

func (s *AuthzService) GrantRole(req *api.AccountRoleGrantRequest) (*api.AccountRoleGrantResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	operatorID, err := s.requireRBACManageOperator()
	if err != nil {
		return nil, err
	}
	if req.AccountId <= 0 {
		return nil, errors.New("目标账号ID不能为空")
	}
	if req.RoleId <= 0 {
		return nil, errors.New("角色ID不能为空")
	}
	scopeType := strings.ToLower(strings.TrimSpace(req.ScopeType))
	if scopeType != model.RBACScopeGlobal && scopeType != model.RBACScopeOrg {
		return nil, errors.New("scopeType 仅支持 global/org")
	}

	role, err := s.repo.GetRBACRoleByID(s.repo.DB, req.RoleId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("角色不存在")
		}
		return nil, err
	}
	if role.Status != 1 {
		return nil, errors.New("角色已禁用")
	}

	scopeID := req.ScopeId
	if scopeType == model.RBACScopeGlobal {
		scopeID = 0
	} else if scopeID <= 0 {
		return nil, errors.New("组织作用域必须提供 scopeId")
	}
	if role.RoleCode == model.RBACRoleSuperAdmin {
		if scopeType != model.RBACScopeGlobal || scopeID != 0 {
			return nil, errors.New("super_admin 仅允许 global 作用域")
		}
	}
	if _, err := s.repo.FindByID(s.repo.DB, req.AccountId); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("目标账号不存在")
		}
		return nil, err
	}
	if scopeType == model.RBACScopeOrg {
		if _, err := s.repo.GetOrganizationByID(s.repo.DB, scopeID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("组织不存在")
			}
			return nil, err
		}
	}

	var expiresAtPtr *time.Time
	if strings.TrimSpace(req.ExpiresAt) != "" {
		expiresAt, err := util.ParseDateTime(strings.TrimSpace(req.ExpiresAt))
		if err != nil {
			return nil, errors.New("expiresAt 格式错误")
		}
		expiresAtPtr = &expiresAt
	}

	err = s.withTransaction(func(tx *gorm.DB) error {
		if err := s.repo.UpsertRBACAccountRoleBinding(
			tx,
			req.AccountId,
			role.ID,
			scopeType,
			scopeID,
			1,
			operatorID,
			expiresAtPtr,
		); err != nil {
			return err
		}
		return s.repo.CreateRBACChangeLog(tx, buildRBACChangeLogPayload(
			operatorID,
			req.AccountId,
			role.ID,
			scopeType,
			scopeID,
			"grant.create_or_update",
			nil,
			map[string]any{
				"account_id": req.AccountId,
				"role_id":    role.ID,
				"scope_type": scopeType,
				"scope_id":   scopeID,
				"status":     1,
				"granted_by": operatorID,
				"expires_at": util.FormatDateTimePtr(expiresAtPtr),
			},
			strings.TrimSpace(req.Remark),
		))
	})
	if err != nil {
		return nil, err
	}
	return &api.AccountRoleGrantResponse{Message: "granted"}, nil
}

func (s *AuthzService) RevokeRole(req *api.AccountRoleRevokeRequest) (*api.AccountRoleRevokeResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	operatorID, err := s.requireRBACManageOperator()
	if err != nil {
		return nil, err
	}
	if req.BindingId <= 0 {
		return nil, errors.New("授权绑定ID不能为空")
	}

	err = s.withTransaction(func(tx *gorm.DB) error {
		binding, err := s.repo.GetRBACAccountRoleBindingByID(tx, req.BindingId)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("授权绑定不存在")
			}
			return err
		}

		if binding.RoleCode == model.RBACRoleSuperAdmin &&
			binding.ScopeType == model.RBACScopeGlobal &&
			binding.ScopeID == 0 &&
			binding.Status == 1 {
			activeSuperCount, err := s.repo.CountActiveGlobalBindingsByRoleCode(tx, model.RBACRoleSuperAdmin)
			if err != nil {
				return err
			}
			if activeSuperCount <= 1 {
				return errors.New("不能撤销最后一个 super_admin")
			}
		}

		before := *binding
		if binding.Status == 1 {
			if err := s.repo.UpdateRBACAccountRoleBindingByID(tx, binding.ID, map[string]any{
				"status":     0,
				"updated_at": time.Now(),
			}); err != nil {
				return err
			}
		}
		return s.repo.CreateRBACChangeLog(tx, buildRBACChangeLogPayload(
			operatorID,
			binding.AccountID,
			binding.RoleID,
			binding.ScopeType,
			binding.ScopeID,
			"grant.revoke",
			before,
			map[string]any{
				"binding_id": binding.ID,
				"status":     0,
			},
			strings.TrimSpace(req.Remark),
		))
	})
	if err != nil {
		return nil, err
	}

	return &api.AccountRoleRevokeResponse{Message: "revoked"}, nil
}

func (s *AuthzService) ListAccountRoleBindings(req *api.AccountRoleBindingListRequest) (*api.AccountRoleBindingListResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if _, err := s.requireRBACManageOperator(); err != nil {
		return nil, err
	}

	page, pageSize := normalizeRBACPage(req.Page, req.PageSize)
	scopeType := strings.ToLower(strings.TrimSpace(req.ScopeType))
	if scopeType != "" && scopeType != model.RBACScopeGlobal && scopeType != model.RBACScopeOrg {
		return nil, errors.New("scopeType 仅支持 global/org")
	}

	rows, total, err := s.repo.ListRBACAccountRoleBindings(
		s.repo.DB,
		req.AccountId,
		scopeType,
		req.ScopeId,
		req.OnlyActive,
		pageSize,
		(page-1)*pageSize,
	)
	if err != nil {
		return nil, err
	}
	resp := &api.AccountRoleBindingListResponse{
		Total: int32(total),
		List:  make([]*api.AccountRoleBindingInfo, 0, len(rows)),
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		resp.List = append(resp.List, convertRoleBindingToAPI(row))
	}
	return resp, nil
}

func (s *AuthzService) MyAuthorization(req *api.MyAuthorizationRequest) (*api.MyAuthorizationResponse, error) {
	_ = req
	accountID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	roles, _, err := s.repo.ListRBACAccountRoleBindings(
		s.repo.DB,
		accountID,
		"",
		0,
		true,
		0,
		0,
	)
	if err != nil {
		return nil, err
	}
	permissions, err := s.repo.ListAccountPermissionScopes(s.repo.DB, accountID)
	if err != nil {
		return nil, err
	}

	resp := &api.MyAuthorizationResponse{
		AccountId:   accountID,
		Roles:       make([]*api.AccountRoleBindingInfo, 0, len(roles)),
		Permissions: make([]*api.MyPermissionScope, 0, len(permissions)),
	}
	for _, row := range roles {
		if row == nil {
			continue
		}
		resp.Roles = append(resp.Roles, convertRoleBindingToAPI(row))
	}
	for _, row := range permissions {
		if row == nil {
			continue
		}
		resp.Permissions = append(resp.Permissions, &api.MyPermissionScope{
			Resource:  row.Resource,
			Action:    row.Action,
			ScopeType: row.ScopeType,
			ScopeId:   row.ScopeID,
		})
	}
	return resp, nil
}

func convertRoleBindingToAPI(row *repository.RBACAccountRoleBinding) *api.AccountRoleBindingInfo {
	if row == nil {
		return nil
	}
	return &api.AccountRoleBindingInfo{
		BindingId: row.ID,
		AccountId: row.AccountID,
		RoleId:    row.RoleID,
		RoleCode:  row.RoleCode,
		RoleName:  row.RoleName,
		ScopeType: row.ScopeType,
		ScopeId:   row.ScopeID,
		Status:    row.Status,
		GrantedBy: row.GrantedBy,
		ExpiresAt: util.FormatDateTimePtr(row.ExpiresAt),
		CreatedAt: util.FormatDateTimeOrEmpty(row.CreatedAt),
		UpdatedAt: util.FormatDateTimeOrEmpty(row.UpdatedAt),
	}
}

func buildRBACChangeLogPayload(
	operatorID int64,
	targetAccountID int64,
	targetRoleID int64,
	scopeType string,
	scopeID int64,
	changeType string,
	before any,
	after any,
	remark string,
) map[string]any {
	return map[string]any{
		"operator_id":       operatorID,
		"target_account_id": targetAccountID,
		"target_role_id":    targetRoleID,
		"scope_type":        scopeType,
		"scope_id":          scopeID,
		"change_type":       changeType,
		"before_value":      mustJSON(before),
		"after_value":       mustJSON(after),
		"remark":            remark,
	}
}

func mustJSON(v any) string {
	if v == nil {
		return ""
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":"%s"}`, strings.ReplaceAll(err.Error(), `"`, `'`))
	}
	return string(raw)
}
