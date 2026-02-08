package util

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword 使用 bcrypt 加密密码
func HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// CheckPassword 验证密码是否匹配
func CheckPassword(password, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// ValidatePasswordStrength 验证密码强度
func ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return errors.New("密码长度至少8位")
	}

	// 检查是否包含数字
	hasDigit := false
	// 检查是否包含字母
	hasLetter := false

	for _, char := range password {
		if char >= '0' && char <= '9' {
			hasDigit = true
		} else if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') {
			hasLetter = true
		}
	}

	if !hasDigit || !hasLetter {
		return errors.New("密码必须包含字母和数字")
	}

	return nil
}