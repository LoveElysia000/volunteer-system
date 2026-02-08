package handler

import (
	"context"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

func PendingVolunteerJoinOrgAuditList(ctx context.Context, c *app.RequestContext) {
	var req api.PendingVolunteerJoinOrgAuditListRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}

	data, err := service.NewAuditService(ctx, c).VolunteerJoinOrgAuditList(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func AuditApproval(ctx context.Context, c *app.RequestContext) {
	var req api.AuditApprovalRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}

	data, err := service.NewAuditService(ctx, c).AuditApproval(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func AuditRejection(ctx context.Context, c *app.RequestContext) {
	var req api.AuditRejectionRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}

	data, err := service.NewAuditService(ctx, c).AuditRejection(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func AuditRecordDetail(ctx context.Context, c *app.RequestContext) {
	var req api.AuditRecordDetailRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}

	data, err := service.NewAuditService(ctx, c).AuditRecordDetail(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}
