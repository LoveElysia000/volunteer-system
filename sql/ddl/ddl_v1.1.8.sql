-- ============================================
-- DDL Version: v1.1.8
-- Description: add audit creator field for signup dedup
-- Created: 2026-02-13
-- ============================================

ALTER TABLE `audit_records`
    ADD COLUMN `creator_id` BIGINT NOT NULL DEFAULT 0 COMMENT '提交人账号ID(关联sys_accounts.id)' AFTER `target_id`;
