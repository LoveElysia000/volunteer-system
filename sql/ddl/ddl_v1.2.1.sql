-- ============================================
-- DDL Version: v1.2.1
-- Description: notification center core tables and optional outbox
-- Created: 2026-02-28
-- ============================================

-- 1) 通知内容主表
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
    UNIQUE KEY `uk_dedupe_key` (`dedupe_key`),
    KEY `idx_biz` (`biz_type`, `biz_id`),
    KEY `idx_source_org_created` (`source_org_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通知内容主表';

-- 2) 用户收件箱（核心表）
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
    KEY `idx_inbox_list` (`receiver_id`, `inbox_status`, `created_at`, `id`),
    KEY `idx_receiver_status_read_created` (`receiver_id`, `inbox_status`, `read_status`, `created_at`, `id`),
    KEY `idx_receiver_source_org_status` (`receiver_id`, `source_org_id`, `inbox_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户通知收件箱';

-- 3) 通知事件投递箱（可选增强，按需启用）
CREATE TABLE IF NOT EXISTS `notification_outbox` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `event_key` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '幂等键',
    `event_type` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '事件类型',
    `biz_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '业务类型',
    `biz_id` BIGINT NOT NULL DEFAULT 0 COMMENT '业务ID',
    `source_org_id` BIGINT NOT NULL DEFAULT 0 COMMENT '来源组织ID',
    `payload` JSON NOT NULL COMMENT '事件负载(JSON)',
    `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态: 0-pending,1-processing,2-success,3-retry_wait,4-dead',
    `retry_count` INT NOT NULL DEFAULT 0 COMMENT '重试次数',
    `next_retry_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '下次重试时间',
    `last_error` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '最后错误',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `processed_at` DATETIME NULL COMMENT '处理完成时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_event_key` (`event_key`),
    KEY `idx_status_retry` (`status`, `next_retry_at`, `id`),
    KEY `idx_biz` (`biz_type`, `biz_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通知事件投递箱';
