package handler

import (
	"context"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// PendingAuditList 处理统一待审核列表查询请求。
func PendingAuditList(ctx context.Context, c *app.RequestContext) {
	var req api.PendingAuditListRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}

	data, err := service.NewAuditService(ctx, c).PendingAuditList(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

// AuditApproval 处理审核通过请求。
func AuditApproval(ctx context.Context, c *app.RequestContext) {
	var req api.AuditApprovalRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}

	data, err := service.NewAuditService(ctx, c).AuditApproval(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

// AuditRejection 处理审核驳回请求。
func AuditRejection(ctx context.Context, c *app.RequestContext) {
	var req api.AuditRejectionRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}

	data, err := service.NewAuditService(ctx, c).AuditRejection(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

// AuditBatchDecision 处理批量审核决策请求。
func AuditBatchDecision(ctx context.Context, c *app.RequestContext) {
	var req api.AuditBatchDecisionRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}

	data, err := service.NewAuditService(ctx, c).AuditBatchDecision(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

// AuditRecordDetail 处理审核记录详情查询请求。
func AuditRecordDetail(ctx context.Context, c *app.RequestContext) {
	var req api.AuditRecordDetailRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}

	data, err := service.NewAuditService(ctx, c).AuditRecordDetail(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}
