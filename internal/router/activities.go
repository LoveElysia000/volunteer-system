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
	r.POST("/activities/checkin", handler.ActivityCheckIn)
	r.POST("/activities/checkout", handler.ActivityCheckOut)

	// 组织端 - 活动管理
	r.POST("/activities/create", handler.CreateActivity)
	r.PUT("/activities/:id", handler.UpdateActivity)
	r.DELETE("/activities/:id", handler.DeleteActivity)
	r.POST("/activities/:id/cancel", handler.CancelActivity)
	r.POST("/activities/:id/finish", handler.FinishActivity)
	r.POST("/activities/supplement-attendance", handler.ActivitySupplementAttendance)
}
