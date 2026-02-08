package handler

import (
	"context"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

func UserLogin(ctx context.Context, c *app.RequestContext) {
	var req api.LoginRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewLoginService(ctx, c).Login(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func UserLogout(ctx context.Context, c *app.RequestContext) {
	var req api.LogoutRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewLoginService(ctx, c).Logout(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func RefreshToken(ctx context.Context, c *app.RequestContext) {
	var req api.RefreshTokenRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewLoginService(ctx, c).RefreshToken(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}
