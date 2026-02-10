-- 1. 志愿者档案表新增状态字段
ALTER TABLE `volunteers`
ADD COLUMN `status` TINYINT NOT NULL DEFAULT 1
COMMENT '志愿者状态: 1-活跃, 2-非活跃, 3-暂停'
AFTER `credit_score`;

-- 2. 状态字段索引（便于后台筛选）
ALTER TABLE `volunteers`
ADD INDEX `idx_status` (`status`);
