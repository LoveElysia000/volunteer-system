-- 1. 组织档案表新增状态字段
ALTER TABLE `organizations`
ADD COLUMN `status` TINYINT NOT NULL DEFAULT 1
COMMENT '状态: 0-停用, 1-正常'
AFTER `audit_status`;

-- 2. 状态字段索引（便于后台筛选）
ALTER TABLE `organizations`
ADD INDEX `idx_status` (`status`);

-- 3. 移除组织审核状态字段（请先确认代码已不再依赖 organizations.audit_status）
ALTER TABLE `organizations`
DROP INDEX `idx_name_audit`,
DROP INDEX `idx_audit_status`,
DROP COLUMN `audit_status`;
