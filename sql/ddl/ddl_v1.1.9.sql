-- ============================================
-- DDL Version: v1.1.9
-- Description: add attendance code fields for activities (plain text now, hash reserved)
-- Created: 2026-02-14
-- ============================================

ALTER TABLE `activities`
    ADD COLUMN `check_in_code` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '签到码' AFTER `status`,
    ADD COLUMN `check_in_code_hash` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '签到码哈希' AFTER `check_in_code`,
    ADD COLUMN `check_in_code_expire_at` DATETIME NULL COMMENT '签到码过期时间' AFTER `check_in_code_hash`,
    ADD COLUMN `check_out_code` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '签退码' AFTER `check_in_code_expire_at`,
    ADD COLUMN `check_out_code_hash` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '签退码哈希' AFTER `check_out_code`,
    ADD COLUMN `check_out_code_expire_at` DATETIME NULL COMMENT '签退码过期时间' AFTER `check_out_code_hash`,
    ADD COLUMN `attendance_code_version` BIGINT NOT NULL DEFAULT 0 COMMENT '签到签退码版本号（每次重置+1）' AFTER `check_out_code_expire_at`,
    ADD COLUMN `attendance_code_updated_at` DATETIME NULL COMMENT '签到签退码最后更新时间' AFTER `attendance_code_version`;
