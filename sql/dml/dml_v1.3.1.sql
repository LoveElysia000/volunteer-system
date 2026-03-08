-- ============================================
-- DML Version: v1.3.1
-- Description: seed RBAC roles, permissions and role-permission bindings
-- Created: 2026-03-08
-- ============================================

-- 1) Seed roles
INSERT INTO `rbac_roles` (`role_code`, `role_name`, `description`, `status`) VALUES
  ('super_admin', 'Super Admin', 'Platform-wide governance role', 1),
  ('org_owner', 'Organization Owner', 'Organization owner role', 1),
  ('org_manager', 'Organization Manager', 'Organization manager role', 1),
  ('volunteer', 'Volunteer', 'Volunteer default role', 1)
ON DUPLICATE KEY UPDATE
  `role_name` = VALUES(`role_name`),
  `description` = VALUES(`description`),
  `status` = VALUES(`status`),
  `updated_at` = CURRENT_TIMESTAMP;

-- 2) Seed permission points
INSERT INTO `rbac_permissions` (`resource`, `action`, `description`, `status`) VALUES
  ('organization', 'manage', 'Manage organization profile and status', 1),
  ('membership', 'manage', 'Manage membership status and role', 1),
  ('audit', 'review', 'Review audit records and take audit actions', 1),
  ('export', 'manage', 'Manage report and data export', 1),
  ('analytics', 'org.read', 'Read organization analytics dashboard', 1)
ON DUPLICATE KEY UPDATE
  `description` = VALUES(`description`),
  `status` = VALUES(`status`),
  `updated_at` = CURRENT_TIMESTAMP;

-- 3) Seed role-permission mappings (idempotent)
-- super_admin only owns rbac.manage (seeded in dml_v1.3.2)
INSERT INTO `rbac_role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `rbac_roles` r
JOIN `rbac_permissions` p
  ON (r.role_code IN ('org_owner', 'org_manager'))
 AND (p.resource, p.action) IN (
   ('organization', 'manage'),
   ('membership', 'manage'),
   ('audit', 'review'),
   ('export', 'manage'),
   ('analytics', 'org.read')
 )
ON DUPLICATE KEY UPDATE
  `updated_at` = CURRENT_TIMESTAMP;
