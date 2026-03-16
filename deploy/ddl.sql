-- ============================================
-- volunteer-system 全量建表脚本（当前代码使用）
-- 维护位置: deploy/ddl.sql
-- 更新时间: 2026-03-08
-- ============================================

-- 1) 系统账号表
CREATE TABLE IF NOT EXISTS `sys_accounts` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `username` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '用户名',
  `mobile` VARCHAR(256) NOT NULL DEFAULT '' COMMENT '手机号 (AES-GCM加密后存储)',
  `mobile_hash` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '手机号哈希值 (SHA-256, 用于唯一性检查)',
  `email` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '邮箱 (唯一登录标识)',
  `password` VARCHAR(255) NOT NULL COMMENT '加密后的密码 (建议使用BCrypt)',
  `identity_type` TINYINT NOT NULL COMMENT '身份类型: 1-志愿者, 2-组织管理者',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '账号状态: 0-禁用, 1-正常',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '注册时间',
  `last_login_time` TIMESTAMP NULL COMMENT '最后登录时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_mobile_hash` (`mobile_hash`),
  UNIQUE KEY `uk_email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户账号主表（所有用户的基础登录信息）';

-- 2) 志愿者档案表
CREATE TABLE IF NOT EXISTS `volunteers` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `account_id` BIGINT NOT NULL COMMENT '关联sys_accounts.id',
  `real_name` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '真实姓名',
  `gender` TINYINT NOT NULL DEFAULT 0 COMMENT '性别: 0-未知, 1-男, 2-女',
  `birthday` DATE NULL COMMENT '出生日期',
  `id_card` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '身份证号 (建议AES加密存储)',
  `avatar_url` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '头像URL',
  `introduction` VARCHAR(2000) NOT NULL DEFAULT '' COMMENT '个人简介',
  `total_hours` DECIMAL(10, 1) NOT NULL DEFAULT 0.0 COMMENT '累计服务时长(小时)',
  `total_points` INT NOT NULL DEFAULT 0 COMMENT '累计积分',
  `level_id` INT NOT NULL DEFAULT 1 COMMENT '当前等级ID',
  `service_count` TINYINT NOT NULL DEFAULT 0 COMMENT '累计服务次数',
  `credit_score` SMALLINT NOT NULL DEFAULT 100 COMMENT '信用分(默认100)',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '志愿者状态: 1-活跃, 2-非活跃, 3-暂停',
  `audit_status` TINYINT NOT NULL DEFAULT 0 COMMENT '实名认证状态: 0-未认证, 1-审核中, 2-已通过, 3-已驳回',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_account` (`account_id`),
  KEY `idx_audit_status` (`audit_status`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='志愿者档案表';

-- 3) 组织档案表
CREATE TABLE IF NOT EXISTS `organizations` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `account_id` BIGINT NOT NULL COMMENT '关联sys_accounts.id',
  `org_name` VARCHAR(100) NOT NULL COMMENT '组织全称',
  `license_code` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '统一社会信用代码/组织机构代码',
  `contact_person` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '负责人姓名',
  `contact_phone` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '办公电话 (AES加密后存储)',
  `address` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '办公地址',
  `logo_url` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '组织Logo URL',
  `introduction` VARCHAR(2000) NOT NULL DEFAULT '' COMMENT '组织介绍',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 0-停用, 1-正常',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_account` (`account_id`),
  UNIQUE KEY `uk_license_code` (`license_code`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='组织档案表';

-- 4) 活动主表
CREATE TABLE IF NOT EXISTS `activities` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `org_id` BIGINT NOT NULL DEFAULT 0 COMMENT '发布组织ID (关联organizations.id)',
  `title` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '活动标题',
  `description` TEXT NOT NULL COMMENT '活动描述/副标题',
  `cover_url` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '活动封面图URL',
  `start_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '开始时间',
  `end_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '结束时间',
  `location` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '地点名称',
  `address` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '详细地址',
  `duration` DECIMAL(4, 1) NOT NULL DEFAULT 0.0 COMMENT '预估工时(小时)',
  `max_people` INT NOT NULL DEFAULT 0 COMMENT '最大招募人数 (0表示不限)',
  `current_people` INT NOT NULL DEFAULT 0 COMMENT '当前已报名人数(冗余字段)',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1-报名中, 2-已结束, 3-已取消',
  `check_in_code` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '签到码',
  `check_in_code_hash` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '签到码哈希',
  `check_in_code_expire_at` DATETIME NULL COMMENT '签到码过期时间',
  `check_out_code` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '签退码',
  `check_out_code_hash` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '签退码哈希',
  `check_out_code_expire_at` DATETIME NULL COMMENT '签退码过期时间',
  `attendance_code_version` BIGINT NOT NULL DEFAULT 0 COMMENT '签到签退码版本号（每次重置+1）',
  `attendance_code_updated_at` DATETIME NULL COMMENT '签到签退码最后更新时间',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_status` (`status`),
  KEY `idx_org_status` (`org_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='活动主表';

-- 5) 活动报名记录表
CREATE TABLE IF NOT EXISTS `activity_signups` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `activity_id` BIGINT NOT NULL COMMENT '活动ID (关联activities.id)',
  `volunteer_id` BIGINT NOT NULL COMMENT '志愿者ID (关联volunteers.id)',
  `signup_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '报名时间',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1-待审核, 2-报名成功, 3-报名驳回, 4-已取消',
  `check_in_status` TINYINT NOT NULL DEFAULT 0 COMMENT '签到状态: 0-未签到, 1-已签到',
  `check_in_time` TIMESTAMP NULL DEFAULT NULL COMMENT '签到时间',
  `check_out_status` TINYINT NOT NULL DEFAULT 0 COMMENT '签退状态：0-未签退，1-已签退',
  `check_out_time` DATETIME NULL COMMENT '签退时间',
  `work_hour_status` TINYINT NOT NULL DEFAULT 0 COMMENT '工时结算状态：0-未结算，1-已发放，2-已作废',
  `work_hour_version` BIGINT NOT NULL DEFAULT 0 COMMENT '工时结算版本号（用于重算）',
  `last_work_hour_log_id` BIGINT NOT NULL DEFAULT 0 COMMENT '最后一次生效的工时流水ID',
  `granted_hours` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '本次报名最终发放工时',
  `granted_at` DATETIME NULL COMMENT '工时发放时间',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_act_vol` (`activity_id`, `volunteer_id`),
  KEY `idx_volunteer_id` (`volunteer_id`),
  KEY `idx_act_status` (`activity_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='活动报名记录表';

-- 6) 组织成员关联表
CREATE TABLE IF NOT EXISTS `org_members` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `org_id` BIGINT NOT NULL COMMENT '组织ID (关联organizations.id)',
  `volunteer_id` BIGINT NOT NULL COMMENT '志愿者ID (关联volunteers.id)',
  `role` INT NOT NULL DEFAULT 1 COMMENT '角色: 1-普通成员, 2-管理员, 3-负责人',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '成员状态: 1-待审核, 2-正式成员, 3-申请驳回, 4-已退出',
  `applied_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '申请时间',
  `joined_at` DATETIME NULL COMMENT '正式加入时间',
  `left_at` DATETIME NULL COMMENT '退出时间',
  `leave_reason` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '退出原因',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_org_volunteer` (`org_id`, `volunteer_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='组织成员关联表';

-- 7) 志愿工时流水表
CREATE TABLE IF NOT EXISTS `work_hour_logs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `volunteer_id` BIGINT NOT NULL COMMENT '志愿者ID（关联 volunteers.id）',
  `activity_id` BIGINT NOT NULL COMMENT '活动ID（关联 activities.id）',
  `signup_id` BIGINT NOT NULL COMMENT '报名ID（关联 activity_signups.id）',
  `operation_type` TINYINT NOT NULL DEFAULT 1 COMMENT '操作类型：1-发放，2-作废，3-重发',
  `hours_delta` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '工时增量（作废时可为负数）',
  `service_count_delta` BIGINT NOT NULL DEFAULT 0 COMMENT '服务次数增量',
  `before_total_hours` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '变更前累计工时',
  `after_total_hours` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '变更后累计工时',
  `before_service_count` BIGINT NOT NULL DEFAULT 0 COMMENT '变更前累计服务次数',
  `after_service_count` BIGINT NOT NULL DEFAULT 0 COMMENT '变更后累计服务次数',
  `work_hour_version` BIGINT NOT NULL DEFAULT 0 COMMENT '结算版本号（与报名表保持一致）',
  `idempotency_key` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '幂等键（防重复发放/作废/重发）',
  `ref_log_id` BIGINT NOT NULL DEFAULT 0 COMMENT '关联原流水ID（作废/重发场景）',
  `reason` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '作废或重发原因',
  `operator_id` BIGINT NOT NULL DEFAULT 0 COMMENT '操作人账号ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_idempotency_key` (`idempotency_key`),
  KEY `idx_whl_volunteer_created` (`volunteer_id`, `created_at`),
  KEY `idx_whl_signup_version` (`signup_id`, `work_hour_version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='志愿工时流水表';

-- 8) 通用审核记录表
CREATE TABLE IF NOT EXISTS `audit_records` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `target_type` TINYINT NOT NULL COMMENT '审核类型: 1-志愿者实名, 2-组织资质, 3-加入组织申请, 4-活动报名',
  `target_id` BIGINT NOT NULL COMMENT '关联目标表的主键ID',
  `creator_id` BIGINT NOT NULL DEFAULT 0 COMMENT '提交人账号ID(关联sys_accounts.id)',
  `auditor_id` BIGINT NOT NULL DEFAULT 0 COMMENT '审核人账号ID(关联sys_accounts.id)',
  `old_content` TEXT NOT NULL COMMENT '变更前数据快照(JSON形式)',
  `new_content` TEXT NOT NULL COMMENT '变更后数据快照(JSON形式)',
  `audit_result` TINYINT NOT NULL DEFAULT 0 COMMENT '审核结论: 1-通过, 2-驳回',
  `reject_reason` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '驳回原因/备注',
  `audit_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '审核时间',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `operation_type` TINYINT NOT NULL DEFAULT 0 COMMENT '操作类型: 1-新增, 2-更新, 3-删除',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '审核状态: 1-待审核, 2-已通过, 3-已驳回',
  PRIMARY KEY (`id`),
  KEY `idx_target_status` (`target_type`, `status`, `id`),
  KEY `idx_creator_status` (`creator_id`, `status`, `target_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通用审核记录表';

-- 9) 志愿者等级规则表
CREATE TABLE IF NOT EXISTS `level_rules` (
  `level` INT NOT NULL COMMENT '等级ID（从1开始）',
  `name` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '等级名称',
  `threshold_hours` INT NOT NULL DEFAULT 0 COMMENT '达到该等级所需累计工时',
  PRIMARY KEY (`level`),
  KEY `idx_level_threshold_hours` (`threshold_hours`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='志愿者等级规则表';

-- 10) 志愿者积分/工时变动流水表
CREATE TABLE IF NOT EXISTS `records` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `volunteer_id` BIGINT NOT NULL COMMENT '志愿者ID（关联 volunteers.id）',
  `type` ENUM('POINT','HOUR') NOT NULL COMMENT '流水类型：POINT-积分，HOUR-工时',
  `amount` DECIMAL(10,1) NOT NULL DEFAULT 0.0 COMMENT '变动值，可正可负',
  `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '变动时间',
  PRIMARY KEY (`id`),
  KEY `idx_records_volunteer_time` (`volunteer_id`, `create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='志愿者积分/工时变动流水表';

-- 11) 通知内容主表
CREATE TABLE IF NOT EXISTS `notifications` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `event_type` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '事件类型: activity_created/activity_updated/member_join',
  `biz_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '业务类型: activity/membership/organization',
  `biz_id` BIGINT NOT NULL DEFAULT 0 COMMENT '业务ID',
  `source_org_id` BIGINT NOT NULL DEFAULT 0 COMMENT '来源组织ID(用于退出组织后归档)',
  `sender_id` BIGINT NOT NULL DEFAULT 0 COMMENT '发送方账号ID(关联sys_accounts.id)',
  `dedupe_key` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '事件幂等键',
  `title` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '通知标题',
  `content` VARCHAR(2000) NOT NULL DEFAULT '' COMMENT '通知正文',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_dedupe_key` (`dedupe_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通知内容主表';

-- 12) 用户通知收件箱
CREATE TABLE IF NOT EXISTS `notification_inbox` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `notification_id` BIGINT NOT NULL COMMENT '通知内容ID(关联 notifications.id)',
  `receiver_id` BIGINT NOT NULL COMMENT '接收人账号ID(关联 sys_accounts.id)',
  `source_org_id` BIGINT NOT NULL DEFAULT 0 COMMENT '来源组织ID(冗余,便于按组织归档)',
  `read_status` TINYINT NOT NULL DEFAULT 0 COMMENT '读取状态: 0-未读, 1-已读',
  `read_at` DATETIME NULL COMMENT '读取时间',
  `inbox_status` TINYINT NOT NULL DEFAULT 1 COMMENT '收件箱状态: 1-normal, 2-archived, 3-user_deleted',
  `archived_reason` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '归档原因: left_org/system_clean/user_action',
  `archived_at` DATETIME NULL COMMENT '归档时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '入箱时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_receiver_notification` (`receiver_id`, `notification_id`),
  KEY `idx_receiver_status_read_created` (`receiver_id`, `inbox_status`, `read_status`, `created_at`, `id`),
  KEY `idx_receiver_source_org_status` (`receiver_id`, `source_org_id`, `inbox_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户通知收件箱';

-- 13) AI 助手会话表
CREATE TABLE IF NOT EXISTS `ai_sessions` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` BIGINT NOT NULL COMMENT '用户ID（关联 sys_accounts.id）',
  `scene` VARCHAR(32) NOT NULL DEFAULT 'general' COMMENT '会话场景：general/activity_draft/ops_advisor',
  `title` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '会话标题',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1-活跃，2-归档',
  `summary` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '会话摘要（可选）',
  `last_message_at` DATETIME NULL COMMENT '最后一条消息时间',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` DATETIME NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 助手会话表';

-- 14) AI 助手消息表
CREATE TABLE IF NOT EXISTS `ai_messages` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `session_id` BIGINT NOT NULL COMMENT '会话ID（关联 ai_sessions.id）',
  `seq_no` INT NOT NULL COMMENT '会话内消息序号（从1递增）',
  `role` TINYINT NOT NULL COMMENT '角色：1-system，2-user，3-assistant，4-tool',
  `content` LONGTEXT NOT NULL COMMENT '消息内容',
  `model` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '模型名称',
  `finish_reason` TINYINT NOT NULL DEFAULT 0 COMMENT '结束原因：0-未知，1-stop，2-length，3-content_filter，4-tool_calls',
  `token_in` INT NOT NULL DEFAULT 0 COMMENT '输入token数',
  `token_out` INT NOT NULL DEFAULT 0 COMMENT '输出token数',
  `latency_ms` INT NOT NULL DEFAULT 0 COMMENT '模型调用耗时（毫秒）',
  `request_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '链路追踪ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_messages_session_seq` (`session_id`, `seq_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 助手消息表';

-- 15) AI 工具调用日志表
CREATE TABLE IF NOT EXISTS `ai_tool_calls` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `session_id` BIGINT NOT NULL COMMENT '会话ID（关联 ai_sessions.id）',
  `message_id` BIGINT NOT NULL DEFAULT 0 COMMENT '触发工具调用的 assistant 消息ID（关联 ai_messages.id）',
  `tool_name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '工具名称',
  `tool_input` LONGTEXT NOT NULL COMMENT '工具输入参数（JSON字符串）',
  `tool_output` LONGTEXT NULL COMMENT '工具输出结果（JSON字符串）',
  `success` TINYINT NOT NULL DEFAULT 0 COMMENT '是否成功：0-失败，1-成功',
  `error_code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '错误码',
  `error_msg` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '错误信息',
  `latency_ms` INT NOT NULL DEFAULT 0 COMMENT '工具执行耗时（毫秒）',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 工具调用日志表';

-- 16) AI 每日用量汇总表
CREATE TABLE IF NOT EXISTS `ai_usage_daily` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `biz_date` DATE NOT NULL COMMENT '业务日期',
  `user_id` BIGINT NOT NULL COMMENT '用户ID（关联 sys_accounts.id）',
  `request_count` BIGINT NOT NULL DEFAULT 0 COMMENT '请求总数',
  `success_count` BIGINT NOT NULL DEFAULT 0 COMMENT '成功请求数',
  `failed_count` BIGINT NOT NULL DEFAULT 0 COMMENT '失败请求数',
  `token_in_total` BIGINT NOT NULL DEFAULT 0 COMMENT '输入token汇总',
  `token_out_total` BIGINT NOT NULL DEFAULT 0 COMMENT '输出token汇总',
  `estimated_cost` DECIMAL(12,4) NOT NULL DEFAULT 0.0000 COMMENT '预估成本（按模型单价计算）',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ai_usage_daily_biz_user` (`biz_date`, `user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 每日用量汇总表';

-- 17) RBAC 角色表
CREATE TABLE IF NOT EXISTS `rbac_roles` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `role_code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '角色编码（唯一）',
  `role_name` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '角色名称',
  `description` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '角色描述',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1-启用，0-禁用',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rbac_roles_role_code` (`role_code`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='RBAC 角色定义表';

-- 18) RBAC 权限点表
CREATE TABLE IF NOT EXISTS `rbac_permissions` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `resource` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '资源标识',
  `action` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '动作标识',
  `description` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '权限描述',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1-启用，0-禁用',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rbac_permissions_resource_action` (`resource`, `action`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='RBAC 权限点定义表';

-- 19) RBAC 角色-权限关联表
CREATE TABLE IF NOT EXISTS `rbac_role_permissions` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `role_id` BIGINT NOT NULL DEFAULT 0 COMMENT '角色ID（关联 rbac_roles.id）',
  `permission_id` BIGINT NOT NULL DEFAULT 0 COMMENT '权限ID（关联 rbac_permissions.id）',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rbac_role_permissions_role_permission` (`role_id`, `permission_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='RBAC 角色权限映射表';

-- 20) RBAC 账号-角色-作用域关联表
CREATE TABLE IF NOT EXISTS `rbac_account_roles` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `account_id` BIGINT NOT NULL DEFAULT 0 COMMENT '账号ID（关联 sys_accounts.id）',
  `role_id` BIGINT NOT NULL DEFAULT 0 COMMENT '角色ID（关联 rbac_roles.id）',
  `scope_type` VARCHAR(16) NOT NULL DEFAULT 'org' COMMENT '作用域类型：global/org',
  `scope_id` BIGINT NOT NULL DEFAULT 0 COMMENT '作用域ID（global 固定为0，org 为 organizations.id）',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1-启用，0-禁用',
  `granted_by` BIGINT NOT NULL DEFAULT 0 COMMENT '授权人账号ID',
  `expires_at` DATETIME NULL COMMENT '过期时间（为空表示不过期）',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_rbac_account_roles_binding` (`account_id`, `role_id`, `scope_type`, `scope_id`),
  KEY `idx_rbac_account_roles_scope` (`scope_type`, `scope_id`),
  KEY `idx_rbac_account_roles_account_status` (`account_id`, `status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='RBAC 账号角色授权表';

-- 21) RBAC 变更审计日志表
CREATE TABLE IF NOT EXISTS `rbac_change_logs` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `operator_id` BIGINT NOT NULL DEFAULT 0 COMMENT '操作人账号ID',
  `target_account_id` BIGINT NOT NULL DEFAULT 0 COMMENT '目标账号ID',
  `target_role_id` BIGINT NOT NULL DEFAULT 0 COMMENT '目标角色ID',
  `scope_type` VARCHAR(16) NOT NULL DEFAULT '' COMMENT '作用域类型：global/org',
  `scope_id` BIGINT NOT NULL DEFAULT 0 COMMENT '作用域ID',
  `change_type` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '变更类型',
  `before_value` LONGTEXT NOT NULL COMMENT '变更前数据(JSON)',
  `after_value` LONGTEXT NOT NULL COMMENT '变更后数据(JSON)',
  `remark` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '备注',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_rbac_change_logs_operator` (`operator_id`),
  KEY `idx_rbac_change_logs_scope` (`scope_type`, `scope_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='RBAC 变更审计日志';
