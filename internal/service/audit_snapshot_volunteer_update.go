package service

import (
	"encoding/json"
	"errors"
	"strings"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
	"volunteer-system/pkg/util"
)

type VolunteerRealNameVerifyAuditPayload struct {
	RealName string `json:"real_name"`
	IDCard   string `json:"id_card"`
}

// VolunteerProfileChangeAuditPayload 志愿者资料变更审核快照（仅承载发生变化的字段）。
// 使用指针字段表示“字段是否参与本次变更”。
type VolunteerProfileChangeAuditPayload struct {
	RealName     *string `json:"real_name,omitempty"`
	Gender       *int32  `json:"gender,omitempty"`
	Birthday     *string `json:"birthday,omitempty"`
	AvatarURL    *string `json:"avatar_url,omitempty"`
	Introduction *string `json:"introduction,omitempty"`
}

// resolveVolunteerUpdateAuditScene 解析审核快照所属场景，并兼容历史无场景封装数据。
func resolveVolunteerUpdateAuditScene(raw string) (string, error) {
	scene, _, isEnvelope, err := parseAuditSnapshotEnvelope(raw)
	if err != nil {
		return "", err
	}
	if isEnvelope {
		return scene, nil
	}
	// 兼容历史记录：未封装 scene 的资料变更按 profile_update 处理。
	return model.AuditSceneVolunteerProfileUpdate, nil
}

// buildVolunteerProfileChangeAuditPayloads 对比请求与当前志愿者信息，构建资料变更前后快照。
func buildVolunteerProfileChangeAuditPayloads(req *api.VolunteerProfileChangeSubmitRequest, volunteer *model.Volunteer) (*VolunteerProfileChangeAuditPayload, *VolunteerProfileChangeAuditPayload, error) {
	if req == nil {
		return nil, nil, errors.New("请求不能为空")
	}
	if volunteer == nil {
		return nil, nil, errors.New("志愿者信息不存在")
	}

	oldPayload := &VolunteerProfileChangeAuditPayload{}
	newPayload := &VolunteerProfileChangeAuditPayload{}

	if req.RealName != nil {
		realName := strings.TrimSpace(req.GetRealName())
		if realName == "" {
			return nil, nil, errors.New("真实姓名不能为空")
		}
		if len(realName) > 50 {
			return nil, nil, errors.New("真实姓名长度不能超过50个字符")
		}
		if realName != volunteer.RealName {
			oldRealName := volunteer.RealName
			newRealName := realName
			oldPayload.RealName = &oldRealName
			newPayload.RealName = &newRealName
		}
	}

	if req.Gender != nil {
		gender := req.GetGender()
		if gender < 0 || gender > 2 {
			return nil, nil, errors.New("性别值无效，0-未知, 1-男, 2-女")
		}
		if gender != volunteer.Gender {
			oldGender := volunteer.Gender
			newGender := gender
			oldPayload.Gender = &oldGender
			newPayload.Gender = &newGender
		}
	}

	if req.Birthday != nil {
		birthdayText := strings.TrimSpace(req.GetBirthday())
		currentBirthday := ""
		if volunteer.Birthday != nil {
			currentBirthday = util.FormatDate(*volunteer.Birthday)
		}
		if birthdayText != "" {
			parsedBirthday, parseErr := util.ParsePastDate(birthdayText)
			if parseErr != nil {
				return nil, nil, errors.New("生日格式错误，请使用 YYYY-MM-DD 格式")
			}
			nextBirthday := util.FormatDate(parsedBirthday)
			if currentBirthday != nextBirthday {
				oldBirthday := currentBirthday
				newBirthday := nextBirthday
				oldPayload.Birthday = &oldBirthday
				newPayload.Birthday = &newBirthday
			}
		} else if volunteer.Birthday != nil {
			oldBirthday := currentBirthday
			newBirthday := ""
			oldPayload.Birthday = &oldBirthday
			newPayload.Birthday = &newBirthday
		}
	}

	if req.AvatarUrl != nil {
		avatarURL := strings.TrimSpace(req.GetAvatarUrl())
		if len(avatarURL) > 255 {
			return nil, nil, errors.New("头像URL长度不能超过255个字符")
		}
		if avatarURL != volunteer.AvatarURL {
			oldAvatarURL := volunteer.AvatarURL
			newAvatarURL := avatarURL
			oldPayload.AvatarURL = &oldAvatarURL
			newPayload.AvatarURL = &newAvatarURL
		}
	}

	if req.Introduction != nil {
		introduction := strings.TrimSpace(req.GetIntroduction())
		if len(introduction) > 2000 {
			return nil, nil, errors.New("个人简介长度不能超过2000个字符")
		}
		if introduction != volunteer.Introduction {
			oldIntroduction := volunteer.Introduction
			newIntroduction := introduction
			oldPayload.Introduction = &oldIntroduction
			newPayload.Introduction = &newIntroduction
		}
	}

	return oldPayload, newPayload, nil
}

// unmarshalVolunteerProfileChangeAuditPayload 解析并校验资料变更审核快照。
func unmarshalVolunteerProfileChangeAuditPayload(raw string) (*VolunteerProfileChangeAuditPayload, error) {
	scene, data, isEnvelope, err := parseAuditSnapshotEnvelope(raw)
	if err != nil {
		return nil, err
	}
	if isEnvelope && scene != model.AuditSceneVolunteerProfileUpdate {
		return nil, errors.New("审核场景不匹配")
	}

	var payload VolunteerProfileChangeAuditPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	if payload.IsEmpty() {
		return nil, errors.New("审核内容无有效字段")
	}
	return &payload, nil
}

// IsEmpty 判断资料变更快照是否不包含任何有效字段。
func (p *VolunteerProfileChangeAuditPayload) IsEmpty() bool {
	return p == nil ||
		(p.RealName == nil &&
			p.Gender == nil &&
			p.Birthday == nil &&
			p.AvatarURL == nil &&
			p.Introduction == nil)
}

// BuildVolunteerUpdates 将资料变更快照转换为志愿者资料更新字段并执行校验。
func (p *VolunteerProfileChangeAuditPayload) BuildVolunteerUpdates() (map[string]any, error) {
	updates := make(map[string]any)

	if p.RealName != nil {
		realName := strings.TrimSpace(*p.RealName)
		if realName == "" {
			return nil, errors.New("真实姓名不能为空")
		}
		if len(realName) > 50 {
			return nil, errors.New("真实姓名长度不能超过50个字符")
		}
		updates["real_name"] = realName
	}

	if p.Gender != nil {
		gender := *p.Gender
		if gender < 0 || gender > 2 {
			return nil, errors.New("性别值无效，0-未知, 1-男, 2-女")
		}
		updates["gender"] = gender
	}

	if p.Birthday != nil {
		birthdayText := strings.TrimSpace(*p.Birthday)
		if birthdayText == "" {
			updates["birthday"] = nil
		} else {
			birthday, parseErr := util.ParsePastDate(birthdayText)
			if parseErr != nil {
				return nil, errors.New("生日格式错误，请使用 YYYY-MM-DD 格式")
			}
			updates["birthday"] = &birthday
		}
	}

	if p.AvatarURL != nil {
		avatarURL := strings.TrimSpace(*p.AvatarURL)
		if len(avatarURL) > 255 {
			return nil, errors.New("头像URL长度不能超过255个字符")
		}
		updates["avatar_url"] = avatarURL
	}

	if p.Introduction != nil {
		introduction := strings.TrimSpace(*p.Introduction)
		if len(introduction) > 2000 {
			return nil, errors.New("个人简介长度不能超过2000个字符")
		}
		updates["introduction"] = introduction
	}

	if len(updates) == 0 {
		return nil, errors.New("审核内容无有效字段")
	}
	return updates, nil
}

// buildVolunteerRealNameVerifyAuditPayloads 对比请求与当前志愿者信息，构建实名审核前后快照。
func buildVolunteerRealNameVerifyAuditPayloads(req *api.VolunteerRealNameSubmitRequest, volunteer *model.Volunteer) (*VolunteerRealNameVerifyAuditPayload, *VolunteerRealNameVerifyAuditPayload, error) {
	if req == nil {
		return nil, nil, errors.New("请求不能为空")
	}
	if volunteer == nil {
		return nil, nil, errors.New("志愿者信息不存在")
	}

	realName, err := normalizeVolunteerRealName(req.RealName)
	if err != nil {
		return nil, nil, err
	}
	idCard, err := normalizeVolunteerIDCard(req.IdCard)
	if err != nil {
		return nil, nil, err
	}

	oldPayload := &VolunteerRealNameVerifyAuditPayload{
		RealName: strings.TrimSpace(volunteer.RealName),
		IDCard:   strings.TrimSpace(volunteer.IDCard),
	}
	newPayload := &VolunteerRealNameVerifyAuditPayload{
		RealName: realName,
		IDCard:   idCard,
	}

	if oldPayload.RealName == newPayload.RealName &&
		oldPayload.IDCard == newPayload.IDCard &&
		volunteer.AuditStatus == model.VolunteerAuditStatusApproved {
		return nil, nil, errors.New("实名认证信息未发生变化")
	}

	return oldPayload, newPayload, nil
}

// unmarshalVolunteerRealNameVerifyAuditPayload 解析并校验实名认证审核快照。
func unmarshalVolunteerRealNameVerifyAuditPayload(raw string) (*VolunteerRealNameVerifyAuditPayload, error) {
	scene, data, isEnvelope, err := parseAuditSnapshotEnvelope(raw)
	if err != nil {
		return nil, err
	}
	if isEnvelope && scene != model.AuditSceneVolunteerRealNameVerify {
		return nil, errors.New("审核场景不匹配")
	}

	var payload VolunteerRealNameVerifyAuditPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	realName, err := normalizeVolunteerRealName(payload.RealName)
	if err != nil {
		return nil, err
	}
	idCard, err := normalizeVolunteerIDCard(payload.IDCard)
	if err != nil {
		return nil, err
	}
	payload.RealName = realName
	payload.IDCard = idCard
	return &payload, nil
}

// BuildVolunteerApprovalUpdates 将实名认证审核快照转换为审核通过后的更新字段。
func (p *VolunteerRealNameVerifyAuditPayload) BuildVolunteerApprovalUpdates() (map[string]any, error) {
	realName, err := normalizeVolunteerRealName(p.RealName)
	if err != nil {
		return nil, err
	}
	idCard, err := normalizeVolunteerIDCard(p.IDCard)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"real_name": realName,
		"id_card":   idCard,
	}, nil
}

// normalizeVolunteerRealName 规范化并校验真实姓名。
func normalizeVolunteerRealName(realName string) (string, error) {
	normalized := strings.TrimSpace(realName)
	if normalized == "" {
		return "", errors.New("真实姓名不能为空")
	}
	if len(normalized) > 50 {
		return "", errors.New("真实姓名长度不能超过50个字符")
	}
	return normalized, nil
}

// normalizeVolunteerIDCard 规范化并校验身份证号格式。
func normalizeVolunteerIDCard(idCard string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(idCard))
	if normalized == "" {
		return "", errors.New("身份证号不能为空")
	}

	length := len(normalized)
	if length != 15 && length != 18 {
		return "", errors.New("身份证号格式不正确")
	}

	for i := 0; i < length; i++ {
		ch := normalized[i]
		if ch >= '0' && ch <= '9' {
			continue
		}
		if length == 18 && i == 17 && ch == 'X' {
			continue
		}
		return "", errors.New("身份证号格式不正确")
	}
	return normalized, nil
}
