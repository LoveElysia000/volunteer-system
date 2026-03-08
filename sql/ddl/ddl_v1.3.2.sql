-- ============================================
-- DDL Version: v1.3.2
-- Description: add rbac change log table
-- Created: 2026-03-08
-- ============================================

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
