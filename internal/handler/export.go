package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// ExportVolunteers 导出志愿者
func ExportVolunteers(ctx context.Context, c *app.RequestContext) {
	var req api.ExportVolunteersRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}

	file, err := service.NewExportService(ctx, c).ExportVolunteers(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", url.PathEscape(file.FileName)))
	c.Header("Content-Type", file.ContentType)
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Access-Control-Expose-Headers", "Content-Disposition")
	c.Response.SetStatusCode(http.StatusOK)
	c.Response.SetBody(file.Content)
}

// ExportActivities 导出活动
func ExportActivities(ctx context.Context, c *app.RequestContext) {
	var req api.ExportActivitiesRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}

	file, err := service.NewExportService(ctx, c).ExportActivities(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", url.PathEscape(file.FileName)))
	c.Header("Content-Type", file.ContentType)
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Access-Control-Expose-Headers", "Content-Disposition")
	c.Response.SetStatusCode(http.StatusOK)
	c.Response.SetBody(file.Content)
}

// ExportOpsReport 导出运营模板报表
func ExportOpsReport(ctx context.Context, c *app.RequestContext) {
	var req api.ExportOpsReportRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}

	file, err := service.NewExportService(ctx, c).ExportOpsReport(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", url.PathEscape(file.FileName)))
	c.Header("Content-Type", file.ContentType)
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Access-Control-Expose-Headers", "Content-Disposition")
	c.Response.SetStatusCode(http.StatusOK)
	c.Response.SetBody(file.Content)
}
