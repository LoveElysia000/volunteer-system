package model

const (
	// 注册类型
	RegisterTypeVolunteer    = "volunteer"    // 志愿者注册
	RegisterTypeOrganization = "organization" // 组织注册

	// 注册类型数字映射
	RegisterTypeVolunteerCode    = 1 // 志愿者注册
	RegisterTypeOrganizationCode = 2 // 组织注册

	// 审核状态
	ApprovalStatusPending  = "pending"  // 待审核
	ApprovalStatusApproved = "approved" // 审核通过
	ApprovalStatusRejected = "rejected" // 审核拒绝

	// 审核状态数字映射
	ApprovalStatusPendingCode  = 0 // 待审核
	ApprovalStatusApprovedCode = 1 // 审核通过
	ApprovalStatusRejectedCode = 2 // 审核拒绝

	// 审核目标类型（对应 audit_records.target_type）
	AuditTargetVolunteer int32 = 1 // 志愿者实名审核
	AuditTargetOrg       int32 = 2 // 组织资质审核
	AuditTargetMember    int32 = 3 // 志愿者加入组织审核
	AuditTargetSignup    int32 = 4 // 活动报名审核

	// 审核结果（对应 audit_records.audit_result）
	AuditResultPass   int32 = 1 // 通过
	AuditResultReject int32 = 2 // 驳回

	// 审核通用状态（用于当前审核目标）
	AuditStatusPending  int32 = 1 // 待审核
	AuditStatusApproved int32 = 2 // 审核通过
	AuditStatusRejected int32 = 3 // 审核拒绝

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

// GetRegisterTypeString 根据注册类型数字代码返回对应的字符串
func GetRegisterTypeString(registerTypeCode int) string {
	switch registerTypeCode {
	case RegisterTypeVolunteerCode:
		return RegisterTypeVolunteer
	case RegisterTypeOrganizationCode:
		return RegisterTypeOrganization
	default:
		return "" // 未知代码返回空字符串
	}
}

// GetApprovalStatusCode 根据审核状态字符串返回对应的数字代码
func GetApprovalStatusCode(approvalStatus string) int {
	switch approvalStatus {
	case ApprovalStatusPending:
		return ApprovalStatusPendingCode
	case ApprovalStatusApproved:
		return ApprovalStatusApprovedCode
	case ApprovalStatusRejected:
		return ApprovalStatusRejectedCode
	default:
		return -1 // 未知状态返回-1
	}
}

// GetApprovalStatusString 根据审核状态数字代码返回对应的字符串
func GetApprovalStatusString(approvalStatusCode int) string {
	switch approvalStatusCode {
	case ApprovalStatusPendingCode:
		return ApprovalStatusPending
	case ApprovalStatusApprovedCode:
		return ApprovalStatusApproved
	case ApprovalStatusRejectedCode:
		return ApprovalStatusRejected
	default:
		return "" // 未知代码返回空字符串
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
	return auditResult == AuditResultPass || auditResult == AuditResultReject
}

// ResolveAuditStatus 根据审核结果计算目标状态
func ResolveAuditStatus(auditResult int32) int32 {
	if auditResult == AuditResultPass {
		return AuditStatusApproved
	}
	return AuditStatusRejected
}
