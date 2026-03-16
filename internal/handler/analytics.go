package handler

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func OrgFunnelSummary(ctx context.Context, c *app.RequestContext) {
	req, err := buildAnalyticsQueryRequest(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	data, err := service.NewAnalyticsService(ctx, c).OrgFunnelSummary(req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

func OpsDashboardSummary(ctx context.Context, c *app.RequestContext) {
	baseReq, err := buildAnalyticsQueryRequest(c)
	if err != nil {
		response.Fail(c, err)
		return
	}

	data, err := service.NewAnalyticsService(ctx, c).OpsDashboardSummary(&api.OpsDashboardSummaryRequest{
		OrgId: baseReq.OrgId,
		Start: baseReq.Start,
		End:   baseReq.End,
	})
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

func buildAnalyticsQueryRequest(c *app.RequestContext) (*api.OrgFunnelSummaryRequest, error) {
	if c == nil {
		return nil, errors.New("请求上下文不能为空")
	}

	orgIDText := strings.TrimSpace(c.Query("orgId"))
	if orgIDText == "" {
		return nil, errors.New("组织ID不能为空")
	}
	orgID, err := strconv.ParseInt(orgIDText, 10, 64)
	if err != nil || orgID <= 0 {
		return nil, errors.New("组织ID不能为空")
	}

	return &api.OrgFunnelSummaryRequest{
		OrgId: orgID,
		Start: strings.TrimSpace(c.Query("start")),
		End:   strings.TrimSpace(c.Query("end")),
	}, nil
}
