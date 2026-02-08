package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

func RegisterVolunteerRouter(r *route.RouterGroup) {
	r.POST("/volunteers/list", handler.VolunteerList)
	r.GET("/volunteers/detail/:volunteerId", handler.VolunteerDetail)
	r.PUT("/volunteers/:volunteerId", handler.VolunteerUpdate)
}
