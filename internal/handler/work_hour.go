package handler

import (
	"context"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

// WorkHourLogList 工时流水查询
func WorkHourLogList(ctx context.Context, c *app.RequestContext) {
	var req api.WorkHourLogListRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewWorkHourService(ctx, c).WorkHourLogList(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

// VoidWorkHour 工时作废
func VoidWorkHour(ctx context.Context, c *app.RequestContext) {
	var req api.VoidWorkHourRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewWorkHourService(ctx, c).VoidWorkHour(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

// RecalculateWorkHour 工时重算
func RecalculateWorkHour(ctx context.Context, c *app.RequestContext) {
	var req api.RecalculateWorkHourRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewWorkHourService(ctx, c).RecalculateWorkHour(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}
