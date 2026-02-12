package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

func RegisterOrganizationRouter(r *route.RouterGroup) {
	// 组织列表
	r.POST("/organizations/list", handler.OrganizationList)
	// 组织详情
	r.GET("/organizations/:id", handler.OrganizationDetail)
	// 创建组织
	r.POST("/organizations/create", handler.CreateOrganization)
	// 更新组织
	r.PUT("/organizations/:id", handler.UpdateOrganization)
	// 删除组织
	r.DELETE("/organizations/:id", handler.DeleteOrganization)
	// 停用组织
	r.POST("/organizations/:id/disable", handler.DisableOrganization)
	// 启用组织
	r.POST("/organizations/:id/enable", handler.EnableOrganization)
	// 搜索组织
	r.POST("/organizations/search", handler.SearchOrganizations)
	// 批量删除组织
	r.POST("/organizations/bulk-delete", handler.BulkDeleteOrganizations)
}
