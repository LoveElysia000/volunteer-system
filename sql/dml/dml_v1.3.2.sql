-- ============================================
-- DML Version: v1.3.2
-- Description: seed rbac.manage permission and super_admin mapping
-- Created: 2026-03-08
-- ============================================

-- 1) Seed permission points for RBAC administration
INSERT INTO `rbac_permissions` (`resource`, `action`, `description`, `status`) VALUES
  ('rbac', 'manage', 'Manage RBAC roles, permissions, and grants', 1)
ON DUPLICATE KEY UPDATE
  `description` = VALUES(`description`),
  `status` = VALUES(`status`),
  `updated_at` = CURRENT_TIMESTAMP;

-- 2) Ensure super_admin owns rbac.manage
INSERT INTO `rbac_role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `rbac_roles` r
JOIN `rbac_permissions` p
  ON p.resource = 'rbac' AND p.action = 'manage'
WHERE r.role_code = 'super_admin'
ON DUPLICATE KEY UPDATE
  `updated_at` = CURRENT_TIMESTAMP;

-- 3) Optional hardening: remove rbac.manage from non-super-admin roles if exists
DELETE rp
FROM `rbac_role_permissions` rp
JOIN `rbac_roles` r ON r.id = rp.role_id
JOIN `rbac_permissions` p ON p.id = rp.permission_id
WHERE p.resource = 'rbac'
  AND p.action = 'manage'
  AND r.role_code <> 'super_admin';

-- 4) Hardening: super_admin only owns rbac.manage (remove business permissions)
DELETE rp
FROM `rbac_role_permissions` rp
JOIN `rbac_roles` r ON r.id = rp.role_id
JOIN `rbac_permissions` p ON p.id = rp.permission_id
WHERE r.role_code = 'super_admin'
  AND NOT (p.resource = 'rbac' AND p.action = 'manage');
