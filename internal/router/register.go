package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

func RegisterRegisterRouter(r *route.RouterGroup) {
	// 志愿者注册接口
	r.POST("/volunteer/register", handler.VolunteerRegister)
	// 组织管理者注册接口
	r.POST("/organization/register", handler.OrganizationRegister)
}
