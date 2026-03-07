package handler

import (
	"context"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// ListNotifications 获取通知列表。
func ListNotifications(ctx context.Context, c *app.RequestContext) {
	var req api.NotificationListRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewNotificationService(ctx, c).ListNotifications(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

// MarkNotificationsRead 批量标记已读。
func MarkNotificationsRead(ctx context.Context, c *app.RequestContext) {
	var req api.NotificationReadRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewNotificationService(ctx, c).MarkNotificationsRead(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}
