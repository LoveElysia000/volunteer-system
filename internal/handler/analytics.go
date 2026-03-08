package handler

import (
	"context"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func OrgFunnelSummary(ctx context.Context, c *app.RequestContext) {
	var req api.OrgFunnelSummaryRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}

	data, err := service.NewAnalyticsService(ctx, c).OrgFunnelSummary(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}
