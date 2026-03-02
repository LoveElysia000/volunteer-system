-- ============================================
-- DDL Version: v1.3.0
-- Description: add AI assistant session/message/tool usage tables
-- Created: 2026-03-02
-- ============================================

-- 1) AI 会话表
CREATE TABLE IF NOT EXISTS `ai_sessions` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `user_id` BIGINT NOT NULL COMMENT '用户ID（关联 sys_accounts.id）',
    `scene` VARCHAR(32) NOT NULL DEFAULT 'general' COMMENT '会话场景：general/activity_draft/ops_advisor',
    `title` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '会话标题',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1-活跃，2-归档',
    `summary` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '会话摘要（可选）',
    `last_message_at` DATETIME NULL COMMENT '最后一条消息时间',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` DATETIME NULL COMMENT '软删除时间',
    PRIMARY KEY (`id`),
    KEY `idx_ai_sessions_user_updated` (`user_id`, `updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 助手会话表';

-- 2) AI 消息表
CREATE TABLE IF NOT EXISTS `ai_messages` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `session_id` BIGINT NOT NULL COMMENT '会话ID（关联 ai_sessions.id）',
    `seq_no` INT NOT NULL COMMENT '会话内消息序号（从1递增）',
    `role` TINYINT NOT NULL COMMENT '角色：1-system，2-user，3-assistant，4-tool',
    `content` LONGTEXT NOT NULL COMMENT '消息内容',
    `model` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '模型名称',
    `finish_reason` TINYINT NOT NULL DEFAULT 0 COMMENT '结束原因：0-未知，1-stop，2-length，3-content_filter，4-tool_calls',
    `token_in` INT NOT NULL DEFAULT 0 COMMENT '输入token数',
    `token_out` INT NOT NULL DEFAULT 0 COMMENT '输出token数',
    `latency_ms` INT NOT NULL DEFAULT 0 COMMENT '模型调用耗时（毫秒）',
    `request_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '链路追踪ID',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_ai_messages_session_seq` (`session_id`, `seq_no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 助手消息表';

-- 3) AI 工具调用日志表
CREATE TABLE IF NOT EXISTS `ai_tool_calls` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `session_id` BIGINT NOT NULL COMMENT '会话ID（关联 ai_sessions.id）',
    `message_id` BIGINT NOT NULL DEFAULT 0 COMMENT '触发工具调用的 assistant 消息ID（关联 ai_messages.id）',
    `tool_name` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '工具名称',
    `tool_input` LONGTEXT NOT NULL COMMENT '工具输入参数（JSON字符串）',
    `tool_output` LONGTEXT NULL COMMENT '工具输出结果（JSON字符串）',
    `success` TINYINT NOT NULL DEFAULT 0 COMMENT '是否成功：0-失败，1-成功',
    `error_code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '错误码',
    `error_msg` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '错误信息',
    `latency_ms` INT NOT NULL DEFAULT 0 COMMENT '工具执行耗时（毫秒）',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    PRIMARY KEY (`id`),
    KEY `idx_ai_tool_calls_session_created` (`session_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 工具调用日志表';

-- 4) AI 每日用量汇总表
CREATE TABLE IF NOT EXISTS `ai_usage_daily` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `biz_date` DATE NOT NULL COMMENT '业务日期',
    `user_id` BIGINT NOT NULL COMMENT '用户ID（关联 sys_accounts.id）',
    `request_count` BIGINT NOT NULL DEFAULT 0 COMMENT '请求总数',
    `success_count` BIGINT NOT NULL DEFAULT 0 COMMENT '成功请求数',
    `failed_count` BIGINT NOT NULL DEFAULT 0 COMMENT '失败请求数',
    `token_in_total` BIGINT NOT NULL DEFAULT 0 COMMENT '输入token汇总',
    `token_out_total` BIGINT NOT NULL DEFAULT 0 COMMENT '输出token汇总',
    `estimated_cost` DECIMAL(12,4) NOT NULL DEFAULT 0.0000 COMMENT '预估成本（按模型单价计算）',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_ai_usage_daily_biz_user` (`biz_date`, `user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 每日用量汇总表';

