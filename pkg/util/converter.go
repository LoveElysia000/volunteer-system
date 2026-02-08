package util

import (
	"strconv"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
)

// ConvertSysAccountToUserInfo 将SysAccount转换为UserInfo
func ConvertSysAccountToUserInfo(account *model.SysAccount) *api.UserInfo {
	if account == nil {
		return nil
	}

	// 将身份类型转换为字符串
	var identity string
	switch account.IdentityType {
	case 1:
		identity = "volunteer"
	case 2:
		identity = "organization"
	default:
		identity = "unknown"
	}

	return &api.UserInfo{
		UserId:      strconv.FormatInt(account.ID, 10),
		Username:    account.Username, // 使用手机号作为用户名
		Email:       "",               // 如果需要邮箱，需要从其他表获取
		Phone:       account.Mobile,
		DisplayName: account.Mobile, // 使用手机号作为显示名
		AvatarUrl:   "",             // 默认头像URL
		Identity:    identity,
		CreatedAt:   account.CreatedAt.Unix(),
		UpdatedAt:   account.UpdatedAt.Unix(),
	}
}
