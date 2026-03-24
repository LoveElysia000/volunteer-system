package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"volunteer-system/pkg/logger"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

const requestIDKey = "request_id"

// RequestLog injects request-scoped fields into logs and emits one access log per request.
func RequestLog() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		requestID := resolveRequestID(c)
		c.Set(requestIDKey, requestID)
		c.Response.Header.Set("X-Request-ID", requestID)

		path := string(c.Request.URI().Path())
		fields := map[string]string{
			"request_id": requestID,
			"method":     string(c.Method()),
			"path":       path,
		}
		if clientIP := strings.TrimSpace(c.ClientIP()); clientIP != "" {
			fields["client_ip"] = clientIP
		}

		release := logger.BindCurrentRequest(fields)
		start := time.Now()
		defer release()

		c.Next(ctx)

		if route := strings.TrimSpace(c.FullPath()); route != "" {
			logger.SetCurrentRequestField("route", route)
		}
		if userID, err := GetUserID(c); err == nil && strings.TrimSpace(userID) != "" {
			logger.SetCurrentRequestField("user_id", userID)
		}
		logger.SetCurrentRequestField("status", strconv.Itoa(c.Response.StatusCode()))
		logger.SetCurrentRequestField("latency_ms", strconv.FormatInt(time.Since(start).Milliseconds(), 10))
		logger.GetLogger().InfoAttrs(
			"http request completed",
			"request_headers", collectHeaders(c),
			"query_params", collectQueryParams(c),
			"request_body", decodeBodyValue(c.Request.Body()),
			"response_body", decodeBodyValue(c.Response.Body()),
			"error", deriveErrorValue(c.Response.Body()),
		)
	}
}

func resolveRequestID(c *app.RequestContext) string {
	if c == nil {
		return uuid.NewString()
	}
	for _, header := range []string{"X-Request-Id", "X-Request-ID"} {
		if rid := strings.TrimSpace(string(c.GetHeader(header))); rid != "" {
			return rid
		}
	}
	return uuid.NewString()
}

func collectHeaders(c *app.RequestContext) map[string][]string {
	headers := make(map[string][]string)
	if c == nil {
		return headers
	}

	c.Request.Header.VisitAll(func(key, value []byte) {
		k := string(key)
		headers[k] = append(headers[k], string(value))
	})
	return headers
}

func collectQueryParams(c *app.RequestContext) map[string][]string {
	params := make(map[string][]string)
	if c == nil {
		return params
	}

	c.Request.URI().QueryArgs().VisitAll(func(key, value []byte) {
		k := string(key)
		params[k] = append(params[k], string(value))
	})
	return params
}

func decodeBodyValue(body []byte) any {
	raw := bytes.TrimSpace(body)
	if len(raw) == 0 {
		return nil
	}

	var payload any
	if json.Unmarshal(raw, &payload) == nil {
		return payload
	}
	return string(raw)
}

func deriveErrorValue(body []byte) any {
	payload, ok := decodeBodyValue(body).(map[string]any)
	if !ok {
		return nil
	}

	codeValue, hasCode := payload["code"]
	msgValue, hasMsg := payload["msg"]
	if !hasCode || !hasMsg {
		return nil
	}

	code, ok := toInt64(codeValue)
	if !ok || code < 400 {
		return nil
	}
	msg, ok := msgValue.(string)
	if !ok || strings.TrimSpace(msg) == "" {
		return nil
	}
	return msg
}

func toInt64(v any) (int64, bool) {
	switch value := v.(type) {
	case int:
		return int64(value), true
	case int32:
		return int64(value), true
	case int64:
		return value, true
	case float64:
		return int64(value), true
	case json.Number:
		n, err := value.Int64()
		return n, err == nil
	default:
		return 0, false
	}
}
