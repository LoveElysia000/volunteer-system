package handler

import (
	"context"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
)

func VolunteerJoinOrganization(ctx context.Context, c *app.RequestContext) {
	var req api.VolunteerJoinRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewMembershipService(ctx, c).VolunteerJoinOrganization(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func VolunteerLeaveOrganization(ctx context.Context, c *app.RequestContext) {
	var req api.VolunteerLeaveRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewMembershipService(ctx, c).VolunteerLeaveOrganization(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func GetOrganizationMembers(ctx context.Context, c *app.RequestContext) {
	var req api.OrganizationMembersRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewMembershipService(ctx, c).GetOrganizationMembers(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func GetVolunteerOrganizations(ctx context.Context, c *app.RequestContext) {
	var req api.VolunteerOrganizationsRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewMembershipService(ctx, c).GetVolunteerOrganizations(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func UpdateMemberStatus(ctx context.Context, c *app.RequestContext) {
	var req api.MemberStatusUpdateRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewMembershipService(ctx, c).UpdateMemberStatus(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}

func MembershipStats(ctx context.Context, c *app.RequestContext) {
	var req api.MembershipStatsRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewMembershipService(ctx, c).MembershipStats(&req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, data)
}
