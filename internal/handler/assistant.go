package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
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

// AssistantChatStream AI 流式对话（SSE）
func AssistantChatStream(ctx context.Context, c *app.RequestContext) {
	var req api.AssistantChatRequest
	if err := c.BindAndValidate(&req); err != nil {
		response.Fail(c, err)
		return
	}

	req.Stream = false
	streamCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	c.Response.SetStatusCode(consts.StatusOK)
	c.SetContentType("text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	pr, pw := io.Pipe()
	c.SetBodyStream(pr, -1)

	go func() {
		defer pw.Close()

		svc := service.NewAssistantService(streamCtx, c)
		resultCh := make(chan struct {
			resp *api.AssistantChatResponse
			err  error
		}, 1)
		go func() {
			resp, err := svc.Chat(&req)
			resultCh <- struct {
				resp *api.AssistantChatResponse
				err  error
			}{resp: resp, err: err}
		}()

		heartbeatTicker := time.NewTicker(15 * time.Second)
		defer heartbeatTicker.Stop()

		if err := writeSSEEvent(pw, "start", map[string]any{"session_id": req.SessionId}); err != nil {
			return
		}
		for {
			select {
			case <-streamCtx.Done():
				_ = writeSSEEvent(pw, "error", map[string]any{"message": "stream timeout or canceled"})
				_ = writeSSEEvent(pw, "done", map[string]any{"finish_reason": "error"})
				return
			case <-c.Finished():
				return
			case <-heartbeatTicker.C:
				if _, err := io.WriteString(pw, ": heartbeat\n\n"); err != nil {
					return
				}
			case result := <-resultCh:
				if result.err != nil {
					_ = writeSSEEvent(pw, "error", map[string]any{"message": result.err.Error()})
					_ = writeSSEEvent(pw, "done", map[string]any{"finish_reason": "error"})
					return
				}
				for _, event := range service.BuildAssistantStreamEvents(req.SessionId, result.resp) {
					if err := writeSSEEvent(pw, event.Event, event.Data); err != nil {
						return
					}
				}
				return
			}
		}
	}()
}

func writeSSEEvent(w io.Writer, event string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "data: %s\n\n", string(encoded)); err != nil {
		return err
	}
	return nil
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
