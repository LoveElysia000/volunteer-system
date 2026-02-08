package router

import (
	"volunteer-system/internal/handler"
	"volunteer-system/internal/middleware"

	"github.com/cloudwego/hertz/pkg/app/server"
)

func RegisterRouter(r *server.Hertz) {

	// 全局中间件
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())

	//分组
	api := r.Group("/api")

	// 注册统一注册路由（无需认证）
	api.POST("/register", handler.UserRegister)

	// 注册登录路由（无需认证?
	RegisterLoginRouter(api)

	// 创建需要认证的路由组
	authApi := api.Group("", middleware.Auth())

	// 注册志愿者功能路由（需要认证）
	RegisterVolunteerRouter(authApi)
	// 注册组织管理员功能路由（需要认证）
	RegisterOrganizationRouter(authApi)
	RegisterMembershipRouter(authApi)
	RegisterAuditRouter(authApi)
	// 注册活动功能路由（需要认证）
	RegisterActivityRouter(authApi)

}
