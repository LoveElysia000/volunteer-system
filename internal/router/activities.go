package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

// RegisterActivityRouter 注册活动相关路由
func RegisterActivityRouter(r *route.RouterGroup) {
	// 志愿者端 - 活动浏览和报名
	r.POST("/activities", handler.ActivityList)
	r.POST("/activities/signup", handler.ActivitySignup)
	r.POST("/activities/cancel", handler.ActivityCancel)
	r.GET("/activities/:id", handler.ActivityDetail)
	r.POST("/activities/my", handler.MyActivities)

	// 组织端 - 活动管理
	r.POST("/activities/create", handler.CreateActivity)
	r.PUT("/activities/:id", handler.UpdateActivity)
	r.DELETE("/activities/:id", handler.DeleteActivity)
	r.POST("/activities/:id/cancel", handler.CancelActivity)
}
