package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

// RegisterAuthzRouter registers RBAC governance routes.
func RegisterAuthzRouter(r *route.RouterGroup) {
	r.GET("/authz/roles", handler.AuthzListRoles)
	r.POST("/authz/roles", handler.AuthzCreateRole)
	r.PUT("/authz/roles/:id", handler.AuthzUpdateRole)
	r.POST("/authz/roles/:id/status", handler.AuthzUpdateRoleStatus)
	r.GET("/authz/permissions", handler.AuthzListPermissions)
	r.GET("/authz/roles/:roleId/permissions", handler.AuthzGetRolePermissions)
	r.POST("/authz/roles/:roleId/permissions/set", handler.AuthzSetRolePermissions)
	r.GET("/authz/grants", handler.AuthzListAccountRoleBindings)
	r.POST("/authz/grants", handler.AuthzGrantRole)
	r.POST("/authz/grants/:bindingId/revoke", handler.AuthzRevokeRole)
	r.GET("/authz/me", handler.AuthzMyAuthorization)
}
