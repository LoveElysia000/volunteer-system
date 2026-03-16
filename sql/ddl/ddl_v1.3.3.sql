-- ============================================
-- DDL Version: v1.3.3
-- Description: membership leave metadata fields
-- Created: 2026-03-13
-- ============================================

-- 1) 组织成员表新增退出时间与退出原因字段。
ALTER TABLE `org_members`
    ADD COLUMN `left_at` DATETIME NULL COMMENT '退出时间' AFTER `joined_at`,
    ADD COLUMN `leave_reason` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '退出原因' AFTER `left_at`;
