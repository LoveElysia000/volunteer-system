-- ============================================
-- DML Version: v1.3.3-demo-core
-- Description: minimal demo seed data for core business tables
-- Created: 2026-03-29
-- ============================================
-- Scope:
--   - Excludes all RBAC / role related tables
--   - Only seeds core tables required by main activity flow
-- Tables:
--   - sys_accounts
--   - volunteers
--   - organizations
--   - activities
--   - activity_signups

SET NAMES utf8mb4;

-- 1) Accounts
INSERT INTO `sys_accounts`
  (`id`, `user_name`, `mobile`, `mobile_hash`, `email`, `password`, `identity_type`, `status`, `created_at`, `last_login_time`)
VALUES
  (1001, 'org_admin_green', 'enc_mobile_13800000001', 'mh_13800000001', 'org_green@example.com', '$2a$10$demoHashedPasswordValue000000000000000000001', 2, 1, '2026-03-01 09:00:00', '2026-03-29 08:30:00'),
  (1002, 'org_admin_sun',   'enc_mobile_13800000002', 'mh_13800000002', 'org_sun@example.com',   '$2a$10$demoHashedPasswordValue000000000000000000002', 2, 1, '2026-03-02 09:30:00', '2026-03-29 09:00:00'),
  (2001, 'vol_li_lei',      'enc_mobile_13900000001', 'mh_13900000001', 'li.lei@example.com',    '$2a$10$demoHashedPasswordValue000000000000000000101', 1, 1, '2026-03-03 10:00:00', '2026-03-29 07:40:00'),
  (2002, 'vol_han_mei',     'enc_mobile_13900000002', 'mh_13900000002', 'han.mei@example.com',   '$2a$10$demoHashedPasswordValue000000000000000000102', 1, 1, '2026-03-03 10:10:00', '2026-03-28 21:10:00'),
  (2003, 'vol_chen_xiao',   'enc_mobile_13900000003', 'mh_13900000003', 'chen.xiao@example.com', '$2a$10$demoHashedPasswordValue000000000000000000103', 1, 1, '2026-03-04 11:00:00', '2026-03-29 06:55:00'),
  (2004, 'vol_wang_min',    'enc_mobile_13900000004', 'mh_13900000004', 'wang.min@example.com',  '$2a$10$demoHashedPasswordValue000000000000000000104', 1, 1, '2026-03-05 14:20:00', '2026-03-27 19:45:00')
ON DUPLICATE KEY UPDATE
  `user_name` = VALUES(`user_name`),
  `mobile` = VALUES(`mobile`),
  `mobile_hash` = VALUES(`mobile_hash`),
  `email` = VALUES(`email`),
  `password` = VALUES(`password`),
  `identity_type` = VALUES(`identity_type`),
  `status` = VALUES(`status`),
  `last_login_time` = VALUES(`last_login_time`);

-- 2) Volunteers
INSERT INTO `volunteers`
  (`id`, `account_id`, `real_name`, `gender`, `birthday`, `id_card`, `avatar_url`, `introduction`, `total_hours`, `total_points`, `level_id`, `service_count`, `credit_score`, `status`, `audit_status`, `created_at`)
VALUES
  (3001, 2001, '李雷',   1, '1999-04-12', 'enc_id_3001', 'https://example.com/avatar/li-lei.png',   '长期参与社区敬老志愿服务。', 12.0, 120, 1, 3, 100, 1, 2, '2026-03-03 10:05:00'),
  (3002, 2002, '韩梅梅', 2, '2000-08-19', 'enc_id_3002', 'https://example.com/avatar/han-mei.png',  '擅长活动组织与现场引导。', 28.5, 260, 2, 6, 100, 1, 2, '2026-03-03 10:15:00'),
  (3003, 2003, '陈晓',   1, '1998-11-03', 'enc_id_3003', 'https://example.com/avatar/chen-xiao.png','喜欢青少年陪伴类公益活动。', 54.0, 500, 3, 10, 98, 1, 2, '2026-03-04 11:05:00'),
  (3004, 2004, '王敏',   2, '2001-01-26', 'enc_id_3004', 'https://example.com/avatar/wang-min.png', '新注册志愿者。',             0.0,   0, 1, 0, 100, 1, 1, '2026-03-05 14:25:00')
ON DUPLICATE KEY UPDATE
  `account_id` = VALUES(`account_id`),
  `real_name` = VALUES(`real_name`),
  `gender` = VALUES(`gender`),
  `birthday` = VALUES(`birthday`),
  `id_card` = VALUES(`id_card`),
  `avatar_url` = VALUES(`avatar_url`),
  `introduction` = VALUES(`introduction`),
  `total_hours` = VALUES(`total_hours`),
  `total_points` = VALUES(`total_points`),
  `level_id` = VALUES(`level_id`),
  `service_count` = VALUES(`service_count`),
  `credit_score` = VALUES(`credit_score`),
  `status` = VALUES(`status`),
  `audit_status` = VALUES(`audit_status`);

-- 3) Organizations
INSERT INTO `organizations`
  (`id`, `account_id`, `org_name`, `license_code`, `contact_person`, `contact_phone`, `address`, `logo_url`, `introduction`, `status`, `created_at`)
VALUES
  (4001, 1001, '青禾社区志愿服务中心', '91500101MADEMOORG01', '周老师', 'enc_phone_4001', '上海市浦东新区云台路88号', 'https://example.com/logo/qinghe.png', '聚焦社区助老和便民服务。', 1, '2026-03-01 09:10:00'),
  (4002, 1002, '暖阳青年公益协会',     '91500101MADEMOORG02', '林老师', 'enc_phone_4002', '上海市杨浦区政学路66号', 'https://example.com/logo/nuanyang.png', '组织青年志愿者开展导览和赛事服务。', 1, '2026-03-02 09:40:00')
ON DUPLICATE KEY UPDATE
  `account_id` = VALUES(`account_id`),
  `org_name` = VALUES(`org_name`),
  `license_code` = VALUES(`license_code`),
  `contact_person` = VALUES(`contact_person`),
  `contact_phone` = VALUES(`contact_phone`),
  `address` = VALUES(`address`),
  `logo_url` = VALUES(`logo_url`),
  `introduction` = VALUES(`introduction`),
  `status` = VALUES(`status`);

-- 4) Activities
INSERT INTO `activities`
  (`id`, `org_id`, `title`, `description`, `cover_url`, `start_time`, `end_time`, `location`, `address`, `duration`, `max_people`, `current_people`, `status`, `check_in_code`, `check_in_code_hash`, `check_in_code_expire_at`, `check_out_code`, `check_out_code_hash`, `check_out_code_expire_at`, `attendance_code_version`, `attendance_code_updated_at`, `created_at`)
VALUES
  (5001, 4001, '社区长者陪伴日', '为长者提供陪伴聊天和基础手机使用辅导。', 'https://example.com/activity/5001.jpg', '2026-04-03 09:00:00', '2026-04-03 12:00:00', '青禾社区服务站', '上海市浦东新区云台路88号一楼活动室', 3.0, 20, 2, 1, 'IN5001', 'hash_in_5001', '2026-04-03 10:00:00', 'OUT5001', 'hash_out_5001', '2026-04-03 12:30:00', 1, '2026-03-29 08:00:00', '2026-03-25 10:00:00'),
  (5002, 4001, '周末河道环保行动', '沿社区河道开展垃圾清理与环保宣传。',       'https://example.com/activity/5002.jpg', '2026-04-06 08:30:00', '2026-04-06 11:30:00', '滨江步道集合点', '上海市浦东新区滨江大道近民生路口', 3.0, 30, 1, 1, 'IN5002', 'hash_in_5002', '2026-04-06 09:30:00', 'OUT5002', 'hash_out_5002', '2026-04-06 12:00:00', 1, '2026-03-29 08:10:00', '2026-03-26 09:00:00'),
  (5003, 4002, '城市书展导览服务', '为参展市民提供路线咨询和秩序引导。',         'https://example.com/activity/5003.jpg', '2026-03-28 13:00:00', '2026-03-28 17:00:00', '市民文化中心', '上海市杨浦区政学路100号', 4.0, 15, 1, 2, 'IN5003', 'hash_in_5003', '2026-03-28 14:00:00', 'OUT5003', 'hash_out_5003', '2026-03-28 17:30:00', 1, '2026-03-28 12:00:00', '2026-03-20 12:00:00')
ON DUPLICATE KEY UPDATE
  `org_id` = VALUES(`org_id`),
  `title` = VALUES(`title`),
  `description` = VALUES(`description`),
  `cover_url` = VALUES(`cover_url`),
  `start_time` = VALUES(`start_time`),
  `end_time` = VALUES(`end_time`),
  `location` = VALUES(`location`),
  `address` = VALUES(`address`),
  `duration` = VALUES(`duration`),
  `max_people` = VALUES(`max_people`),
  `current_people` = VALUES(`current_people`),
  `status` = VALUES(`status`),
  `check_in_code` = VALUES(`check_in_code`),
  `check_in_code_hash` = VALUES(`check_in_code_hash`),
  `check_in_code_expire_at` = VALUES(`check_in_code_expire_at`),
  `check_out_code` = VALUES(`check_out_code`),
  `check_out_code_hash` = VALUES(`check_out_code_hash`),
  `check_out_code_expire_at` = VALUES(`check_out_code_expire_at`),
  `attendance_code_version` = VALUES(`attendance_code_version`),
  `attendance_code_updated_at` = VALUES(`attendance_code_updated_at`);

-- 5) Activity signups
INSERT INTO `activity_signups`
  (`id`, `activity_id`, `volunteer_id`, `signup_time`, `status`, `check_in_status`, `check_in_time`, `check_out_status`, `check_out_time`, `work_hour_status`, `work_hour_version`, `last_work_hour_log_id`, `granted_hours`, `granted_at`, `created_at`)
VALUES
  (7001, 5001, 3001, '2026-03-27 09:00:00', 2, 0, NULL, 0, NULL, 0, 0, 0, 0.00, NULL, '2026-03-27 09:00:00'),
  (7002, 5001, 3002, '2026-03-27 09:10:00', 2, 0, NULL, 0, NULL, 0, 0, 0, 0.00, NULL, '2026-03-27 09:10:00'),
  (7003, 5002, 3004, '2026-03-28 19:30:00', 1, 0, NULL, 0, NULL, 0, 0, 0, 0.00, NULL, '2026-03-28 19:30:00'),
  (7004, 5003, 3003, '2026-03-22 13:00:00', 2, 1, '2026-03-28 12:50:00', 1, '2026-03-28 17:05:00', 1, 1, 0, 4.00, '2026-03-28 17:20:00', '2026-03-22 13:00:00')
ON DUPLICATE KEY UPDATE
  `signup_time` = VALUES(`signup_time`),
  `status` = VALUES(`status`),
  `check_in_status` = VALUES(`check_in_status`),
  `check_in_time` = VALUES(`check_in_time`),
  `check_out_status` = VALUES(`check_out_status`),
  `check_out_time` = VALUES(`check_out_time`),
  `work_hour_status` = VALUES(`work_hour_status`),
  `work_hour_version` = VALUES(`work_hour_version`),
  `last_work_hour_log_id` = VALUES(`last_work_hour_log_id`),
  `granted_hours` = VALUES(`granted_hours`),
  `granted_at` = VALUES(`granted_at`);
