package router

import (
	"volunteer-system/internal/handler"

	"github.com/cloudwego/hertz/pkg/route"
)

// RegisterAssistantRouter 注册 AI 助手路由
func RegisterAssistantRouter(r *route.RouterGroup) {
	assistant := r.Group("/assistant")
	assistant.POST("/sessions", handler.CreateAssistantSession)
	assistant.POST("/chat", handler.AssistantChat)
	assistant.GET("/sessions/:id/messages", handler.AssistantSessionMessages)
	assistant.POST("/actions/activity-draft", handler.AssistantActivityDraftAction)
}
