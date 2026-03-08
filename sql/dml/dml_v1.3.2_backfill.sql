-- ============================================
-- DML Version: v1.3.2-backfill
-- Description: backfill default RBAC grants for existing accounts
-- Created: 2026-03-08
-- ============================================

-- 1) Backfill volunteer -> volunteer(global)
INSERT INTO `rbac_account_roles` (
  `account_id`, `role_id`, `scope_type`, `scope_id`, `status`, `granted_by`
)
SELECT
  v.account_id,
  r.id,
  'global',
  0,
  1,
  0
FROM `volunteers` v
JOIN `rbac_roles` r ON r.role_code = 'volunteer'
WHERE v.account_id > 0
ON DUPLICATE KEY UPDATE
  `status` = VALUES(`status`),
  `updated_at` = CURRENT_TIMESTAMP;

-- 2) Backfill organization account -> org_owner(org)
INSERT INTO `rbac_account_roles` (
  `account_id`, `role_id`, `scope_type`, `scope_id`, `status`, `granted_by`
)
SELECT
  o.account_id,
  r.id,
  'org',
  o.id,
  1,
  0
FROM `organizations` o
JOIN `rbac_roles` r ON r.role_code = 'org_owner'
WHERE o.account_id > 0
ON DUPLICATE KEY UPDATE
  `status` = VALUES(`status`),
  `updated_at` = CURRENT_TIMESTAMP;

-- 3) Bootstrap guide (manual): choose first super admin account
-- Example:
-- INSERT INTO rbac_account_roles(account_id, role_id, scope_type, scope_id, status, granted_by)
-- SELECT 1, id, 'global', 0, 1, 1 FROM rbac_roles WHERE role_code='super_admin'
-- ON DUPLICATE KEY UPDATE status=1, updated_at=CURRENT_TIMESTAMP;
