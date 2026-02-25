-- ============================================
-- DDL Version: v1.2.0
-- Description: dashboard growth and level rules
-- Created: 2026-02-25
-- ============================================

-- 1) 扩展 volunteers：补齐积分与等级字段
ALTER TABLE `volunteers`
    ADD COLUMN `total_points` INT NOT NULL DEFAULT 0 COMMENT '累计积分' AFTER `total_hours`,
    ADD COLUMN `level_id` INT NOT NULL DEFAULT 1 COMMENT '当前等级ID' AFTER `total_points`;

ALTER TABLE `volunteers`
    ADD INDEX `idx_volunteer_level_id` (`level_id`);

-- 2) 新增积分/工时变动流水表（用于月增统计与审计追溯）
CREATE TABLE IF NOT EXISTS `records` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `volunteer_id` BIGINT NOT NULL COMMENT '志愿者ID（关联 volunteers.id）',
    `type` ENUM('POINT','HOUR') NOT NULL COMMENT '流水类型：POINT-积分，HOUR-工时',
    `amount` DECIMAL(10,1) NOT NULL DEFAULT 0.0 COMMENT '变动值，可正可负',
    `create_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '变动时间',
    PRIMARY KEY (`id`),
    KEY `idx_records_volunteer_time` (`volunteer_id`, `create_time`),
    KEY `idx_records_type_time` (`type`, `create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='志愿者积分/工时变动流水表';

-- 3) 新增等级规则表（用于等级进度和升级门槛计算）
CREATE TABLE IF NOT EXISTS `level_rules` (
    `level` INT NOT NULL COMMENT '等级ID（从1开始）',
    `name` VARCHAR(50) NOT NULL DEFAULT '' COMMENT '等级名称',
    `threshold_hours` INT NOT NULL DEFAULT 0 COMMENT '达到该等级所需累计工时',
    PRIMARY KEY (`level`),
    KEY `idx_level_threshold_hours` (`threshold_hours`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='志愿者等级规则表';
