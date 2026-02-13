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

	// 志愿者状态
	VolunteerActiveStatus   int32 = 1 // 活跃
	VolunteerInactiveStatus int32 = 2 // 非活跃
	VolunteerEtcStatus      int32 = 3 // 其他

	// 志愿者认证状态（volunteers.audit_status）
	VolunteerAuditStatusUnverified int32 = 0 // 未认证
	VolunteerAuditStatusPending    int32 = 1 // 审核中
	VolunteerAuditStatusApproved   int32 = 2 // 已通过
	VolunteerAuditStatusRejected   int32 = 3 // 已驳回

	// 活动报名状态
	ActivitySignupStatusPending  int32 = 1 // 待审核
	ActivitySignupStatusSuccess  int32 = 2 // 报名成功
	ActivitySignupStatusRejected int32 = 3 // 报名驳回
	ActivitySignupStatusCanceled int32 = 4 // 已取消

	// 活动状态（activities.status）
	ActivityStatusRecruiting int32 = 1 // 报名中
	ActivityStatusFinished   int32 = 2 // 已结束
	ActivityStatusCanceled   int32 = 3 // 已取消

	// 活动签到/签退状态（activity_signups）
	ActivityCheckInPending  int32 = 0 // 未签到
	ActivityCheckInDone     int32 = 1 // 已签到
	ActivityCheckOutPending int32 = 0 // 未签退
	ActivityCheckOutDone    int32 = 1 // 已签退

	// 工时结算状态（activity_signups.work_hour_status）
	WorkHourStatusPending int32 = 0 // 未结算
	WorkHourStatusGranted int32 = 1 // 已发放
	WorkHourStatusVoided  int32 = 2 // 已作废

	// 工时流水操作类型（work_hour_logs.operation_type）
	WorkHourOperationGrant   int32 = 1 // 发放
	WorkHourOperationVoid    int32 = 2 // 作废
	WorkHourOperationRegrant int32 = 3 // 重发/重算

	// 账号状态
	SysAccountNotNormal int32 = 0 // 禁用
	SysAccountNormal    int32 = 1 // 正常
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
