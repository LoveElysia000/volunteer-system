-- ============================================
-- DML Version: v1.1.0
-- Description: seed level_rules
-- Scope: level_rules
-- ============================================
-- Behavior:
--   - Idempotent by ON DUPLICATE KEY UPDATE.
--   - Rerun will synchronize name/threshold_hours with latest seed values.

INSERT INTO `level_rules` (`level`, `name`, `threshold_hours`) VALUES
  (1, 'Lv1',   0),
  (2, 'Lv2',  20),
  (3, 'Lv3',  50),
  (4, 'Lv4', 100),
  (5, 'Lv5', 200),
  (6, 'Lv6', 350),
  (7, 'Lv7', 550),
  (8, 'Lv8', 800)
ON DUPLICATE KEY UPDATE
  `name` = VALUES(`name`),
  `threshold_hours` = VALUES(`threshold_hours`);
