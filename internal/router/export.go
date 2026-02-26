package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

// RegisterExportRouter 注册导出相关路由
func RegisterExportRouter(r *route.RouterGroup) {
	r.POST("/admin/export/volunteers", handler.ExportVolunteers)
	r.POST("/admin/export/activities", handler.ExportActivities)
}
