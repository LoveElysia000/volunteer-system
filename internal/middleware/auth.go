package middleware

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"volunteer-system/internal/response"
	"volunteer-system/pkg/logger"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/golang-jwt/jwt/v5"
)

// 上下文中的用户信息键
const (
	AccountIDKey = "account_id"
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

		c.Set(AccountIDKey, claims.GetAccountID())
		c.Set(TokenTypeKey, claims.TokenType)
		c.Set(DeviceIDKey, claims.DeviceID)
		logger.SetCurrentRequestField("account_id", claims.GetAccountID())

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

		c.Set(AccountIDKey, claims.GetAccountID())
		c.Set(TokenTypeKey, claims.TokenType)
		c.Set(DeviceIDKey, claims.DeviceID)
		logger.SetCurrentRequestField("account_id", claims.GetAccountID())

		c.Next(ctx)
	}
}

// GetAccountID 从上下文获取账户ID
func GetAccountID(c *app.RequestContext) (string, error) {
	accountID, exists := c.Get(AccountIDKey)
	if !exists {
		return "", errors.New("账户ID未找到")
	}
	accountIDStr, ok := accountID.(string)
	if !ok {
		return "", errors.New("账户ID类型错误")
	}
	return accountIDStr, nil
}

// GetAccountIDInt 从上下文获取账户ID（int64）
func GetAccountIDInt(c *app.RequestContext) (int64, error) {
	accountID, err := GetAccountID(c)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(accountID, 10, 64)
}

// GetUserID keeps backward compatibility while the codebase migrates to account terminology.
func GetUserID(c *app.RequestContext) (string, error) {
	return GetAccountID(c)
}

// GetUserIDInt keeps backward compatibility while the codebase migrates to account terminology.
func GetUserIDInt(c *app.RequestContext) (int64, error) {
	return GetAccountIDInt(c)
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
