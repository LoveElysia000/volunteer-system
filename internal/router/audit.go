package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

// RegisterAuditRouter registers audit related routes.
func RegisterAuditRouter(r *route.RouterGroup) {
	r.POST("/audits/pending", handler.PendingAuditList)
	r.POST("/audits/approval", handler.AuditApproval)
	r.POST("/audits/rejection", handler.AuditRejection)
	r.GET("/audits/records/:id", handler.AuditRecordDetail)
}
