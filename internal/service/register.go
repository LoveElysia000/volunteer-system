package service

import (
	"context"
	"errors"
	"regexp"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

type RegisterService struct {
	Service
}

func NewRegisterService(ctx context.Context, c *app.RequestContext) *RegisterService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &RegisterService{
		Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
}

// RegisterVolunteer 志愿者注册
func (s *RegisterService) RegisterVolunteer(req *api.RegisterRequest) (*api.RegisterResponse, error) {
	// 验证必填字段
	if err := s.validateVolunteerRequest(req); err != nil {
		log.Warn("志愿者注册验证失败: %v", err)
		return nil, err
	}

	// 处理手机号：生成哈希和加密值
	mobilePair, err := util.ProcessSensitiveField(req.Phone)
	if err != nil {
		log.Error("志愿者注册 - 手机号处理失败: %v", err)
		return nil, errors.New("手机号处理失败")
	}

	// 检查手机号是否已存在（通过哈希值）
	exists, err := s.repo.CheckMobileExists(s.repo.DB, mobilePair.Hash)
	if err != nil {
		log.Error("志愿者注册 - 检查手机号是否存在失败: %v", err)
		return nil, errors.New("检查手机号失败")
	}
	if exists {
		log.Warn("志愿者注册失败: 手机号已存在 - %s", req.Phone)
		return nil, errors.New("手机号已存在")
	}

	// 检查邮箱是否已存在
	exists, err = s.repo.CheckEmailExists(s.repo.DB, req.Email)
	if err != nil {
		log.Error("志愿者注册 - 检查邮箱是否存在失败: %v", err)
		return nil, errors.New("检查邮箱失败")
	}
	if exists {
		log.Warn("志愿者注册失败: 邮箱已存在 - %s", req.Email)
		return nil, errors.New("邮箱已存在")
	}
	// 密码加密
	hashedPassword, err := util.HashPassword(req.Password) // 默认密码，实际应用中应该由用户设置
	if err != nil {
		log.Error("志愿者注册 - 密码加密失败: %v", err)
		return nil, errors.New("密码加密失败")
	}

	// 转换性别
	genderCode, err := s.convertGenderToCode(req.Gender)
	if err != nil {
		log.Warn("志愿者注册 - 性别转换失败: %v", err)
		return nil, err
	}

	var account *model.SysAccount
	err = s.repo.DB.Transaction(func(tx *gorm.DB) error {
		// 创建系统账户
		account = &model.SysAccount{
			Username:     req.UserName,
			Mobile:       mobilePair.Encrypted,
			MobileHash:   mobilePair.Hash,
			Email:        req.Email,
			Password:     hashedPassword,
			IdentityType: model.RegisterTypeVolunteerCode,
			Status:       1, // 正常状态
			CreatedAt:    time.Now(),
		}
		err = s.repo.CreateAccount(tx, account)
		if err != nil {
			log.Error("志愿者注册 - 创建系统账户失败: %v", err)
			return err
		}

		// 创建志愿者档案
		volunteer := &model.Volunteer{
			AccountID:   account.ID,
			RealName:    req.Name,
			Gender:      genderCode,
			AuditStatus: 0, // 未认证
			CreatedAt:   time.Now(),
		}
		err = s.repo.CreateVolunteer(tx, volunteer)
		if err != nil {
			log.Error("志愿者注册 - 创建志愿者档案失败: %v", err)
			return err
		}
		return nil
	})
	if err != nil {
		log.Error("志愿者注册 - 事务执行失败: %v", err)
		return nil, err
	}

	log.Info("志愿者注册成功: 账户ID=%d, 姓名=%s, 手机号=%s", account.ID, req.Name, util.GetMobileMask(req.Phone))
	return &api.RegisterResponse{}, nil
}

// RegisterOrganization 组织注册
func (s *RegisterService) RegisterOrganization(req *api.RegisterRequest) (*api.RegisterResponse, error) {
	// 验证必填字段
	if err := s.validateOrganizationRequest(req); err != nil {
		log.Warn("组织注册验证失败: %v", err)
		return nil, err
	}

	// 处理手机号：生成哈希和加密值
	mobilePair, err := util.ProcessSensitiveField(req.Phone)
	if err != nil {
		log.Error("组织注册 - 手机号处理失败: %v", err)
		return nil, errors.New("手机号处理失败")
	}

	// 检查手机号是否已存在（通过哈希值）
	exists, err := s.repo.CheckMobileExists(s.repo.DB, mobilePair.Hash)
	if err != nil {
		log.Error("组织注册 - 检查手机号是否存在失败: %v", err)
		return nil, errors.New("检查手机号失败")
	}
	if exists {
		log.Warn("组织注册失败: 手机号已存在 - %s", req.Phone)
		return nil, errors.New("手机号已存在")
	}

	// 检查邮箱是否已存在
	exists, err = s.repo.CheckEmailExists(s.repo.DB, req.Email)
	if err != nil {
		log.Error("组织注册 - 检查邮箱是否存在失败: %v", err)
		return nil, errors.New("检查邮箱失败")
	}
	if exists {
		log.Warn("组织注册失败: 邮箱已存在 - %s", req.Email)
		return nil, errors.New("邮箱已存在")
	}

	// 密码加密
	hashedPassword, err := util.HashPassword(req.Password) // 默认密码，实际应用中应该由用户设置
	if err != nil {
		log.Error("组织注册 - 密码加密失败: %v", err)
		return nil, errors.New("密码加密失败")
	}

	var account *model.SysAccount
	err = s.repo.DB.Transaction(func(tx *gorm.DB) error {
		// 创建系统账户
		account = &model.SysAccount{
			Username:     req.OrganizationName,
			Mobile:       mobilePair.Encrypted,
			MobileHash:   mobilePair.Hash,
			Email:        req.Email,
			Password:     hashedPassword,
			IdentityType: model.RegisterTypeOrganizationCode,
			Status:       1, // 正常状态
			CreatedAt:    time.Now(),
		}

		err = s.repo.CreateAccount(tx, account)
		if err != nil {
			log.Error("组织注册 - 创建系统账户失败: %v", err)
			return err
		}

		// 创建组织档案
		org := &model.Organization{
			AccountID:     account.ID,
			OrgName:       req.OrganizationName,
			ContactPerson: req.Name,
			ContactPhone:  mobilePair.Encrypted, // 组织联系手机号也使用加密存储
			// TODO(audit-status-removal): organizations 表移除 audit_status 后，删除该字段初始化。
			AuditStatus: 0, // 未提交
			CreatedAt:   time.Now(),
		}
		err = s.repo.CreateOrganization(tx, org)
		if err != nil {
			log.Error("组织注册 - 创建组织档案失败: %v", err)
			return err
		}

		return nil
	})

	if err != nil {
		log.Error("组织注册 - 事务执行失败: %v", err)
		return nil, err
	}
	log.Info("组织注册成功: 账户ID=%d, 组织名称=%s, 联系人=%s", account.ID, req.OrganizationName, req.Name)
	return &api.RegisterResponse{}, nil
}

// validateVolunteerRequest 验证志愿者注册请求
func (s *RegisterService) validateVolunteerRequest(req *api.RegisterRequest) error {
	if req.Name == "" {
		return errors.New("姓名不能为空")
	}

	if req.Phone == "" {
		return errors.New("手机号不能为空")
	}

	// 验证手机号格式
	if !s.isValidMobile(req.Phone) {
		return errors.New("手机号格式不正确")
	}

	if req.Email == "" {
		return errors.New("邮箱不能为空")
	}

	// 验证邮箱格式
	if !s.isValidEmail(req.Email) {
		return errors.New("邮箱格式不正确")
	}

	if req.Age <= 0 || req.Age > 120 {
		return errors.New("年龄必须在1-120岁之间")
	}

	if req.Gender == "" {
		return errors.New("性别不能为空")
	}

	// 验证性别格式
	if _, err := s.convertGenderToCode(req.Gender); err != nil {
		return errors.New("性别格式不正确，支持：男、女、未知")
	}

	return nil
}

// validateOrganizationRequest 验证组织注册请求
func (s *RegisterService) validateOrganizationRequest(req *api.RegisterRequest) error {
	if req.Name == "" {
		return errors.New("联系人姓名不能为空")
	}

	if req.Phone == "" {
		return errors.New("手机号不能为空")
	}

	// 验证手机号格式
	if !s.isValidMobile(req.Phone) {
		return errors.New("手机号格式不正确")
	}

	if req.Email == "" {
		return errors.New("邮箱不能为空")
	}

	// 验证邮箱格式
	if !s.isValidEmail(req.Email) {
		return errors.New("邮箱格式不正确")
	}

	if req.OrganizationName == "" {
		return errors.New("组织名称不能为空")
	}

	return nil
}

// isValidMobile 验证手机号格式
func (s *RegisterService) isValidMobile(mobile string) bool {
	// 简单的手机号格式验证，支持11位数字
	matched, _ := regexp.MatchString(`^1[3-9]\d{9}$`, mobile)
	return matched
}

// isValidEmail 验证邮箱格式
func (s *RegisterService) isValidEmail(email string) bool {
	// 简单的邮箱格式验证
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`, email)
	return matched
}

// convertGenderToCode 转换性别字符串为数字代码
func (s *RegisterService) convertGenderToCode(gender string) (int32, error) {
	switch gender {
	case "男":
		return 1, nil
	case "女":
		return 2, nil
	case "未知":
		return 0, nil
	default:
		return 0, errors.New("不支持的性别")
	}
}
