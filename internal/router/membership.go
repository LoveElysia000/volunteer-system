package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

// RegisterMembershipRouter registers membership-related routes.
func RegisterMembershipRouter(r *route.RouterGroup) {
	r.POST("/memberships/join", handler.VolunteerJoinOrganization)
	r.POST("/memberships/leave", handler.VolunteerLeaveOrganization)
	r.GET("/organizations/:organizationId/members", handler.GetOrganizationMembers)
	r.GET("/volunteers/:volunteerId/organizations", handler.GetVolunteerOrganizations)
	r.POST("/memberships/status/update", handler.UpdateMemberStatus)
	r.GET("/memberships/stats", handler.MembershipStats)
}
