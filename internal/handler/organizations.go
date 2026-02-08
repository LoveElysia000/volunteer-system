package handler

import (
	"context"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

func OrganizationList(ctx context.Context, c *app.RequestContext) {
	var req api.OrganizationListRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewOrganizationService(ctx, c).OrganizationList(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func OrganizationDetail(ctx context.Context, c *app.RequestContext) {
	var req api.OrganizationDetailRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewOrganizationService(ctx, c).OrganizationDetail(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func CreateOrganization(ctx context.Context, c *app.RequestContext) {
	var req api.OrganizationCreateRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewOrganizationService(ctx, c).CreateOrganization(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func UpdateOrganization(ctx context.Context, c *app.RequestContext) {
	var req api.OrganizationUpdateRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewOrganizationService(ctx, c).UpdateOrganization(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func DeleteOrganization(ctx context.Context, c *app.RequestContext) {
	var req api.DeleteOrganizationRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewOrganizationService(ctx, c).DeleteOrganization(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func DisableOrganization(ctx context.Context, c *app.RequestContext) {
	var req api.DisableOrganizationRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewOrganizationService(ctx, c).DisableOrganization(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func EnableOrganization(ctx context.Context, c *app.RequestContext) {
	var req api.EnableOrganizationRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewOrganizationService(ctx, c).EnableOrganization(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func SearchOrganizations(ctx context.Context, c *app.RequestContext) {
	var req api.OrganizationSearchRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewOrganizationService(ctx, c).SearchOrganizations(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func BulkDeleteOrganizations(ctx context.Context, c *app.RequestContext) {
	var req api.BulkDeleteOrganizationRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewOrganizationService(ctx, c).BulkDeleteOrganizations(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}