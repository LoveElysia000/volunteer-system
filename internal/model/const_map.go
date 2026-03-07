package model

const (
	DefaultUnknownText  = "未知"
	DefaultUnknownValue = "unknown"
	DefaultOtherText    = "其他"
)

var (
	AuditResultByStatus = map[int32]int32{
		AuditStatusApproved: auditResultPassCode,
		AuditStatusRejected: auditResultRejectCode,
	}

	GenderCodeByText = map[string]int32{
		"男":  1,
		"女":  2,
		"未知": 0,
	}

	VolunteerGenderTextByCode = map[int32]string{
		0: "未知",
		1: "男",
		2: "女",
	}

	VolunteerStatusTextByCode = map[int32]string{
		VolunteerActiveStatus:   "活跃",
		VolunteerInactiveStatus: "非活跃",
		VolunteerEtcStatus:      "其他",
	}

	VolunteerAuditStatusTextByCode = map[int32]string{
		VolunteerAuditStatusUnverified: "未认证",
		VolunteerAuditStatusPending:    "审核中",
		VolunteerAuditStatusApproved:   "已通过",
		VolunteerAuditStatusRejected:   "已驳回",
	}

	ActivityStatusTextByCode = map[int32]string{
		ActivityStatusRecruiting: "报名中",
		ActivityStatusFinished:   "已结束",
		ActivityStatusCanceled:   "已取消",
	}

	IdentityTypeCodeMap = map[string]int32{
		RegisterTypeVolunteer:    RegisterTypeVolunteerCode,
		RegisterTypeOrganization: RegisterTypeOrganizationCode,
	}

	IdentityTypeTextMap = map[int32]string{
		RegisterTypeVolunteerCode:    RegisterTypeVolunteer,
		RegisterTypeOrganizationCode: RegisterTypeOrganization,
	}

	LoginTypeSet = map[string]struct{}{
		"email": {},
		"phone": {},
	}
)
