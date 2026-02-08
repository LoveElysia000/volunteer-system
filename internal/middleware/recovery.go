package middleware

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"

	"volunteer-system/internal/response"
	"volunteer-system/pkg/logger"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// Recovery 恢复中间件，防止panic导致程序崩溃
func Recovery() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		defer func() {
			if err := recover(); err != nil {
				// 获取panic堆栈信息
				stack := debug.Stack()

				// 记录panic信息
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				logMsg := fmt.Sprintf("[PANIC] %v\nTime: %s\nStack: %s", err, timestamp, string(stack))

				if log := logger.GetLogger(); log != nil {
					log.Error("%s", logMsg)
				} else {
					fmt.Printf("%s\n", logMsg)
				}

				// 返回错误响应给客户端
				response.FailWithCode(c, consts.StatusInternalServerError,
					fmt.Errorf("服务器内部错误"))

				// 确保请求被中止
				c.Abort()
			}
		}()

		c.Next(ctx)
	}
}