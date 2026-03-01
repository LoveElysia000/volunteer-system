package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

// RegisterNotificationRouter 注册通知路由。
func RegisterNotificationRouter(r *route.RouterGroup) {
	r.GET("/notifications", handler.ListNotifications)
	r.POST("/notifications/read", handler.MarkNotificationsRead)
}
