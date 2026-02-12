package handler

import (
	"context"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

func VolunteerList(ctx context.Context, c *app.RequestContext) {
	var req api.VolunteerListRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewVolunteerService(ctx, c).VolunteerList(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func VolunteerDetail(ctx context.Context, c *app.RequestContext) {
	var req api.VolunteerDetailRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewVolunteerService(ctx, c).VolunteerDetail(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func MyProfile(ctx context.Context, c *app.RequestContext) {
	var req api.MyProfileRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewVolunteerService(ctx, c).MyProfile(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func VolunteerUpdate(ctx context.Context, c *app.RequestContext) {
	var req api.VolunteerUpdateRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewVolunteerService(ctx, c).VolunteerUpdate(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}
