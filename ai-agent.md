# 环保志愿者平台：独立 AI 助手模块集成方案（重写版）

## 1. 背景与目标

当前文档以“文案生成、推荐”等零散能力为主，不利于后续扩展与统一治理。
本次改造目标是：将 AI 能力从业务细节中抽离，建设一个可复用、可审计、可扩展的独立 AI 助手模块（Assistant Domain），由该模块统一对外提供智能服务。

### 1.1 目标

1. 建立统一入口：所有 AI 能力统一走 `/api/assistant/*`。
2. 建立统一会话模型：支持“多轮对话 + 业务工具调用 + 历史追踪”。
3. 建立统一安全与审计：权限校验、敏感信息脱敏、调用日志留痕。
4. 保持业务可演进：后续新增 AI 场景无需改动核心业务模块结构。

### 1.2 非目标（当前阶段不做）

1. 不引入复杂 Agent 框架（LangGraph / AutoGen 等）。
2. 不做多模型动态路由编排（先单主模型 + 可切换 Provider）。
3. 不做向量数据库集群（先 MySQL + 可选 Redis 缓存，后续再升级）。

---

## 2. 模块定位（从“功能补丁”到“独立子系统”）

AI 助手模块独立为新领域，不挂靠在 `activities` 或其他单一业务模块下。

### 2.1 模块职责

1. 会话管理：创建会话、上下文裁剪、历史查询。
2. LLM 调用：统一封装 Provider（OpenAI/DeepSeek/其他兼容接口）。
3. 工具调用（Tool Calling）：AI 通过受控工具访问业务能力。
4. 输出治理：统一响应结构、错误码、审计日志。

### 2.2 对外能力（MVP）

1. 智能问答：平台使用说明、活动相关咨询。
2. 活动发布助手：生成活动草案（标题、简介、流程、风险提示）。
3. 组织运营助手：根据组织近期活动给出优化建议。

---

## 3. 总体架构（贴合当前项目分层）

```text
Client
  -> /api/assistant/* (router)
      -> handler/assistant.go
          -> service/assistant_service.go
              -> service/assistant_tool_service.go
              -> pkg/ai/client.go
              -> repository/assistant_repo.go
                  -> MySQL / Redis
```

### 3.1 分层说明

1. `router`：只做路由注册与鉴权接入。
2. `handler`：参数校验、统一响应包装。
3. `service`：对话编排、工具执行、上下文窗口管理。
4. `repository`：会话、消息、调用日志持久化。
5. `pkg/ai`：AI SDK 客户端封装（含超时、重试、限流控制）。

---

## 4. 数据库建表设计（完整 DDL）

建议新增 DDL 版本：`sql/ddl/ddl_v1.3.0.sql`。  
以下 SQL 可直接作为初版迁移脚本。

### 4.1 建表 SQL（可直接执行）

```sql
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
    KEY `idx_ai_sessions_user_scene_status` (`user_id`, `scene`, `status`),
    KEY `idx_ai_sessions_user_updated` (`user_id`, `updated_at`),
    KEY `idx_ai_sessions_last_message_at` (`last_message_at`)
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
    UNIQUE KEY `uk_ai_messages_session_seq` (`session_id`, `seq_no`),
    KEY `idx_ai_messages_session_created` (`session_id`, `created_at`),
    KEY `idx_ai_messages_role_created` (`role`, `created_at`)
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
    KEY `idx_ai_tool_calls_session_created` (`session_id`, `created_at`),
    KEY `idx_ai_tool_calls_tool_success` (`tool_name`, `success`, `created_at`),
    KEY `idx_ai_tool_calls_message_id` (`message_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 工具调用日志表';

-- 4) AI 每日用量汇总表
CREATE TABLE IF NOT EXISTS `ai_usage_daily` (
    `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `biz_date` DATE NOT NULL COMMENT '业务日期',
    `user_id` BIGINT NOT NULL COMMENT '用户ID（关联 sys_accounts.id）',
    `request_count` INT NOT NULL DEFAULT 0 COMMENT '请求总数',
    `success_count` INT NOT NULL DEFAULT 0 COMMENT '成功请求数',
    `failed_count` INT NOT NULL DEFAULT 0 COMMENT '失败请求数',
    `token_in_total` BIGINT NOT NULL DEFAULT 0 COMMENT '输入token汇总',
    `token_out_total` BIGINT NOT NULL DEFAULT 0 COMMENT '输出token汇总',
    `estimated_cost` DECIMAL(12,4) NOT NULL DEFAULT 0.0000 COMMENT '预估成本（按模型单价计算）',
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_ai_usage_daily_biz_user` (`biz_date`, `user_id`),
    KEY `idx_ai_usage_daily_user_date` (`user_id`, `biz_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='AI 每日用量汇总表';

```

### 4.2 索引与查询场景对应

1. 会话列表查询：`idx_ai_sessions_user_updated` 支持“按用户分页 + 最近更新时间排序”。
2. 历史消息拉取：`idx_ai_messages_session_created` 支持“按会话拉取时间线”。
3. 工具调用排障：`idx_ai_tool_calls_tool_success` 支持“按工具名 + 成功率 + 时间窗统计”。
4. 用量报表统计：`uk_ai_usage_daily_biz_user` 保证每日单用户唯一聚合，避免重复累加。

### 4.3 约束策略（不使用外键）

1. 统一不使用数据库外键（FK）。
2. 通过应用层保证引用完整性（会话删除前先清理消息与工具日志）。
3. 通过唯一索引和业务校验保证数据一致性。

### 4.4 回滚 SQL（紧急撤回）

```sql
DROP TABLE IF EXISTS `ai_usage_daily`;
DROP TABLE IF EXISTS `ai_tool_calls`;
DROP TABLE IF EXISTS `ai_messages`;
DROP TABLE IF EXISTS `ai_sessions`;
```

---

## 5. API 设计（独立助手入口）

统一前缀：`/api/assistant`

### 5.1 创建会话

- `POST /api/assistant/sessions`
- 入参：`scene`, `title(optional)`
- 出参：`session_id`

### 5.2 对话交互

- `POST /api/assistant/chat`
- 入参：`session_id`, `message`, `stream(optional)`
- 出参：`reply`, `tool_calls`, `usage`

### 5.3 历史消息

- `GET /api/assistant/sessions/:id/messages`
- 出参：消息列表（按时间升序）

### 5.4 快捷能力（可选）

- `POST /api/assistant/actions/activity-draft`
- 用于前端“生成活动草案”按钮，内部仍走统一会话机制。

---

## 6. 工具调用设计（受控能力开放）

AI 不直接访问数据库，必须通过受控工具层调用业务服务。

### 6.1 首批工具

1. `activity_search`
- 功能：检索活动（按关键词、时间范围、状态）。
- 权限：登录用户可用，仅返回可见字段。

2. `activity_stats`
- 功能：统计组织活动数量、参与人数、完结率。
- 权限：组织管理员及以上。

3. `activity_draft_generate`
- 功能：根据主题/目标人群/地点生成发布草案。
- 权限：组织成员（含管理员）。

### 6.2 工具执行约束

1. 每轮最多 3 次工具调用。
2. 单工具超时 3 秒，失败可重试 1 次。
3. 工具返回统一 JSON Schema，防止模型误解结构。

---

## 7. 配置设计（新增 `ai` 配置块）

需在 `config/config.go` 增加 `AIConfig`，并在 `config/config.yaml` 增加配置项：

```yaml
ai:
  enabled: true
  provider: "openai"        # openai / deepseek / compatible
  api_key: "${AI_API_KEY:}"
  base_url: "https://api.openai.com/v1"
  chat_model: "gpt-4o-mini"
  embedding_model: "text-embedding-3-small"
  request_timeout_ms: 15000
  max_retries: 2
  max_context_messages: 20
  daily_user_quota: 200
```

说明：`api_key` 必须走环境变量，不落库、不写日志。

---

## 8. 代码落地清单（按当前仓库结构）

### 8.1 新增文件

1. `pkg/ai/client.go`
2. `pkg/ai/types.go`
3. `internal/router/assistant.go`
4. `internal/handler/assistant.go`
5. `internal/service/assistant_service.go`
6. `internal/service/assistant_tool_service.go`
7. `internal/repository/assistant.go`
8. `sql/ddl/ddl_v1.3.0.sql`
9. `internal/api/assistant.proto`（如继续使用 proto 驱动接口）

### 8.2 修改文件

1. `config/config.go`（新增 `AIConfig`）
2. `config/config.yaml` / `config/config.yaml.example`（新增 `ai` 节）
3. `internal/router/router.go`（注册 `RegisterAssistantRouter`）
4. `internal/service/service.go`（按需注入 assistant service 依赖）

---

## 9. 分阶段实施计划（以“模块上线”为核心）

### Phase A（1 天）：基础能力可用

1. 完成配置、`pkg/ai`、基础 chat 接口。
2. 支持创建会话 + 单轮对话。
3. 完成最小日志与错误处理。

验收：能通过 `/api/assistant/chat` 返回稳定答案。

### Phase B（1~2 天）：会话与审计完善

1. 落地 `ai_sessions` / `ai_messages`。
2. 支持历史查询、上下文裁剪。
3. 增加 usage 统计与限流。

验收：多轮对话可追踪，可复盘。

### Phase C（1~2 天）：工具调用接入业务

1. 接入 `activity_search`、`activity_stats`、`activity_draft_generate`。
2. 模型可通过工具返回结构化事实，再组织自然语言回答。

验收：AI 助手可以完成“查活动 + 给建议 + 生成草案”闭环。

### Phase D（0.5~1 天）：质量与上线

1. 压测、超时与降级策略验证。
2. 补充 OpenAPI 文档与示例。
3. 灰度发布并观察指标。

验收：达到上线阈值并可灰度放量。

---

## 10. 安全、合规与稳定性要求

1. 权限隔离：工具调用前必须做业务权限校验，不信任模型指令。
2. Prompt 注入防护：系统提示词中禁止执行越权请求，工具层二次拦截。
3. 敏感信息保护：对手机号、身份证、邮箱做脱敏再入日志。
4. 限流熔断：按用户 + IP 双限流，外部模型异常时快速降级。
5. 可观测性：记录 `request_id`、模型耗时、token、工具成功率。

---

## 11. 与原“零碎功能”方案的关系

1. 文案生成：升级为 `activity_draft_generate` 工具，不再单独散落。
2. 推荐能力：先作为助手工具中的“建议生成”能力，向量检索可后置接入。
3. 旧接口兼容：若已存在 `/ai/generate-desc`，保留一段过渡期，内部转发到助手模块。

---

## 12. 验收标准（上线门槛）

1. 功能可用：会话、聊天、历史、至少 2 个工具调用稳定可用。
2. 质量达标：P95 响应时间 < 3s（不含超长输出场景）。
3. 可治理：调用日志完整，异常可追踪到用户与请求。
4. 可扩展：新增助手场景无需改动核心路由与 SDK 封装。

---

## 13. 下一步开发顺序（建议）

1. 先落地 `pkg/ai` + `assistant chat` 基础链路。
2. 再补数据库表与会话持久化。
3. 最后接业务工具与前端入口。

这样可以最短路径上线“可用的独立 AI 助手”，并在后续迭代中持续增强。
