package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrInvalidType  = errors.New("invalid token type")
)

const (
	AccessTokenTTL     = 1 * 24 * time.Hour
	RefreshTokenTTL    = 7 * 24 * time.Hour
	RefreshLimit       = 5
	RefreshLimitWindow = 5 * time.Minute
)

type Claims struct {
	AccountID string `json:"accountId"`
	TokenID   string `json:"jti"` // JWT ID
	TokenType string `json:"type"`
	DeviceID  string `json:"device"`
	jwt.RegisteredClaims
}

func (c *Claims) GetAccountID() string {
	if c == nil {
		return ""
	}
	if strings.TrimSpace(c.AccountID) != "" {
		return c.AccountID
	}
	return strings.TrimSpace(c.Subject)
}

type TokenPair struct {
	AccessToken    string
	RefreshToken   string
	RefreshTokenID string
	ExpiresIn      int64
}

// Manager JWT 管理器
type Manager struct {
	secretKey []byte
	redis     *redis.Client
}

// NewManager 创建 JWT 管理器
func NewManager(secretKey string, redis *redis.Client) *Manager {
	return &Manager{
		secretKey: []byte(secretKey),
		redis:     redis,
	}
}

// GenerateTokenPair 生成双 Token
func (m *Manager) GenerateTokenPair(accountID string, deviceID string, accessTokenTTL time.Duration, refreshTokenTTL time.Duration) (string, string, string, string, error) {
	// 生成 Access Token
	accessTokenID, accessToken, err := m.generateAccessToken(accountID, deviceID, accessTokenTTL)
	if err != nil {
		return "", "", "", "", err
	}

	// 生成 Refresh Token
	refreshTokenID, refreshToken, err := m.generateRefreshToken(accountID, deviceID, refreshTokenTTL)
	if err != nil {
		return "", "", "", "", err
	}

	return accessTokenID, accessToken, refreshTokenID, refreshToken, nil
}

// generateAccessToken 生成短期 Access Token
func (m *Manager) generateAccessToken(accountID string, deviceID string, ttl time.Duration) (string, string, error) {
	tokenID := uuid.New().String()
	now := time.Now()
	claims := Claims{
		AccountID: accountID,
		TokenID:   tokenID,
		TokenType: "access",
		DeviceID:  deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   accountID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", "", err
	}

	return tokenID, tokenStr, nil
}

// generateRefreshToken 生成长期 Refresh Token
func (m *Manager) generateRefreshToken(accountID string, deviceID string, ttl time.Duration) (string, string, error) {
	tokenID := uuid.New().String()
	now := time.Now()
	claims := Claims{
		AccountID: accountID,
		TokenID:   tokenID,
		TokenType: "refresh",
		DeviceID:  deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   accountID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", "", err
	}

	return tokenID, tokenStr, nil
}

// ValidateAccessToken 验证 Access Token
func (m *Manager) ValidateAccessToken(tokenString string) (*Claims, error) {
	return m.validateToken(tokenString, "access")
}

// ValidateRefreshToken 验证 Refresh Token
func (m *Manager) ValidateRefreshToken(tokenString string) (*Claims, error) {
	return m.validateToken(tokenString, "refresh")
}

// validateToken 验证 Token（通用方法）
func (m *Manager) validateToken(tokenString string, expectedType string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secretKey, nil
	})

	if err != nil {
		// 检查是否为token过期错误
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, jwt.ErrTokenExpired
		}
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	if claims.TokenType != expectedType {
		return nil, ErrInvalidType
	}

	return claims, nil
}

// GenerateTokenPairWithStorage 生成双Token（带 Redis 存储）
func (m *Manager) GenerateTokenPairWithStorage(ctx context.Context, accountID string, deviceID string, ipAddress string, userAgent string) (*TokenPair, error) {
	// 1. 调用原有 GenerateTokenPair 生成双 Token
	_, accessToken, refreshTokenID, refreshToken, err := m.GenerateTokenPair(accountID, deviceID, AccessTokenTTL, RefreshTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("generate token pair failed: %w", err)
	}

	// 2. 存储撤销状态到 Redis（使用 tokenID 作为 key）
	key := fmt.Sprintf("refresh:%s", refreshTokenID)
	if err := m.redis.Set(ctx, key, "valid", RefreshTokenTTL).Err(); err != nil {
		return nil, fmt.Errorf("store refresh token failed: %w", err)
	}

	// 3. 添加到用户 Token 列表（存储 tokenID）
	userTokensKey := fmt.Sprintf("user:tokens:%s", accountID)
	m.redis.Set(ctx, userTokensKey, refreshTokenID, RefreshTokenTTL)

	return &TokenPair{
		AccessToken:    accessToken,
		RefreshToken:   refreshToken,
		RefreshTokenID: refreshTokenID,
		ExpiresIn:      int64(AccessTokenTTL / time.Second),
	}, nil
}

// RefreshTokenWithStorage 刷新 Access Token（带 Redis 存储）
func (m *Manager) RefreshTokenWithStorage(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// 1. 验证 Refresh Token 并获取用户信息
	claims, err := m.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	accountID := claims.GetAccountID()
	deviceID := claims.DeviceID
	oldTokenID := claims.TokenID

	// 2. 检查 Refresh Token 是否在 Redis 中存在（未被撤销）
	key := fmt.Sprintf("refresh:%s", oldTokenID)
	exists, err := m.redis.Exists(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("check refresh token failed: %w", err)
	}
	if exists == 0 {
		return nil, fmt.Errorf("refresh token not found or revoked")
	}

	// 3. 检查刷新限流
	limitKey := fmt.Sprintf("refresh:limit:%s:%s", accountID, deviceID)
	count, _ := m.redis.Incr(ctx, limitKey).Result()
	if count == 1 {
		m.redis.Expire(ctx, limitKey, RefreshLimitWindow)
	}
	if count > RefreshLimit {
		return nil, fmt.Errorf("refresh rate limit exceeded")
	}

	// 4. 生成新的双 Token
	_, newAccessToken, newRefreshTokenID, newRefreshToken, err := m.GenerateTokenPair(
		accountID, deviceID, AccessTokenTTL, RefreshTokenTTL,
	)
	if err != nil {
		return nil, fmt.Errorf("generate new tokens failed: %w", err)
	}

	// 5. 存储新 Refresh Token（使用 tokenID 作为 key）
	newKey := fmt.Sprintf("refresh:%s", newRefreshTokenID)
	if err := m.redis.Set(ctx, newKey, "valid", RefreshTokenTTL).Err(); err != nil {
		return nil, fmt.Errorf("store new refresh token failed: %w", err)
	}

	// 6. 撤销旧的 Refresh Token（删除 Redis 中的记录）
	m.redis.Del(ctx, key)

	// 7. 更新用户 Token 列表（存储 tokenID）
	userTokensKey := fmt.Sprintf("user:tokens:%s", accountID)
	m.redis.Set(ctx, userTokensKey, newRefreshTokenID, RefreshTokenTTL)

	return &TokenPair{
		AccessToken:    newAccessToken,
		RefreshToken:   newRefreshToken,
		RefreshTokenID: newRefreshTokenID,
		ExpiresIn:      int64(AccessTokenTTL / time.Second),
	}, nil
}

// RevokeToken 撤销指定的 Token
func (m *Manager) RevokeToken(ctx context.Context, tokenID string, accountID string) error {
	// 1. 删除 Redis 中的 token 记录
	key := fmt.Sprintf("refresh:%s", tokenID)
	if err := m.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("delete token failed: %w", err)
	}

	// 2. 从用户 token 列表中移除
	userTokensKey := fmt.Sprintf("user:tokens:%s", accountID)
	m.redis.Del(ctx, userTokensKey)

	return nil
}
