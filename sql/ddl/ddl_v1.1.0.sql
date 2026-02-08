-- ============================================
-- DDL Version: v1.1.0
-- Description: 组织成员关联表和志愿时长流水表
-- Created: 2026-02-05
-- ============================================

-- 组织成员关联表
CREATE TABLE IF NOT EXISTS `org_members` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `org_id` BIGINT NOT NULL COMMENT '组织ID (关联organizations.id)',
    `volunteer_id` BIGINT NOT NULL COMMENT '志愿者ID (关联volunteers.id)',
    `role` INT NOT NULL DEFAULT 1 COMMENT '角色: 1-普通成员, 2-管理员, 3-负责人',
    `status` INT NOT NULL DEFAULT 1 COMMENT '成员状态: 1-待审核, 2-正式成员, 3-已拒绝, 4-已退出',
    `applied_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '申请时间',
    `joined_at` DATETIME NULL COMMENT '正式加入时间',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    INDEX `idx_org_vol` (`org_id`, `volunteer_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='组织成员关联表';

-- 志愿时长流水表
CREATE TABLE IF NOT EXISTS `work_hour_logs` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `volunteer_id` BIGINT NOT NULL COMMENT '志愿者ID',
    `activity_id` BIGINT NOT NULL COMMENT '关联活动ID',
    `org_id` BIGINT NOT NULL COMMENT '发放工时的组织ID',
    `hours` DOUBLE NOT NULL COMMENT '本次获得工时',
    `status` INT NOT NULL DEFAULT 1 COMMENT '状态: 1-待发放, 2-已发放, 3-已作废',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    INDEX `idx_volunteer_id` (`volunteer_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='志愿时长流水表';
