package repository

import (
	"volunteer-system/internal/model"

	"gorm.io/gorm"
)

// FindByMobile 根据手机号查找用户（通过哈希值）
func (r *Repository) FindByMobile(db *gorm.DB, mobileHash string) (*model.SysAccount, error) {
	var user model.SysAccount
	err := db.WithContext(r.ctx).Model(&model.SysAccount{}).Where("mobile_hash = ?", mobileHash).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID 根据ID查找用户
func (r *Repository) FindByID(db *gorm.DB, id int64) (*model.SysAccount, error) {
	var user model.SysAccount
	err := db.WithContext(r.ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateLastLoginTime 更新最后登录时间
func (r *Repository) UpdateLastLoginTime(db *gorm.DB, userID int64) error {
	err := db.WithContext(r.ctx).Model(&model.SysAccount{}).
		Where("id = ?", userID).
		Update("last_login_time", gorm.Expr("CURRENT_TIMESTAMP")).Error
	if err != nil {
		return err
	}
	return nil
}

// CheckUserStatus 检查用户状态
func (r *Repository) CheckUserStatus(db *gorm.DB, userID int64) (bool, error) {
	var user model.SysAccount
	err := db.WithContext(r.ctx).Select("status").Where("id = ?", userID).First(&user).Error
	if err != nil {
		return false, err
	}
	return user.Status == 1, nil // 1表示正常状态
}

// CreateAccount 创建系统账户
func (r *Repository) CreateAccount(db *gorm.DB, account *model.SysAccount) error {
	err := db.WithContext(r.ctx).Create(account).Error
	if err != nil {
		return err
	}
	return nil
}

// CheckMobileExists 检查手机号是否已存在（通过哈希值）
func (r *Repository) CheckMobileExists(db *gorm.DB, mobileHash string) (bool, error) {
	var count int64
	err := db.WithContext(r.ctx).Model(&model.SysAccount{}).Where("mobile_hash = ? AND deleted_at IS NULL", mobileHash).Count(&count).Error
	if err != nil {
		return false, err
	}

	if count > 0 {
		return true, nil
	}
	return false, nil
}

// CheckEmailExists 检查邮箱是否已存在
func (r *Repository) CheckEmailExists(db *gorm.DB, email string) (bool, error) {
	var count int64
	err := db.WithContext(r.ctx).Model(&model.SysAccount{}).Where("email = ? AND deleted_at IS NULL", email).Count(&count).Error
	if err != nil {
		return false, err
	}

	if count > 0 {
		return true, nil
	}
	return false, nil
}

func (r *Repository) FindByEmail(db *gorm.DB, email string) (*model.SysAccount, error) {
	var user model.SysAccount
	err := db.WithContext(r.ctx).Model(&model.SysAccount{}).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}