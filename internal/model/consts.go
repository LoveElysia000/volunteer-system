package model

const (
	// 注册类型
	RegisterTypeVolunteer    = "volunteer"    // 志愿者注册
	RegisterTypeOrganization = "organization" // 组织注册

	// 注册类型数字映射
	RegisterTypeVolunteerCode    = 1 // 志愿者注册
	RegisterTypeOrganizationCode = 2 // 组织管理者注册

	// 审核目标类型（对应 audit_records.target_type）
	AuditTargetVolunteer int32 = 1 // 志愿者实名审核
	AuditTargetOrg       int32 = 2 // 组织资质审核
	AuditTargetMember    int32 = 3 // 志愿者加入组织审核
	AuditTargetSignup    int32 = 4 // 活动报名审核

	// 审核通用状态（用于当前审核目标）
	AuditStatusPending  int32 = 1 // 待审核
	AuditStatusApproved int32 = 2 // 审核通过
	AuditStatusRejected int32 = 3 // 审核拒绝

	// 审核结果编码（对应 audit_records.audit_result，由审核状态推导）
	auditResultPassCode   int32 = 1 // 通过
	auditResultRejectCode int32 = 2 // 驳回

	// 审核类型（当前仅支持志愿者加入组织审核）
	AuditTypeVolunteerJoinOrganization int32 = AuditTargetMember // 志愿者加入组织

	// 成员状态
	MemberStatusPending  int32 = 1 // 待审核
	MemberStatusActive   int32 = 2 // 正式成员（已通过）
	MemberStatusRejected int32 = 3 // 已拒绝
	MemberStatusLeft     int32 = 4 // 已退出

	// 成员角色
	MemberRoleMember  int32 = 1 // 普通成员
	MemberRoleManager int32 = 2 // 管理员
	MemberRoleLeader  int32 = 3 // 负责人

	// 数据操作类型
	OperationTypeCreate int32 = 1 // 新增
	OperationTypeUpdate int32 = 2 // 更新
	OperationTypeDelete int32 = 3 // 删除

	// 组织状态
	OrganizationDisabled int32 = 0 // 停用
	OrganizationNormal   int32 = 1 // 正常
)

// GetRegisterTypeCode 根据注册类型字符串返回对应的数字代码
func GetRegisterTypeCode(registerType string) int {
	switch registerType {
	case RegisterTypeVolunteer:
		return RegisterTypeVolunteerCode
	case RegisterTypeOrganization:
		return RegisterTypeOrganizationCode
	default:
		return 0 // 未知类型返回0
	}
}

// IsValidAuditTargetType 返回审核目标类型是否合法
func IsValidAuditTargetType(targetType int32) bool {
	switch targetType {
	case AuditTargetVolunteer, AuditTargetOrg, AuditTargetMember, AuditTargetSignup:
		return true
	default:
		return false
	}
}

// IsValidAuditResult 返回审核结果是否合法
func IsValidAuditResult(auditResult int32) bool {
	return auditResult == auditResultPassCode || auditResult == auditResultRejectCode
}

// ResolveAuditResult 根据审核状态计算审核结果
func ResolveAuditResult(auditStatus int32) int32 {
	switch auditStatus {
	case AuditStatusApproved:
		return auditResultPassCode
	case AuditStatusRejected:
		return auditResultRejectCode
	default:
		return 0
	}
}
