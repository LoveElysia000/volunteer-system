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

// ActivityCheckIn 活动签到（志愿者侧）
func ActivityCheckIn(ctx context.Context, c *app.RequestContext) {
	var req api.ActivityCheckInRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewActivityService(ctx, c).ActivityCheckIn(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

// ActivityCheckOut 活动签退（志愿者侧）
func ActivityCheckOut(ctx context.Context, c *app.RequestContext) {
	var req api.ActivityCheckOutRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewActivityService(ctx, c).ActivityCheckOut(&req)
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

// FinishActivity 完结活动
func FinishActivity(ctx context.Context, c *app.RequestContext) {
	var req api.FinishActivityRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewActivityService(ctx, c).FinishActivity(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

// ActivitySupplementAttendance 活动签到签退补录（组织侧）
func ActivitySupplementAttendance(ctx context.Context, c *app.RequestContext) {
	var req api.ActivitySupplementAttendanceRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewActivityService(ctx, c).ActivitySupplementAttendance(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}
