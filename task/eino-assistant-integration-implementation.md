# AI 助手 Eino 接入实施文档（未来规划）

更新时间：2026-03-05  
适用项目：`volunteer-system`

## 1. 背景与目标

当前项目已有独立 AI 助手域（`/api/assistant/*`、独立 service/repository、独立 AI 表），但核心链路仍以 native 规则编排为主。  
本文件聚焦“后续接入 Eino 框架”所需工作，不描述本轮 native 优化内容。

目标：

1. 引入 Eino 作为可选运行时，支持多步工具调用与中断恢复。
2. 保持现有 API 协议、数据库模型、落库语义不变。
3. 提供可灰度、可回滚的接入方案（`native` 与 `eino` 并存）。

## 2. 范围与边界

### 2.1 本期范围（接入 Eino）

1. 运行时抽象：`AssistantRuntime` 接口化。
2. 新增 `EinoRuntime`，复用现有工具能力。
3. 支持 `Runner Query/Resume` 与 `CheckPointStore`。
4. 工具事件、模型事件映射到现有 `ai_messages / ai_tool_calls / ai_usage_daily`。
5. 配置化灰度：`runtime=native|eino`、`runtime_fallback`。
6. 补齐集成测试、回滚方案、观测指标。

### 2.2 非目标

1. 不改 `internal/api/assistant.proto` 协议结构。
2. 不改 `sql/ddl/ddl_v1.3.0.sql` 表结构（仅复用）。
3. 不在本期引入向量数据库与长期记忆系统。
4. 不做多框架混用（Eino 与 LangChainGo 二选一，本期仅 Eino）。

## 3. 当前基线（供接入评估）

1. 助手主链路：`internal/service/assistant_service.go`
2. 工具层：`internal/service/assistant_tool_service.go`
3. 模型调用：`pkg/ai/client.go`
4. 持久化：`internal/repository/assistant.go`
5. 当前未引入 Eino 依赖（`go.mod` 中无 `cloudwego/eino`）

## 4. 目标架构

```text
Client
  -> /api/assistant/*
      -> handler/assistant.go
          -> service/assistant_service.go
              -> AssistantRuntime (interface)
                 |- NativeRuntime (现有逻辑适配)
                 |- EinoRuntime  (新实现)
              -> assistant_tool_service.go (工具能力源)
              -> repository/assistant.go   (统一落库)
```

设计原则：

1. service 负责会话生命周期与落库一致性；runtime 负责“推理与工具执行”。
2. runtime 不直接写数据库，避免双写路径。
3. Eino 失败可回落到 native。

## 5. 关键设计与待办

### 5.1 运行时接口抽象

新增文件建议：`internal/service/assistant_runtime.go`

```go
type AssistantRuntime interface {
    Name() string
    Chat(ctx context.Context, in *RuntimeChatInput) (*RuntimeChatOutput, error)
}

type RuntimeChatInput struct {
    UserID    int64
    SessionID int64
    Scene     string
    Message   string
    History   []ai.Message
    RequestID string
}

type RuntimeToolCall struct {
    ToolName   string
    InputJSON  string
    OutputJSON string
    Success    bool
    ErrorCode  string
    ErrorMsg   string
    LatencyMS  int32
}

type RuntimeChatOutput struct {
    Reply        string
    Model        string
    FinishReason string
    TokenIn      int32
    TokenOut     int32
    LatencyMS    int32
    ToolCalls    []RuntimeToolCall
}
```

待办：

1. 把现有“工具规划 + 工具执行 + 模型调用”从 `AssistantService.Chat` 下沉到 `NativeRuntime`。
2. `AssistantService` 保留会话校验、消息序列号、工具日志绑定、usage upsert。

### 5.2 EinoRuntime 实现

新增文件建议：

1. `internal/service/assistant_runtime_eino.go`
2. `internal/service/assistant_tool_eino_adapter.go`

待办：

1. 初始化 ChatModel（按现有 `ai.provider/base_url/chat_model` 映射）。
2. 将现有 3 个工具注册为 Eino Tool（不搬业务逻辑，只做适配）。
3. 使用 `adk.Runner` 执行 Query；中断场景支持 Resume。
4. 将 Eino 输出映射为 `RuntimeChatOutput`，保持现有响应兼容。

### 5.3 CheckPointStore 与恢复策略

必须补齐（当前文档缺项）：

1. 指定 `CheckPointStore` 存储（建议 Redis，兜底可 MySQL）。
2. 定义 checkpoint key 规范：`assistant:{session_id}:{request_id}`。
3. 定义 TTL、清理策略、跨实例恢复约束。
4. 明确 `Resume` 入口与超时策略。

### 5.4 工具事件与落库映射（必须细化）

Eino 事件 -> 现有表：

1. ToolStart/ToolEnd -> `ai_tool_calls`
2. Agent 最终回复 -> `ai_messages(role=assistant)`
3. token 使用 -> `ai_usage_daily`

必须增加：

1. 幂等键设计（避免重试导致重复写工具日志）。
2. 失败事件落库规范（工具失败也要记录）。
3. message_id 回填时机与事务边界。

### 5.5 流式策略（`stream=true`）

需要明确：

1. 对外传输协议（SSE 或分片 JSON）。
2. streaming 过程中消息落库策略（分片缓存 + 最终落单条 assistant 消息）。
3. 工具事件在流中的可见性与顺序保证。

### 5.6 错误分类与回退

统一错误码建议：

1. `MODEL_TIMEOUT`
2. `MODEL_RATE_LIMIT`
3. `TOOL_TIMEOUT`
4. `PERMISSION_DENIED`
5. `RUNTIME_EXEC_FAILED`
6. `CHECKPOINT_STORE_FAILED`
7. `RESUME_CONTEXT_MISSING`

回退策略：

1. `runtime=eino` 且失败时，若 `runtime_fallback=true` 则自动降级 native。
2. 回退流程同样写 usage 与审计日志。

### 5.7 配置扩展（接入前必须落地）

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

  runtime: "native"         # native / eino
  runtime_fallback: true    # eino失败是否自动回退native

  eino:
    max_steps: 4
    tool_timeout_ms: 3000
    enable_stream: false
    checkpoint_ttl_seconds: 1800
```

### 5.8 版本锁定策略（必须补齐）

1. 锁定 `github.com/cloudwego/eino` 与 `eino-ext` 版本（禁止 floating）。
2. 引入依赖后执行 `go test ./...` + smoke tests。
3. 在文档记录升级流程与破坏性变更检查项。

## 6. 分阶段实施计划

### Phase 0：依赖与基线（0.5 天）

1. 引入 Eino 依赖并锁版本。
2. 跑通现有接口基线（chat、history、action）。

### Phase 1：运行时抽象（1 天）

1. 新增 runtime 接口与 `NativeRuntime`。
2. `AssistantService` 改为 runtime 驱动。

### Phase 2：EinoRuntime 接入（1.5~2 天）

1. 接入 ChatModel + Tool Adapter + Runner Query。
2. 对齐现有落库语义。

### Phase 3：恢复能力与灰度（1 天）

1. CheckPointStore 接入与 Resume 实测。
2. 开启 `runtime=eino` 小流量灰度，观测 24h。

### Phase 4：流式与增强（后续）

1. `stream=true` 真正可用。
2. 工具注册中心化，弱化硬编码规则。

## 7. 测试计划

### 7.1 单元测试

1. runtime 选择逻辑（native/eino/fallback）。
2. Eino 输出映射准确性。
3. Checkpoint 失效/缺失/恢复失败分支。
4. 工具适配层权限与超时分支。

### 7.2 集成测试

1. 活动检索（`activity_search`）
2. 运营统计（`activity_stats`）
3. 草案生成（`activity_draft_generate`）
4. 中断后 Resume 场景

验收点：

1. `reply` 非空且语义正确。
2. `tool_calls` 完整落库。
3. `ai_messages / ai_tool_calls / ai_usage_daily` 数据一致。

## 8. 发布与回滚

发布策略：

1. 默认 `runtime=native` 上线代码。
2. 小流量切 `runtime=eino`，观察成功率、P95、工具失败率、恢复成功率。
3. 指标达标后逐步放量。

回滚策略：

1. 配置回切 `runtime=native`（无须回滚代码）。
2. 异常扩大时关闭 `ai.enabled`。

## 9. DoD（完成标准）

1. `runtime=native` 与 `runtime=eino` 都能完成完整会话流程。
2. 三类工具场景在 Eino 下稳定可用，失败可追踪。
3. Checkpoint 恢复链路可用并有监控。
4. 数据落库语义与现有系统一致。
5. 配置可灰度、可回滚、文档与代码一致。

## 10. 参考资料（官方）

1. Eino Agent with Tools Quick Start  
   https://cloudwego.cn/docs/eino/quick_start/agent_llm_with_tools/
2. Eino ADK ChatModel Agent  
   https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_implementation/chat_model/
3. Eino ADK Extension / Runner / Resume  
   https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_extension/
4. Eino DeepSeek ChatModel  
   https://www.cloudwego.io/docs/eino/ecosystem_integration/chat_model/chat_model_deepseek/
