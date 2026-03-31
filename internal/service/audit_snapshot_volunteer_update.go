package service

import (
	"encoding/json"
	"errors"
	"strings"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
)

type VolunteerRealNameVerifyAuditPayload struct {
	RealName string `json:"real_name"`
	IDCard   string `json:"id_card"`
	OrgID    int64  `json:"org_id"`
}

// resolveVolunteerUpdateAuditScene 解析实名认证审核快照所属场景。
func resolveVolunteerUpdateAuditScene(raw string) (string, error) {
	scene, _, isEnvelope, err := parseAuditSnapshotEnvelope(raw)
	if err != nil {
		return "", err
	}
	if !isEnvelope {
		return "", errors.New("审核场景不匹配")
	}
	if scene != model.AuditSceneVolunteerRealNameVerify {
		return "", errors.New("审核场景不匹配")
	}
	return scene, nil
}

// buildVolunteerRealNameVerifyAuditPayloads 对比请求与当前志愿者信息，构建实名审核前后快照。
func buildVolunteerRealNameVerifyAuditPayloads(req *api.VolunteerRealNameSubmitRequest, volunteer *model.Volunteer, orgID int64) (*VolunteerRealNameVerifyAuditPayload, *VolunteerRealNameVerifyAuditPayload, error) {
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
		OrgID:    orgID,
	}
	newPayload := &VolunteerRealNameVerifyAuditPayload{
		RealName: realName,
		IDCard:   idCard,
		OrgID:    orgID,
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
	if payload.OrgID < 0 {
		return nil, errors.New("组织ID不合法")
	}
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
