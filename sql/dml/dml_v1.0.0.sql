-- ============================================
-- DML Version: v1.0.0
-- Description: unique index fix and duplicate data check
-- Scope: sys_accounts
-- ============================================
-- Behavior:
--   - Index add statements are one-time operations.
--   - If index already exists, database may return duplicate-key-name error; can be ignored.
--   - SELECT statements are verification queries and do not modify data.

-- 检查并添加手机号哈希唯一索引
ALTER TABLE `sys_accounts` ADD UNIQUE KEY `uk_mobile_hash` (`mobile_hash`) COMMENT '手机号哈希去重索引';
-- 如果报错索引已存在，说明已存在，可以忽略该错误

-- 检查并添加邮箱唯一索引
ALTER TABLE `sys_accounts` ADD UNIQUE KEY `uk_email` (`email`) COMMENT '邮箱去重索引';
-- 如果报错索引已存在，说明已存在，可以忽略该错误

-- ============================================
-- 检查重复数据
-- ============================================

-- 查找手机号哈希重复的记录
SELECT mobile_hash, COUNT(*) as cnt
FROM sys_accounts
WHERE deleted_at IS NULL
GROUP BY mobile_hash
HAVING cnt > 1;

-- 查找邮箱重复的记录
SELECT email, COUNT(*) as cnt
FROM sys_accounts
WHERE deleted_at IS NULL
GROUP BY email
HAVING cnt > 1;
