-- ============================================
-- 活动主表 (activities)
-- 存储志愿者活动的核心信息
-- ============================================

CREATE TABLE `activities` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `org_id` BIGINT NOT NULL DEFAULT 0 COMMENT '发布组织ID (关联organizations.id)',

  -- 活动基本信息
  `title` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '活动标题',
  `description` TEXT NOT NULL COMMENT '活动描述/副标题',
  `cover_url` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '活动封面图URL',

  -- 时间地点
  `start_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '开始时间',
  `end_time` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '结束时间',
  `location` VARCHAR(100) NOT NULL DEFAULT '' COMMENT '地点名称',
  `address` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '详细地址',

  -- 招募信息
  `duration` DECIMAL(4, 1) NOT NULL DEFAULT '0.0' COMMENT '预估工时(小时)',
  `max_people` INT NOT NULL DEFAULT '0' COMMENT '最大招募人数 (0表示不限)',
  `current_people` INT NOT NULL DEFAULT '0' COMMENT '当前已报名人数(冗余字段)',

  -- 状态
  `status` TINYINT NOT NULL DEFAULT '1' COMMENT '状态: 1-报名中, 2-已结束, 3-已取消',

  -- 系统元数据
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

  PRIMARY KEY (`id`),
  KEY `idx_org_id` (`org_id`) COMMENT '组织ID索引',
  KEY `idx_status` (`status`) COMMENT '状态索引',
  KEY `idx_start_time` (`start_time`) COMMENT '开始时间索引',
  KEY `idx_created_at` (`created_at`) COMMENT '创建时间索引',
  KEY `idx_org_status` (`org_id`, `status`) COMMENT '组织状态联合索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='活动主表';

-- ============================================
-- 活动报名记录表 (activity_signups)
-- 存储志愿者对活动的报名记录
-- ============================================

CREATE TABLE `activity_signups` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `activity_id` BIGINT NOT NULL COMMENT '活动ID (关联activities.id)',
  `volunteer_id` BIGINT NOT NULL COMMENT '志愿者ID (关联volunteers.id)',

  -- 报名信息
  `signup_time` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '报名时间',
  `status` TINYINT NOT NULL DEFAULT '1' COMMENT '状态: 1-已报名, 2-已取消',

  -- 签到信息
  `check_in_status` TINYINT NOT NULL DEFAULT '0' COMMENT '签到状态: 0-未签到, 1-已签到',
  `check_in_time` TIMESTAMP NULL DEFAULT NULL COMMENT '签到时间',

  -- 系统元数据
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_act_vol` (`activity_id`, `volunteer_id`) COMMENT '防止重复报名',
  KEY `idx_activity_id` (`activity_id`) COMMENT '活动ID索引',
  KEY `idx_volunteer_id` (`volunteer_id`) COMMENT '志愿者ID索引',
  KEY `idx_status` (`status`) COMMENT '状态索引',
  KEY `idx_act_status` (`activity_id`, `status`) COMMENT '活动状态联合索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='活动报名记录表';