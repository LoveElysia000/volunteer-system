package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

func RegisterVolunteerRouter(r *route.RouterGroup) {
	r.POST("/volunteers/list", handler.VolunteerList)
	r.GET("/volunteers/detail/:id", handler.VolunteerDetail)
	r.GET("/volunteers/my/profile/:id", handler.MyProfile)
	r.GET("/volunteers/home/summary", handler.VolunteerHomeSummary)
	r.PUT("/volunteers/account", handler.VolunteerAccountUpdate)
	r.PUT("/volunteers/:id", handler.VolunteerUpdate)
	r.POST("/volunteers/real-name/submit", handler.VolunteerRealNameSubmit)
}
