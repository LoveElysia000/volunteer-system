package handler

import (
	"context"
	"errors"
	"fmt"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"
	"volunteer-system/pkg/auth"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/golang-jwt/jwt/v5"
)

func UserLogin(ctx context.Context, c *app.RequestContext) {
	var req api.LoginRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewLoginService(ctx, c).Login(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
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
		code, mappedErr := mapLogoutError(err)
		response.FailWithCode(c, code, mappedErr)
		return
	}
	response.Success(c, data)
}

func mapLogoutError(err error) (int, error) {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired), errors.Is(err, auth.ErrInvalidToken), errors.Is(err, auth.ErrInvalidType):
		return consts.StatusUnauthorized, fmt.Errorf("无效的刷新令牌")
	default:
		return consts.StatusInternalServerError, err
	}
}

func RefreshToken(ctx context.Context, c *app.RequestContext) {
	var req api.RefreshTokenRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewLoginService(ctx, c).RefreshToken(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}
