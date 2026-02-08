package handler

import (
	"context"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

// UserRegister 统一注册接口
func UserRegister(ctx context.Context, c *app.RequestContext) {
	var req api.RegisterRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, response.ErrInvalidParams.WithDetails(err.Error()))
		return
	}

	// 根据注册类型调用不同的注册服务
	registerType := model.GetRegisterTypeCode(req.RegisterType)

	var resp *api.RegisterResponse
	var err error
	switch registerType {
	case model.RegisterTypeVolunteerCode:
		resp, err = service.NewRegisterService(ctx, c).RegisterVolunteer(&req)
	case model.RegisterTypeOrganizationCode:
		resp, err = service.NewRegisterService(ctx, c).RegisterOrganization(&req)
	default:
		response.Error(c, response.ErrInvalidParams.WithDetails("不支持的注册类型"))
		return
	}

	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, resp)
}
