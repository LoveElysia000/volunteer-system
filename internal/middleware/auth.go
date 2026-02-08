package middleware

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"volunteer-system/internal/response"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/golang-jwt/jwt/v5"
)

// 上下文中的用户信息键
const (
	UserIDKey    = "user_id"
	TokenTypeKey = "token_type"
	DeviceIDKey  = "device_id"
)

// Auth JWT认证中间件
func Auth() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		authHeader := string(c.GetHeader("Authorization"))
		if authHeader == "" {
			response.FailWithCode(c, consts.StatusUnauthorized, errors.New("未提供认证令牌"))
			c.Abort()
			return
		}

		tokenParts := strings.Fields(authHeader)
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			response.FailWithCode(c, consts.StatusUnauthorized, errors.New("认证令牌格式错误"))
			c.Abort()
			return
		}

		tokenString := tokenParts[1]
		if tokenString == "" {
			response.FailWithCode(c, consts.StatusUnauthorized, errors.New("认证令牌为空"))
			c.Abort()
			return
		}

		jwtManager := util.GetJWTManager()
		claims, err := jwtManager.ValidateAccessToken(tokenString)
		if err != nil {
			errMsg := "认证令牌无效"
			if errors.Is(err, jwt.ErrTokenExpired) {
				errMsg = "认证令牌已过期，请刷新"
			}
			response.FailWithCode(c, consts.StatusUnauthorized, errors.New(errMsg))
			c.Abort()
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(TokenTypeKey, claims.TokenType)
		c.Set(DeviceIDKey, claims.DeviceID)

		c.Next(ctx)
	}
}

// Optional 可选认证中间件（认证失败不中断）
func Optional() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		authHeader := string(c.GetHeader("Authorization"))
		if authHeader == "" {
			c.Next(ctx)
			return
		}

		tokenParts := strings.Fields(authHeader)
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" || tokenParts[1] == "" {
			c.Next(ctx)
			return
		}

		tokenString := tokenParts[1]
		jwtManager := util.GetJWTManager()
		claims, err := jwtManager.ValidateAccessToken(tokenString)
		if err != nil {
			c.Next(ctx)
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(TokenTypeKey, claims.TokenType)
		c.Set(DeviceIDKey, claims.DeviceID)

		c.Next(ctx)
	}
}

// GetUserID 从上下文获取用户ID
func GetUserID(c *app.RequestContext) (string, error) {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return "", errors.New("用户ID未找到")
	}
	userIDStr, ok := userID.(string)
	if !ok {
		return "", errors.New("用户ID类型错误")
	}
	return userIDStr, nil
}

// GetUserIDInt 从上下文获取用户ID（int64）
func GetUserIDInt(c *app.RequestContext) (int64, error) {
	userID, err := GetUserID(c)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(userID, 10, 64)
}

// GetDeviceID 从上下文获取设备ID
func GetDeviceID(c *app.RequestContext) (string, error) {
	deviceID, exists := c.Get(DeviceIDKey)
	if !exists {
		return "", errors.New("设备ID未找到")
	}
	deviceIDStr, ok := deviceID.(string)
	if !ok {
		return "", errors.New("设备ID类型错误")
	}
	return deviceIDStr, nil
}
