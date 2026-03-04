# AI 助手 Eino 集成实施文档

更新时间：2026-03-04  
适用项目：`volunteer-system`

## 1. 背景与目标

当前项目已具备独立 AI 助手模块能力：

1. 独立 API：`/api/assistant/*`
2. 独立服务编排：`assistant_service + assistant_tool_service`
3. 独立持久化：`ai_sessions / ai_messages / ai_tool_calls / ai_usage_daily`
4. 独立模型访问层：`pkg/ai/client.go`

本实施文档目标是：在不破坏现有业务 API 与数据库模型的前提下，引入 Eino 作为可选运行时，支持后续更复杂的 Agent 编排能力。

## 2. 范围与非目标

### 2.1 本期范围

1. 引入 Eino 运行时并可开关切换（`native` / `eino`）。
2. 复用现有工具能力（活动检索/统计/草案）。
3. 保持现有落库与观测逻辑。
4. 支持失败回退到现有 native 逻辑。

### 2.2 非目标

1. 不重构现有 API 协议（`assistant.proto` 保持兼容）。
2. 不更改现有 DDL 结构。
3. 不在本期引入向量库或长期记忆系统。
4. `stream` 先不强制上线（可预留）。

## 3. 现状基线

当前核心链路：

1. `internal/service/assistant_service.go`：会话管理、工具调用、模型调用、消息落库。
2. `internal/service/assistant_tool_service.go`：规则规划 + 受控工具执行。
3. `pkg/ai/client.go`：多 provider chat completions 调用。
4. `internal/repository/assistant.go`：assistant 域数据访问。

现状特点：

1. 工具规划为规则启发式（非自治多步 Agent）。
2. 模型失败有 fallback 文本回复。
3. 已具备 token 用量与工具调用日志。

## 4. 目标架构（增量改造）

```text
Client
  -> /api/assistant/*
      -> handler/assistant.go
          -> service/assistant_service.go
              -> AssistantRuntime (interface)
                 |- NativeRuntime (当前逻辑)
                 |- EinoRuntime  (新实现)
              -> assistant_tool_service.go (工具能力源)
              -> repository/assistant.go   (落库保持不变)
```

设计原则：

1. 先抽象运行时，再接入 Eino。
2. API、DB、日志模型保持不变。
3. 通过配置灰度切换，支持秒级回滚。

## 5. 实施阶段

## Phase 0：准备与依赖校验（0.5 天）

1. 引入 Eino 相关依赖（`github.com/cloudwego/eino` 及对应 provider 扩展）。
2. 本地执行 `go mod tidy`，确认与 Hertz/GORM 依赖无冲突。
3. 记录基线接口行为（`CreateSession/Chat/SessionMessages/ActivityDraftAction`）。

交付：依赖可编译，现有功能无回归。

## Phase 1：运行时抽象（1 天）

新增接口（建议文件：`internal/service/assistant_runtime.go`）：

```go
type AssistantRuntime interface {
    Chat(ctx context.Context, in *RuntimeChatInput) (*RuntimeChatOutput, error)
    Name() string
}
```

新增结构：

1. `RuntimeChatInput`：scene、message、history、userID、sessionID、requestID。
2. `RuntimeChatOutput`：reply、model、finishReason、tokenIn/out、toolCalls、latency。
3. `RuntimeToolCall`：tool_name/input/output/success/error/latency。

改造点：

1. 将 `assistant_service.go` 中“工具 + 模型调用”段落下沉到 runtime。
2. 保留 `appendAiMessage`、`UpdateAiToolCallMessageID`、`UpsertAiUsageDaily` 逻辑不变。

交付：默认仍走 `NativeRuntime`，行为与当前一致。

## Phase 2：EinoRuntime 落地（1.5~2 天）

新增文件建议：

1. `internal/service/assistant_runtime_eino.go`
2. `internal/service/assistant_tool_eino_adapter.go`

关键实现：

1. 初始化 ChatModel（DeepSeek/OpenAI，按现有 `ai` 配置映射）。
2. 通过工具适配器把现有 3 个工具注册为 Eino Tool。
3. 创建 `ChatModelAgent` + `Runner`。
4. 执行 Query，提取最终回复与工具调用过程。
5. 将 Eino 输出映射为 `RuntimeChatOutput`。

工具适配约束：

1. 工具内部继续做权限校验（不能依赖模型判断权限）。
2. 工具输入输出仍使用 JSON 字符串形态，方便复用当前审计表。
3. 单次对话设置最大步骤上限（建议 3~5）避免失控循环。

交付：`runtime=eino` 可稳定完成三类场景。

## Phase 3：灰度、降级与回滚（0.5~1 天）

1. 新增配置开关：默认 `runtime=native`。
2. 在 `AssistantService` 统一选择 runtime。
3. `EinoRuntime` 调用失败时可自动 fallback 到 `NativeRuntime`。
4. 记录 runtime 维度日志：成功率、耗时、工具失败率。

交付：生产可灰度启用 Eino，异常可快速切回 native。

## Phase 4：可选增强（后续迭代）

1. 流式输出（`stream=true`）按 Eino stream 能力实现。
2. 工具注册中心（去除硬编码规划）。
3. 日配额拦截（当前仅统计未拦截）。

## 6. 配置设计

在现有 `ai` 配置下新增：

```yaml
ai:
  enabled: true
  provider: "deepseek"
  api_key: "${AI_API_KEY:}"
  base_url: "https://api.deepseek.com/v1"
  chat_model: "deepseek-chat"
  request_timeout_ms: 15000
  max_retries: 2
  max_context_messages: 20

  runtime: "native"     # native / eino
  runtime_fallback: true # eino失败是否回退native

  eino:
    max_steps: 4
    tool_timeout_ms: 3000
    enable_stream: false
```

说明：

1. `runtime` 默认建议 `native`，灰度阶段再切 `eino`。
2. 保持 `api_key` 使用环境变量，不写日志。

## 7. 文件改造清单

### 7.1 新增文件

1. `internal/service/assistant_runtime.go`
2. `internal/service/assistant_runtime_native.go`
3. `internal/service/assistant_runtime_eino.go`
4. `internal/service/assistant_tool_eino_adapter.go`
5. `task/eino-assistant-integration-implementation.md`（本文档）

### 7.2 修改文件

1. `config/config.go`：扩展 AIConfig（runtime/eino配置）。
2. `config/config.yaml.example`：补充示例配置。
3. `config/config.yaml`：本地环境配置补齐。
4. `internal/service/assistant_service.go`：改为 runtime 驱动。

### 7.3 不改文件

1. `internal/api/assistant.proto`（本期保持协议兼容）。
2. `internal/repository/assistant.go`（持久化接口继续复用）。
3. `sql/ddl/ddl_v1.3.0.sql`（本期不新增表）。

## 8. 关键实现细节

### 8.1 工具调用日志映射

Eino 的工具事件需映射到现有 `ai_tool_calls`：

1. `tool_name` <- Eino tool 名称
2. `tool_input` <- 请求参数 JSON
3. `tool_output` <- 响应 JSON
4. `success/error_code/error_msg/latency_ms` <- 运行结果

要求：即使工具失败，也要记录调用日志。

### 8.2 消息序号与并发安全

继续沿用现有逻辑：

1. `(session_id, seq_no)` 唯一键 + 重试。
2. runtime 不直接写库，由 `assistant_service.go` 统一落库。

### 8.3 错误分类建议

1. `MODEL_TIMEOUT`
2. `MODEL_RATE_LIMIT`
3. `TOOL_TIMEOUT`
4. `PERMISSION_DENIED`
5. `RUNTIME_EXEC_FAILED`

## 9. 测试计划

## 9.1 单元测试

1. runtime 选择逻辑（`native/eino/fallback`）。
2. Eino 输出到 `RuntimeChatOutput` 的映射正确性。
3. 工具适配层的权限拒绝与超时行为。

## 9.2 集成测试

覆盖 3 条主链路：

1. 活动检索问答（`activity_search`）
2. 运营分析问答（`activity_stats`）
3. 活动草案生成（`activity_draft_generate`）

每条链路验证：

1. reply 非空
2. tool_calls 完整
3. `ai_messages / ai_tool_calls / ai_usage_daily` 有落库

## 9.3 回归检查

1. 非 AI 模块接口不受影响。
2. `runtime=native` 时行为与改造前一致。

## 10. 发布与回滚

发布策略：

1. 先上线代码，默认 `runtime=native`。
2. 小流量环境切 `runtime=eino` 验证 24 小时。
3. 观察错误率/耗时/工具成功率后逐步放量。

回滚策略：

1. 配置切回 `runtime=native`（无需回滚代码）。
2. 若依赖异常，临时关闭 `ai.enabled`。

## 11. 风险与应对

1. 依赖兼容风险：固定版本并在 CI 做 `go test` + smoke。
2. 工具调用失控：限制 `max_steps` + 单工具超时。
3. 权限绕过风险：权限校验保留在工具服务内部。
4. 成本波动：沿用日用量统计，后续增加日配额拦截。

## 12. 验收标准（DoD）

1. `runtime=native` 和 `runtime=eino` 均可完成对话主流程。
2. 三类工具场景稳定可用，失败可追踪。
3. 关键表落库完整且字段语义一致。
4. 配置可灰度、可回滚。
5. 文档与配置模板同步更新。

## 13. 参考资料（官方）

1. Eino Agent with Tools Quick Start：
   https://cloudwego.cn/docs/eino/quick_start/agent_llm_with_tools/
2. Eino ADK ChatModel Agent：
   https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_implementation/chat_model/
3. Eino ADK 扩展与事件：
   https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_extension/
4. Eino DeepSeek ChatModel：
   https://www.cloudwego.io/docs/eino/ecosystem_integration/chat_model/chat_model_deepseek/
