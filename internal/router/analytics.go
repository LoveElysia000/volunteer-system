package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

func RegisterAnalyticsRouter(r *route.RouterGroup) {
	r.GET("/analytics/org/funnel", handler.OrgFunnelSummary)
	r.GET("/analytics/org/dashboard", handler.OpsDashboardSummary)
}
