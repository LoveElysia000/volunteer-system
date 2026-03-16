-- ============================================
-- DML Version: v1.3.2-rbac
-- Description: consolidated RBAC seed script (empty database)
-- Created: 2026-03-15
-- ============================================
-- Prerequisite:
--   - Run ddl_v1.3.1 first so RBAC tables exist.
-- Goal:
--   - Keep all RBAC initialization statements in a single file.
-- Behavior:
--   - Designed for empty database initialization.
--   - No backfill and no cleanup statements are included.

-- 1) Seed roles
INSERT INTO `rbac_roles` (`role_code`, `role_name`, `description`, `status`) VALUES
  ('super_admin', 'Super Admin', 'Platform-wide governance role', 1),
  ('org_owner', 'Organization Owner', 'Organization owner role', 1),
  ('volunteer', 'Volunteer', 'Volunteer default role', 1);

-- 2) Seed permission points (business + RBAC governance)
INSERT INTO `rbac_permissions` (`resource`, `action`, `description`, `status`) VALUES
  ('organization', 'manage', 'Manage organization profile and status', 1),
  ('membership', 'manage', 'Manage membership status and role', 1),
  ('audit', 'review', 'Review audit records and take audit actions', 1),
  ('export', 'manage', 'Manage report and data export', 1),
  ('analytics', 'org.read', 'Read organization analytics dashboard', 1),
  ('rbac', 'manage', 'Manage RBAC roles, permissions, and grants', 1);

-- 3) Seed org_owner business role-permission mappings
INSERT INTO `rbac_role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `rbac_roles` r
JOIN `rbac_permissions` p
  ON r.role_code = 'org_owner'
 AND (p.resource, p.action) IN (
   ('organization', 'manage'),
   ('membership', 'manage'),
   ('audit', 'review'),
   ('export', 'manage'),
   ('analytics', 'org.read')
 );

-- 4) Ensure super_admin owns rbac.manage
INSERT INTO `rbac_role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id
FROM `rbac_roles` r
JOIN `rbac_permissions` p
  ON p.resource = 'rbac' AND p.action = 'manage'
WHERE r.role_code = 'super_admin';

-- 5) Bootstrap guide (manual): choose first super admin account
-- Example:
-- INSERT INTO rbac_account_roles(account_id, role_id, scope_type, scope_id, status, granted_by)
-- SELECT 1, id, 'global', 0, 1, 1 FROM rbac_roles WHERE role_code='super_admin'
-- ON DUPLICATE KEY UPDATE status=1, updated_at=CURRENT_TIMESTAMP;
