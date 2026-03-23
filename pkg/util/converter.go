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

	identity := model.DefaultUnknownValue
	if value, ok := model.IdentityTypeTextMap[account.IdentityType]; ok {
		identity = value
	}

	phone := ""
	if account.Mobile != "" {
		decrypted, err := DecryptSensitiveField(account.Mobile)
		if err == nil {
			phone = decrypted
		}
	}

	displayName := account.Username
	if displayName == "" {
		displayName = "用户" + phone
	}

	return &api.UserInfo{
		UserId:      strconv.FormatInt(account.ID, 10),
		UserName:    account.Username,
		Email:       account.Email,
		Phone:       phone,
		DisplayName: displayName,
		AvatarUrl:   "",
		Identity:    identity,
		CreatedAt:   account.CreatedAt.Unix(),
		UpdatedAt:   account.UpdatedAt.Unix(),
	}
}
