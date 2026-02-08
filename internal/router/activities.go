package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

// RegisterActivityRouter 注册活动相关路由
func RegisterActivityRouter(r *route.RouterGroup) {
	// 志愿者端 - 活动浏览和报名
	r.POST("/api/activities", handler.ActivityList)
	r.POST("/api/activities/signup", handler.ActivitySignup)
	r.POST("/api/activities/cancel", handler.ActivityCancel)
	r.GET("/api/activities/:id", handler.ActivityDetail)
	r.POST("/api/activities/my", handler.MyActivities)

	// 组织端 - 活动管理
	r.POST("/api/activities/create", handler.CreateActivity)
	r.PUT("/api/activities/:id", handler.UpdateActivity)
	r.DELETE("/api/activities/:id", handler.DeleteActivity)
	r.POST("/api/activities/:id/cancel", handler.CancelActivity)
}
