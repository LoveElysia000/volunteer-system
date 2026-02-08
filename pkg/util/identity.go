package util

// GetIdentityTypeFromString 将身份字符串转换为数据库中的身份类型
func GetIdentityTypeFromString(identity string) int32 {
	switch identity {
	case "volunteer":
		return 1
	case "organization":
		return 2
	default:
		return 0 // 未知类型
	}
}

// ValidateLoginType 验证登录类型是否有效
func ValidateLoginType(loginType string) bool {
	switch loginType {
	case "email", "phone":
		return true
	default:
		return false
	}
}

// ValidateIdentity 验证身份类型是否有效
func ValidateIdentity(identity string) bool {
	switch identity {
	case "volunteer", "organization":
		return true
	default:
		return false
	}
}