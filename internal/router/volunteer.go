package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

func RegisterVolunteerRouter(r *route.RouterGroup) {
	r.POST("/volunteers/list", handler.VolunteerList)
	r.GET("/volunteers/:volunteerId", handler.VolunteerDetail)
	r.GET("/me/profile", handler.MyProfile)
	r.GET("/volunteers/home/summary", handler.VolunteerHomeSummary)
	r.PUT("/me/account", handler.VolunteerAccountUpdate)
	r.PUT("/me/volunteer-profile", handler.VolunteerUpdate)
	r.POST("/volunteers/real-name/submit", handler.VolunteerRealNameSubmit)
}
