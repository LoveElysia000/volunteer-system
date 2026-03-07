package handler

import (
	"context"
	"volunteer-system/internal/api"
	"volunteer-system/internal/response"
	"volunteer-system/internal/service"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// CreateAssistantSession 创建 AI 会话
func CreateAssistantSession(ctx context.Context, c *app.RequestContext) {
	var req api.AssistantCreateSessionRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewAssistantService(ctx, c).CreateSession(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

// AssistantChat AI 对话
func AssistantChat(ctx context.Context, c *app.RequestContext) {
	var req api.AssistantChatRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewAssistantService(ctx, c).Chat(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

// AssistantSessionMessages 会话历史消息
func AssistantSessionMessages(ctx context.Context, c *app.RequestContext) {
	var req api.AssistantSessionMessagesRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewAssistantService(ctx, c).GetSessionMessages(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}

// AssistantActivityDraftAction 活动草案快捷入口
func AssistantActivityDraftAction(ctx context.Context, c *app.RequestContext) {
	var req api.AssistantActivityDraftActionRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}
	data, err := service.NewAssistantService(ctx, c).ActivityDraftAction(&req)
	if err != nil {
		response.FailWithCode(c, consts.StatusInternalServerError, err)
		return
	}
	response.Success(c, data)
}
