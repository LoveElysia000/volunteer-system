package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

func RegisterRegisterRouter(r *route.RouterGroup) {
	// 注册接口
	r.POST("/api/register", handler.UserRegister)
}
