-- 1. 志愿者表：去掉可能存在的审核备注（如果之前有的话）
ALTER TABLE `volunteers` DROP COLUMN `audit_remark`;

-- 2. 组织档案表：去掉审核相关的详细备注
ALTER TABLE `organizations` DROP COLUMN `audit_remark`;

-- 3. 活动报名表：去掉审核结果、审核人等冗余信息
-- 以后这些信息通过 audit_records 表关联获取
ALTER TABLE `activity_signups` 
    DROP COLUMN `reject_reason`,
    DROP COLUMN `auditor_id`,
    DROP COLUMN `audit_time`;

-- 4. 组织成员表：去掉审核备注
ALTER TABLE `org_members` DROP COLUMN `audit_remark`;

CREATE TABLE `audit_records` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `target_type` TINYINT NOT NULL COMMENT '审核类型: 1-志愿者实名, 2-组织资质, 3-加入组织申请, 4-活动报名',
  `target_id` BIGINT NOT NULL COMMENT '关联目标表的主键ID',
  `auditor_id` BIGINT NOT NULL COMMENT '审核人账号ID(关联sys_accounts.id)',
  `old_status` TINYINT DEFAULT NULL COMMENT '变更前状态',
  `new_status` TINYINT NOT NULL COMMENT '变更后状态',
  `old_content` JSON DEFAULT NULL COMMENT '变更前数据快照(JSON形式)',
  `new_content` JSON DEFAULT NULL COMMENT '变更后数据快照(JSON形式)',
  `audit_result` TINYINT NOT NULL COMMENT '审核结论: 1-通过, 2-驳回',
  `reject_reason` VARCHAR(500) DEFAULT NULL COMMENT '驳回原因/备注',
  `audit_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '审核时间',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  PRIMARY KEY (`id`),
  KEY `idx_target` (`target_type`, `target_id`) COMMENT '目标关联索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通用审核记录表';