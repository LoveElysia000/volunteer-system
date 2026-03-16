package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

func RegisterImportRouter(r *route.RouterGroup) {
	r.POST("/admin/import/volunteers", handler.ImportVolunteers)
	r.POST("/admin/import/activities", handler.ImportActivities)
}
