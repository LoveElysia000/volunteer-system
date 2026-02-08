package handler

import (
	"context"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

// ActivityList 获取活动列表
func ActivityList(ctx context.Context, c *app.RequestContext) {
	var req api.ActivityListRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewActivityService(ctx, c).ActivityList(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

// ActivitySignup 活动报名
func ActivitySignup(ctx context.Context, c *app.RequestContext) {
	var req api.ActivitySignupRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewActivityService(ctx, c).ActivitySignup(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

// ActivityCancel 取消报名
func ActivityCancel(ctx context.Context, c *app.RequestContext) {
	var req api.ActivityCancelRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewActivityService(ctx, c).ActivityCancel(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

// ActivityDetail 获取活动详情
func ActivityDetail(ctx context.Context, c *app.RequestContext) {
	var req api.ActivityDetailRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewActivityService(ctx, c).ActivityDetail(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

// MyActivities 获取我的活动列表
func MyActivities(ctx context.Context, c *app.RequestContext) {
	var req api.MyActivitiesRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewActivityService(ctx, c).MyActivities(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

// ========== 组织端活动管理 ==========

// CreateActivity 创建活动
func CreateActivity(ctx context.Context, c *app.RequestContext) {
	var req api.CreateActivityRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewActivityService(ctx, c).CreateActivity(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

// UpdateActivity 更新活动
func UpdateActivity(ctx context.Context, c *app.RequestContext) {
	var req api.UpdateActivityRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewActivityService(ctx, c).UpdateActivity(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

// DeleteActivity 删除活动
func DeleteActivity(ctx context.Context, c *app.RequestContext) {
	var req api.DeleteActivityRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewActivityService(ctx, c).DeleteActivity(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

// CancelActivity 取消活动
func CancelActivity(ctx context.Context, c *app.RequestContext) {
	var req api.CancelActivityRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewActivityService(ctx, c).CancelActivity(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}