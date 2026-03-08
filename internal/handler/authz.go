package handler

import (
	"context"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func AuthzListRoles(ctx context.Context, c *app.RequestContext) {
	var req api.RoleListRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewAuthzService(ctx, c).ListRoles(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

func AuthzCreateRole(ctx context.Context, c *app.RequestContext) {
	var req api.RoleCreateRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewAuthzService(ctx, c).CreateRole(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

func AuthzUpdateRole(ctx context.Context, c *app.RequestContext) {
	var req api.RoleUpdateRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewAuthzService(ctx, c).UpdateRole(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

func AuthzUpdateRoleStatus(ctx context.Context, c *app.RequestContext) {
	var req api.RoleStatusUpdateRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewAuthzService(ctx, c).UpdateRoleStatus(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

func AuthzListPermissions(ctx context.Context, c *app.RequestContext) {
	var req api.PermissionListRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewAuthzService(ctx, c).ListPermissions(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

func AuthzGetRolePermissions(ctx context.Context, c *app.RequestContext) {
	var req api.RolePermissionsRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewAuthzService(ctx, c).GetRolePermissions(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

func AuthzSetRolePermissions(ctx context.Context, c *app.RequestContext) {
	var req api.RolePermissionsSetRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewAuthzService(ctx, c).SetRolePermissions(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

func AuthzGrantRole(ctx context.Context, c *app.RequestContext) {
	var req api.AccountRoleGrantRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewAuthzService(ctx, c).GrantRole(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

func AuthzRevokeRole(ctx context.Context, c *app.RequestContext) {
	var req api.AccountRoleRevokeRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewAuthzService(ctx, c).RevokeRole(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

func AuthzListAccountRoleBindings(ctx context.Context, c *app.RequestContext) {
	var req api.AccountRoleBindingListRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewAuthzService(ctx, c).ListAccountRoleBindings(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

func AuthzMyAuthorization(ctx context.Context, c *app.RequestContext) {
	var req api.MyAuthorizationRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewAuthzService(ctx, c).MyAuthorization(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}
