-- ============================================
-- 系统账号表 (sys_accounts)
-- 负责存储所有用户的登录凭证和基础信息
-- ============================================

CREATE TABLE `sys_accounts` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `username` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '用户名',
  `mobile` VARCHAR(256) NOT NULL DEFAULT '' COMMENT '手机号 (AES-GCM加密后存储)',
  `mobile_hash` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '手机号哈希值 (SHA-256, 用于唯一性检查)',
  `email` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '邮箱 (唯一登录标识)',
  `password` VARCHAR(255) NOT NULL COMMENT '加密后的密码 (建议使用BCrypt)',
  `identity_type` TINYINT NOT NULL COMMENT '身份类型: 1-志愿者, 2-组织管理者',

  -- 系统元数据
  `status` TINYINT NOT NULL DEFAULT '1' COMMENT '账号状态: 0-禁用, 1-正常',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '注册时间',
  `last_login_time` TIMESTAMP NULL COMMENT '最后登录时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL COMMENT '软删除时间',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_mobile_hash` (`mobile_hash`) COMMENT '手机号哈希去重索引',
  UNIQUE KEY `uk_email` (`email`) COMMENT '邮箱去重索引',
  KEY `idx_username` (`username`) COMMENT '用户名索引',
  KEY `idx_identity_status` (`identity_type`, `status`) COMMENT '身份状态联合索引',
  KEY `idx_last_login` (`last_login_time`) COMMENT '登录时间索引',
  KEY `idx_created_at` (`created_at`) COMMENT '注册时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户账号主表（所有用户的基础登录信息）';

-- ============================================
-- 志愿者档案表 (volunteers)
-- 仅存储身份为"志愿者"的业务数据
-- 注意：身份证号建议在应用层AES加密后存储
-- ============================================

CREATE TABLE `volunteers` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `account_id` BIGINT UNSIGNED NOT NULL COMMENT '关联sys_accounts.id',

  -- 基础档案
  `real_name` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '真实姓名',
  `gender` TINYINT NOT NULL DEFAULT '0' COMMENT '性别: 0-未知, 1-男, 2-女',
  `birthday` DATE NULL COMMENT '出生日期',
  `id_card` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '身份证号 (建议AES加密存储)',
  `avatar_url` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '头像URL',
  `introduction` VARCHAR(2000) NOT NULL DEFAULT '' COMMENT '个人简介',

  -- 核心统计
  `total_hours` DECIMAL(10, 1) NOT NULL DEFAULT '0.0' COMMENT '累计服务时长(小时)',
  `service_count` TINYINT NOT NULL DEFAULT '0' COMMENT '累计服务次数',
  `credit_score` SMALLINT NOT NULL DEFAULT '100' COMMENT '信用分(默认100)',

  -- 状态
  `audit_status` TINYINT NOT NULL DEFAULT '0' COMMENT '实名认证状态: 0-未认证, 1-审核中, 2-已通过, 3-驳回',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_account` (`account_id`) COMMENT '账号唯一索引',
  KEY `idx_real_name` (`real_name`) COMMENT '姓名索引',
  KEY `idx_audit_status` (`audit_status`) COMMENT '认证状态索引',
  KEY `idx_credit_score` (`credit_score`) COMMENT '信用分索引',
  KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='志愿者档案表';

-- ============================================
-- 组织档案表 (organizations)
-- 仅存储身份为"组织管理者"的业务数据
-- ============================================

CREATE TABLE `organizations` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `account_id` BIGINT UNSIGNED NOT NULL COMMENT '关联sys_accounts.id',

  -- 组织概况
  `org_name` VARCHAR(100) NOT NULL COMMENT '组织全称',
  `license_code` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '统一社会信用代码/组织机构代码',
  `contact_person` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '负责人姓名',
  `contact_phone` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '办公电话 (AES加密后存储)',
  `address` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '办公地址',
  `logo_url` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '组织Logo URL',
  `introduction` VARCHAR(2000) NOT NULL DEFAULT '' COMMENT '组织介绍',

  -- 状态
  `audit_status` TINYINT NOT NULL DEFAULT '0' COMMENT '资质审核状态: 0-未提交, 1-审核中, 2-已通过, 3-驳回',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_account` (`account_id`) COMMENT '账号唯一索引',
  UNIQUE KEY `uk_license_code` (`license_code`) COMMENT '社会信用代码索引',
  KEY `idx_org_name` (`org_name`) COMMENT '组织名称索引',
  KEY `idx_audit_status` (`audit_status`) COMMENT '审核状态索引',
  KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='组织档案表';

-- ============================================
-- 补充索引优化
-- ============================================

-- 为sys_accounts表添加复合索引，优化登录和状态查询
ALTER TABLE `sys_accounts` ADD INDEX `idx_mobile_status` (`mobile`, `status`);
ALTER TABLE `sys_accounts` ADD INDEX `idx_identity_login` (`identity_type`, `last_login_time`);

-- 为volunteers表添加业务查询索引
ALTER TABLE `volunteers` ADD INDEX `idx_gender_audit` (`gender`, `audit_status`);

-- 为organizations表添加业务查询索引
ALTER TABLE `organizations` ADD INDEX `idx_name_audit` (`org_name`, `audit_status`);

-- ============================================
-- 去除 UNSIGNED 限制
-- ============================================

-- 修改 sys_accounts 表
ALTER TABLE `sys_accounts` MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID';

-- 修改 volunteers 表
ALTER TABLE `volunteers` MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID';
ALTER TABLE `volunteers` MODIFY COLUMN `account_id` BIGINT NOT NULL COMMENT '关联sys_accounts.id';

-- 修改 organizations 表
ALTER TABLE `organizations` MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID';
ALTER TABLE `organizations` MODIFY COLUMN `account_id` BIGINT NOT NULL COMMENT '关联sys_accounts.id';
