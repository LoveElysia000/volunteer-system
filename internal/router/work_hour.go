package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

// RegisterWorkHourRouter 注册工时相关路由
func RegisterWorkHourRouter(r *route.RouterGroup) {
	r.POST("/work-hours/list", handler.WorkHourLogList)
	r.POST("/work-hours/void", handler.VoidWorkHour)
	r.POST("/work-hours/recalculate", handler.RecalculateWorkHour)
}
