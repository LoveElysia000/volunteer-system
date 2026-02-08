package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

func RegisterLoginRouter(r *route.RouterGroup) {
	r.POST("/login", handler.UserLogin)
	r.POST("/logout", handler.UserLogout)
	r.POST("/refresh", handler.RefreshToken)
}
