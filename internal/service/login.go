package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type LoginService struct {
	Service
}

func NewLoginService(ctx context.Context, c *app.RequestContext) *LoginService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &LoginService{
		Service: Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
}

func (s *LoginService) Login(req *api.LoginRequest) (*api.LoginResponse, error) {
	var resp api.LoginResponse

	// 记录登录请求
	log.Info("用户登录请求: 登录类型=%s, 标识=%s, 身份类型=%s", req.LoginType, maskIdentifier(req.Identifier), req.Identity)

	// 1. 验证请求参数
	if err := s.validateLoginRequest(req); err != nil {
		log.Warn("登录请求验证失败: %v, 登录类型=%s, 标识=%s", err, req.LoginType, req.Identifier)
		resp.Success = false
		resp.Message = err.Error()
		return &resp, nil
	}

	// 2. 根据登录类型查找用户
	log.Debug("开始查找用户: 登录类型=%s", req.LoginType)
	user, err := s.findUserByIdentifier(s.repo.DB, req)
	if err != nil {
		log.Error("查找用户失败: %v, 登录类型=%s, 标识=%s", err, req.LoginType, req.Identifier)
		resp.Success = false
		resp.Message = "系统错误，请稍后重试"
		return &resp, err
	}

	if user == nil {
		log.Warn("用户不存在: 登录类型=%s, 标识=%s", req.LoginType, maskIdentifier(req.Identifier))
		resp.Success = false
		resp.Message = "用户不存在"
		return &resp, nil
	}
	log.Debug("用户查询成功: 用户ID=%d, 身份类型=%d", user.ID, user.IdentityType)

	// 3. 验证用户状态
	if user.Status != 1 {
		log.Warn("账号已被禁用: 用户ID=%d, 状态=%d", user.ID, user.Status)
		resp.Success = false
		resp.Message = "账号已被禁用"
		return &resp, nil
	}

	// 4. 验证密码
	if !util.CheckPassword(req.Password, user.Password) {
		log.Warn("密码错误: 用户ID=%d, 登录类型=%s, 标识=%s", user.ID, req.LoginType, maskIdentifier(req.Identifier))
		resp.Success = false
		resp.Message = "密码错误"
		return &resp, nil
	}

	// 5. 验证身份类型是否匹配
	expectedIdentityType := util.GetIdentityTypeFromString(req.Identity)
	if user.IdentityType != expectedIdentityType {
		log.Warn("身份类型不匹配: 用户ID=%d, 预期=%s, 实际=%d", user.ID, req.Identity, user.IdentityType)
		resp.Success = false
		resp.Message = "身份类型不匹配"
		return &resp, nil
	}

	// 6. 生成JWT令牌
	log.Debug("开始生成令牌: 用户ID=%d", user.ID)
	jwtManager := util.GetJWTManager()
	userID := fmt.Sprintf("%d", user.ID)

	// 获取设备信息（简化处理，实际可以从请求头获取）
	deviceID := "web" // 简化处理
	ipAddress := ""   // 可以从请求中获取
	userAgent := ""   // 可以从请求中获取

	tokenPair, err := jwtManager.GenerateTokenPairWithStorage(s.ctx, userID, deviceID, ipAddress, userAgent)
	if err != nil {
		log.Error("令牌生成失败: %v, 用户ID=%d", err, user.ID)
		resp.Success = false
		resp.Message = "令牌生成失败"
		return &resp, err
	}
	log.Debug("令牌生成成功: 用户ID=%d", user.ID)

	// 7. 更新最后登录时间
	if err := s.repo.UpdateLastLoginTime(s.repo.DB, user.ID); err != nil {
		log.Error("更新最后登录时间失败: %v, 用户ID=%d", err, user.ID)
	}

	// 8. 构建响应
	resp.Success = true
	resp.Message = "登录成功"
	resp.AccessToken = tokenPair.AccessToken
	resp.RefreshToken = tokenPair.RefreshToken
	resp.ExpiresAt = time.Now().Add(24 * time.Hour).Unix() // 24小时过期（与 AccessTokenTTL 一致）
	resp.UserInfo = util.ConvertSysAccountToUserInfo(user)

	log.Info("用户登录成功: 用户ID=%d, 邮箱=%s, 身份类型=%s", user.ID, user.Email, req.Identity)

	return &resp, nil
}

// validateLoginRequest 验证登录请求参数
func (s *LoginService) validateLoginRequest(req *api.LoginRequest) error {
	log.Debug("验证登录请求参数")

	if req.LoginType == "" {
		log.Debug("验证失败: 登录类型不能为空")
		return errors.New("登录类型不能为空")
	}
	if !util.ValidateLoginType(req.LoginType) {
		log.Warn("验证失败: 无效的登录类型 - %s", req.LoginType)
		return errors.New("无效的登录类型")
	}
	if req.Identifier == "" {
		log.Debug("验证失败: 登录标识不能为空")
		return errors.New("登录标识不能为空")
	}
	if req.Password == "" {
		log.Debug("验证失败: 密码不能为空")
		return errors.New("密码不能为空")
	}
	if req.Identity == "" {
		log.Debug("验证失败: 身份类型不能为空")
		return errors.New("身份类型不能为空")
	}
	if !util.ValidateIdentity(req.Identity) {
		log.Warn("验证失败: 无效的身份类型 - %s", req.Identity)
		return errors.New("无效的身份类型")
	}

	log.Debug("登录请求参数验证通过")
	return nil
}

// findUserByIdentifier 根据登录标识查找用户
func (s *LoginService) findUserByIdentifier(db *gorm.DB, req *api.LoginRequest) (*model.SysAccount, error) {
	switch req.LoginType {
	case "phone":
		// 手机号登录 - 生成哈希值查询
		log.Debug("手机号登录: 生成哈希值")
		mobileHash, err := util.HashSensitiveField(req.Identifier)
		if err != nil {
			log.Error("手机号哈希生成失败: %v, 手机号=%s", err, req.Identifier)
			return nil, errors.New("手机号处理失败")
		}
		log.Debug("手机号哈希已生成，开始查询用户")

		user, err := s.repo.FindByMobile(db, mobileHash)
		if err != nil {
			log.Debug("通过手机号查询用户失败: %v", err)
			return nil, errors.New("用户不存在")
		}
		return user, nil
	case "email":
		// 邮箱登录
		log.Debug("邮箱登录: 开始查询用户, 邮箱=%s", req.Identifier)
		user, err := s.repo.FindByEmail(db, req.Identifier)
		if err != nil {
			log.Debug("通过邮箱查询用户失败: %v", err)
			return nil, errors.New("用户不存在")
		}
		return user, nil
	default:
		log.Error("不支持的登录类型: %s", req.LoginType)
		return nil, errors.New("不支持的登录类型")
	}
}
func (s *LoginService) Logout(req *api.LogoutRequest) (*api.LogoutResponse, error) {
	var resp api.LogoutResponse

	log.Info("用户登出请求")

	// 1. 验证请求参数
	if req.Token == "" {
		log.Warn("登出失败: 令牌不能为空")
		resp.Success = false
		resp.Message = "令牌不能为空"
		return &resp, errors.New("令牌不能为空")
	}

	// 2. 验证 Refresh Token 并获取用户信息
	log.Debug("开始验证令牌")
	jwtManager := util.GetJWTManager()
	claims, err := jwtManager.ValidateRefreshToken(req.Token)
	if err != nil {
		log.Warn("令牌验证失败: %v", err)
		resp.Success = false
		resp.Message = "无效的令牌"
		if err == jwt.ErrTokenExpired {
			resp.Message = "令牌已过期"
		}
		return &resp, err
	}
	log.Debug("令牌验证成功: 用户ID=%s, TokenID=%s", claims.UserID, claims.TokenID)

	// 3. 撤销 token
	log.Debug("开始撤销令牌: 用户ID=%s, TokenID=%s", claims.UserID, claims.TokenID)
	if err := jwtManager.RevokeToken(s.ctx, claims.TokenID, claims.UserID); err != nil {
		log.Error("撤销令牌失败: %v, 用户ID=%s, TokenID=%s", err, claims.UserID, claims.TokenID)
		resp.Success = false
		resp.Message = "登出失败"
		return &resp, err
	}

	// 4. 返回成功响应
	log.Info("用户登出成功: 用户ID=%s", claims.UserID)
	resp.Success = true
	resp.Message = "登出成功"
	return &resp, nil
}

func (s *LoginService) RefreshToken(req *api.RefreshTokenRequest) (*api.RefreshTokenResponse, error) {
	var resp api.RefreshTokenResponse

	log.Info("刷新令牌请求")

	// 1. 验证请求参数
	if req.RefreshToken == "" {
		log.Warn("刷新令牌失败: 刷新令牌不能为空")
		resp.Success = false
		resp.Message = "刷新令牌不能为空"
		return &resp, errors.New("刷新令牌不能为空")
	}

	log.Debug("开始刷新令牌")
	jwtManager := util.GetJWTManager()

	// 2. 调用 JWT 管理器刷新令牌（内部已包含完整验证逻辑）
	tokenPair, err := jwtManager.RefreshTokenWithStorage(s.ctx, req.RefreshToken)
	if err != nil {
		// 根据错误类型返回对应的错误码
		resp.Success = false
		if errors.Is(err, jwt.ErrTokenExpired) {
			log.Warn("刷新令牌已过期: %v", err)
			resp.Message = "刷新令牌已过期，请重新登录"
		} else {
			log.Error("令牌刷新失败: %v", err)
			resp.Message = "令牌刷新失败: " + err.Error()
		}
		return &resp, err
	}
	log.Debug("令牌刷新成功")

	// 3. 获取用户信息
	var userID int64
	// 从新生成的刷新令牌中提取用户信息
	claims, err := jwtManager.ValidateRefreshToken(tokenPair.RefreshToken)
	if err == nil && claims != nil {
		// 将字符串用户ID转换为int64
		if id, err := strconv.ParseInt(claims.UserID, 10, 64); err == nil {
			userID = id
		}
		log.Debug("从令牌中提取用户信息: 用户ID=%d", userID)
	}

	// 4. 构建响应
	resp.Success = true
	resp.Message = "令牌刷新成功"
	resp.Token = tokenPair.AccessToken
	resp.RefreshToken = tokenPair.RefreshToken
	resp.ExpiresAt = time.Now().Add(24 * time.Hour).Unix() // 24小时过期（与 AccessTokenTTL 一致）

	// 5. 获取用户信息（如果用户ID有效）
	if userID != 0 {
		log.Debug("开始获取用户信息: 用户ID=%d", userID)
		user, err := s.repo.FindByID(s.repo.DB, userID)
		if err == nil && user != nil {
			resp.UserInfo = util.ConvertSysAccountToUserInfo(user)
			log.Debug("用户信息获取成功: 用户ID=%d, 邮箱=%s", user.ID, user.Email)
		} else {
			log.Warn("获取用户信息失败: %v, 用户ID=%d", err, userID)
		}
	}

	log.Info("令牌刷新成功: 用户ID=%d", userID)
	return &resp, nil
}

// maskIdentifier 对登录标识进行脱敏处理
func maskIdentifier(identifier string) string {
	if len(identifier) == 0 {
		return ""
	}

	// 判断是否是邮箱
	if len(identifier) > 0 && identifier[len(identifier)-1] != '@' {
		// 简单判断：包含@且@不在末尾
		for i, c := range identifier {
			if c == '@' && i > 0 && i < len(identifier)-1 {
				// 邮箱脱敏：保留前3位和@后面的域名
				parts := []rune(identifier)
				if len(parts) > 3 {
					return string(parts[:3]) + "***" + string(parts[i:])
				}
				return identifier
			}
		}
	}

	// 手机号脱敏：保留前3位和后4位
	if len(identifier) == 11 {
		return identifier[:3] + "****" + identifier[7:]
	}

	// 其他情况：只显示前2位
	if len(identifier) > 2 {
		return identifier[:2] + "***"
	}

	return identifier
}
