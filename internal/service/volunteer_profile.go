package service

import (
	"errors"
	"regexp"
	"strings"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
	"volunteer-system/pkg/util"
)

var (
	volunteerAccountMobilePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)
	volunteerAccountEmailPattern  = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

func buildVolunteerMyProfileResponse(volunteer *model.Volunteer, account *model.SysAccount) (*api.MyProfileResponse, error) {
	if volunteer == nil {
		return nil, errors.New("志愿者信息不存在")
	}

	birthday := ""
	if volunteer.Birthday != nil {
		birthday = volunteer.Birthday.Format("2006-01-02")
	}

	phone := ""
	if account != nil && strings.TrimSpace(account.Mobile) != "" {
		decrypted, err := util.DecryptSensitiveField(account.Mobile)
		if err != nil {
			return nil, err
		}
		phone = decrypted
	}

	resp := &api.MyProfileResponse{
		Volunteer: &api.VolunteerInfo{
			Id:           volunteer.ID,
			AccountId:    volunteer.AccountID,
			RealName:     volunteer.RealName,
			Gender:       volunteer.Gender,
			Birthday:     birthday,
			IdCard:       volunteer.IDCard,
			AvatarUrl:    volunteer.AvatarURL,
			Introduction: volunteer.Introduction,
			TotalHours:   volunteer.TotalHours,
			ServiceCount: volunteer.ServiceCount,
			CreditScore:  volunteer.CreditScore,
			Status:       volunteer.Status,
			AuditStatus:  volunteer.AuditStatus,
			CreatedAt:    volunteer.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:    volunteer.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
		AccountInfo: &api.VolunteerAccountInfo{},
		Profile: &api.VolunteerProfileInfo{
			Gender:       volunteer.Gender,
			Birthday:     birthday,
			AvatarUrl:    volunteer.AvatarURL,
			Introduction: volunteer.Introduction,
		},
		Verification: &api.VolunteerVerificationInfo{
			RealName:    volunteer.RealName,
			IdCard:      volunteer.IDCard,
			AuditStatus: volunteer.AuditStatus,
		},
	}

	if account != nil {
		resp.AccountInfo = &api.VolunteerAccountInfo{
			UserName: account.UserName,
			Email:    account.Email,
			Phone:    phone,
		}
	}

	return resp, nil
}

func buildVolunteerAccountUpdateMutations(req *api.VolunteerAccountUpdateRequest) (map[string]any, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}

	updates := make(map[string]any)

	if req.UserName != "" {
		userName := strings.TrimSpace(req.UserName)
		if userName == "" {
			return nil, errors.New("用户名不能为空")
		}
		if len(userName) > 50 {
			return nil, errors.New("用户名长度不能超过50个字符")
		}
		updates["user_name"] = userName
	}

	if req.Email != "" {
		email := strings.TrimSpace(strings.ToLower(req.Email))
		if !volunteerAccountEmailPattern.MatchString(email) {
			return nil, errors.New("邮箱格式不正确")
		}
		updates["email"] = email
	}

	if req.Phone != "" {
		phone := strings.TrimSpace(req.Phone)
		if !volunteerAccountMobilePattern.MatchString(phone) {
			return nil, errors.New("手机号格式不正确")
		}
		phonePair, err := util.ProcessSensitiveField(phone)
		if err != nil {
			return nil, errors.New("手机号处理失败")
		}
		updates["mobile"] = phonePair.Encrypted
		updates["mobile_hash"] = phonePair.Hash
	}

	if len(updates) == 0 {
		return nil, errors.New("没有需要更新的字段")
	}

	return updates, nil
}
