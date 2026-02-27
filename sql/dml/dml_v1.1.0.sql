-- ============================================
-- 初始化 level_rules 等级规则（可重复执行）
-- ============================================

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
