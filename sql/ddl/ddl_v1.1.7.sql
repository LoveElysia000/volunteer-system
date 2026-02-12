-- ============================================
-- DDL Version: v1.1.7
-- Description: Iteration 3 - work-hour and activity-result closed loop
-- Created: 2026-02-12
-- ============================================

-- 1) 扩展 activity_signups：补齐签退与工时结算字段。
ALTER TABLE `activity_signups`
    ADD COLUMN `check_out_status` TINYINT NOT NULL DEFAULT 0 COMMENT '签退状态：0-未签退，1-已签退' AFTER `check_in_time`,
    ADD COLUMN `check_out_time` DATETIME NULL COMMENT '签退时间' AFTER `check_out_status`,
    ADD COLUMN `work_hour_status` TINYINT NOT NULL DEFAULT 0 COMMENT '工时结算状态：0-未结算，1-已发放，2-已作废' AFTER `check_out_time`,
    ADD COLUMN `work_hour_version` BIGINT NOT NULL DEFAULT 0 COMMENT '工时结算版本号（用于重算）' AFTER `work_hour_status`,
    ADD COLUMN `last_work_hour_log_id` BIGINT NOT NULL DEFAULT 0 COMMENT '最后一次生效的工时流水ID' AFTER `work_hour_version`,
    ADD COLUMN `granted_hours` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '本次报名最终发放工时' AFTER `last_work_hour_log_id`,
    ADD COLUMN `granted_at` DATETIME NULL COMMENT '工时发放时间' AFTER `granted_hours`;

ALTER TABLE `activity_signups`
    ADD INDEX `idx_signup_work_hour_status` (`work_hour_status`, `updated_at`),
    ADD INDEX `idx_signup_work_hour_log_id` (`last_work_hour_log_id`);

-- 2) 新建 work_hour_logs（使用新结构）。
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
