package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

func RegisterUploadRouter(r *route.RouterGroup) {
	r.POST("/upload", handler.Upload)
}
