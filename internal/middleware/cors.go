package middleware

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

// CORS 设置跨域请求头
func CORS() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		// 处理预检请求
		if string(c.Method()) == "OPTIONS" {
			c.SetStatusCode(204)
			return
		}

		c.Next(ctx)
	}
}
