package handler

import (
	"context"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

// VolunteerRegister 志愿者注册接口
func VolunteerRegister(ctx context.Context, c *app.RequestContext) {
	var req api.VolunteerRegisterRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, response.ErrInvalidParams.WithDetails(err.Error()))
		return
	}

	resp, err := service.NewRegisterService(ctx, c).RegisterVolunteer(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, resp)
}

// OrganizationRegister 组织管理者注册接口
func OrganizationRegister(ctx context.Context, c *app.RequestContext) {
	var req api.OrganizationRegisterRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, response.ErrInvalidParams.WithDetails(err.Error()))
		return
	}

	resp, err := service.NewRegisterService(ctx, c).RegisterOrganization(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}

	response.Success(c, resp)
}
