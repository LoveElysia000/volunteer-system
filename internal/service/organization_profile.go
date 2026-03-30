package service

import (
	"errors"
	"strings"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
	"volunteer-system/pkg/util"
)

func buildOrganizationListItem(org *model.Organization) (*api.OrganizationListItem, error) {
	if org == nil {
		return nil, errors.New("组织信息不存在")
	}

	contactPhone, err := decryptOptionalSensitiveValue(org.ContactPhone)
	if err != nil {
		return nil, err
	}

	return &api.OrganizationListItem{
		Id:               org.ID,
		Name:             org.OrgName,
		OrganizationCode: org.LicenseCode,
		ContactPerson:    org.ContactPerson,
		ContactPhone:     contactPhone,
		Address:          org.Address,
		Status:           org.Status,
		OrganizationType: "",
		Region:           "",
		CreatedAt:        org.CreatedAt.Format(util.DateTimeLayout),
	}, nil
}

func buildPublicOrganizationListItem(org *model.Organization) (*api.OrganizationListItem, error) {
	if org == nil {
		return nil, errors.New("组织信息不存在")
	}

	return &api.OrganizationListItem{
		Id:               org.ID,
		Name:             org.OrgName,
		OrganizationCode: org.LicenseCode,
		ContactPerson:    org.ContactPerson,
		ContactPhone:     "",
		Address:          org.Address,
		Status:           org.Status,
		OrganizationType: "",
		Region:           "",
		CreatedAt:        org.CreatedAt.Format(util.DateTimeLayout),
	}, nil
}

func buildOrganizationDetailResponse(org *model.Organization, account *model.SysAccount) (*api.OrganizationDetailResponse, error) {
	if org == nil {
		return nil, errors.New("组织信息不存在")
	}

	contactPhone, err := decryptOptionalSensitiveValue(org.ContactPhone)
	if err != nil {
		return nil, err
	}
	accountPhone := ""
	if account != nil {
		accountPhone, err = decryptOptionalSensitiveValue(account.Mobile)
		if err != nil {
			return nil, err
		}
	}
	email := ""
	userName := ""
	if account != nil {
		email = account.Email
		userName = account.UserName
	}

	return &api.OrganizationDetailResponse{
		Organization: &api.OrganizationInfo{
			Id:               org.ID,
			AccountId:        org.AccountID,
			Name:             org.OrgName,
			OrganizationCode: org.LicenseCode,
			ContactPerson:    org.ContactPerson,
			ContactPhone:     contactPhone,
			Email:            email,
			Address:          org.Address,
			Status:           org.Status,
			OrganizationType: "",
			Region:           "",
			Description:      org.Introduction,
			WebsiteUrl:       "",
			LogoUrl:          org.LogoURL,
			CreatedAt:        org.CreatedAt.Format(util.DateTimeLayout),
			UpdatedAt:        org.UpdatedAt.Format(util.DateTimeLayout),
		},
		AccountInfo: &api.OrganizationAccountInfo{
			UserName: userName,
			Email:    email,
			Phone:    accountPhone,
		},
		OrganizationProfile: &api.OrganizationProfileInfo{
			Name:          org.OrgName,
			ContactPerson: org.ContactPerson,
			ContactPhone:  contactPhone,
			Address:       org.Address,
			Description:   org.Introduction,
			LogoUrl:       org.LogoURL,
		},
		OrganizationCertification: &api.OrganizationCertificationInfo{
			OrganizationCode: org.LicenseCode,
		},
	}, nil
}

func buildPublicOrganizationDetailResponse(org *model.Organization) (*api.PublicOrganizationDetailResponse, error) {
	if org == nil {
		return nil, errors.New("组织信息不存在")
	}

	contactPhone, err := decryptOptionalSensitiveValue(org.ContactPhone)
	if err != nil {
		return nil, err
	}

	return &api.PublicOrganizationDetailResponse{
		Organization: &api.PublicOrganizationInfo{
			Id:               org.ID,
			Name:             org.OrgName,
			OrganizationCode: org.LicenseCode,
			ContactPerson:    org.ContactPerson,
			ContactPhone:     contactPhone,
			Address:          org.Address,
			Status:           org.Status,
			OrganizationType: "",
			Region:           "",
			Description:      org.Introduction,
			WebsiteUrl:       "",
			LogoUrl:          org.LogoURL,
			CreatedAt:        org.CreatedAt.Format(util.DateTimeLayout),
			UpdatedAt:        org.UpdatedAt.Format(util.DateTimeLayout),
		},
		OrganizationProfile: &api.OrganizationProfileInfo{
			Name:          org.OrgName,
			ContactPerson: org.ContactPerson,
			ContactPhone:  contactPhone,
			Address:       org.Address,
			Description:   org.Introduction,
			LogoUrl:       org.LogoURL,
		},
		OrganizationCertification: &api.OrganizationCertificationInfo{
			OrganizationCode: org.LicenseCode,
		},
	}, nil
}

func buildOrganizationAccountUpdateMutations(req *api.OrganizationAccountUpdateRequest) (map[string]any, error) {
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

func decryptOptionalSensitiveValue(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	decrypted, err := util.DecryptSensitiveField(trimmed)
	if err == nil {
		return decrypted, nil
	}
	if looksLikeEncryptedValue(trimmed) {
		return "", err
	}
	return trimmed, nil
}

func organizationPhoneContainsKeyword(storedPhone, keyword string) (bool, error) {
	phone, err := decryptOptionalSensitiveValue(storedPhone)
	if err != nil {
		return false, err
	}
	return strings.Contains(phone, keyword), nil
}

func looksLikeEncryptedValue(value string) bool {
	if len(value) < 16 || len(value)%4 != 0 {
		return false
	}
	for _, ch := range value {
		switch {
		case ch >= 'A' && ch <= 'Z':
		case ch >= 'a' && ch <= 'z':
		case ch >= '0' && ch <= '9':
		case ch == '+' || ch == '/' || ch == '=':
		default:
			return false
		}
	}
	return true
}
