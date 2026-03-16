# P0 Operations Closure Design

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:writing-plans to create the implementation plan for this design.

**Goal:** 在不做大规模重构的前提下，完成当前保留的 P0 运营闭环能力，覆盖审核中心升级、导入中心、运营看板 V1 和端到端回归基线。

**Architecture:** 继续沿用现有 `router -> handler -> service -> repository` 分层，所有能力按现有领域增量扩展。优先复用 `audit`、`activities`、`analytics`、`notification`、`export` 模块，避免引入新的平台级聚合层和额外持久化表。

**Tech Stack:** Go, Hertz, GORM, MySQL, Protobuf/OpenAPI, Go test

---

## 1. 目标范围

本次仅实现路线图中的 P0：

1. 审核中心升级
2. 导入中心
3. 运营看板 V1
4. 端到端回归与稳定性基线

P1、P2 不在本轮实施范围内。

## 2. 设计决策

### 2.1 审核中心升级

保留现有统一审核入口和审核副作用实现，增强查询和批量处理能力：

1. `PendingAuditList` 支持多目标类型筛选，而不是单一 `targetType`。
2. 支持关键词匹配统一待审项标题与副标题。
3. 支持创建时间区间和 SLA 超时标记。
4. 保留组织作用域与 RBAC 权限校验，所有结果在 service 层做最终权限过滤。
5. 批量审核保持“部分成功”语义，返回成功数量和失败记录 ID。

### 2.2 导入中心

导入中心采用轻量同步方案，不新增任务表：

1. 支持 `xlsx` 与 `csv` 两种上传格式。
2. 服务端把文件解析为统一行模型，再分发到“志愿者导入”和“活动导入”两个处理器。
3. 导入时逐行校验，合法行写入已有业务表，非法行收集为错误结构体。
4. 响应体返回 `successCount`、`failedCount`、失败列表摘要，以及一份错误回执 `csv` 内容；前端在失败时自动下载该文件。
5. 第一版不做异步任务化、不做导入历史持久化、不做 Redis 队列。

### 2.3 运营看板 V1

运营看板先做“可决策的最小指标集”：

1. 查询维度仅支持 `orgId + start + end`。
2. 指标范围包括报名数、报名通过数、出勤数、出勤率、工时发放量。
3. 指标统一由后端聚合计算，前端不做业务口径拼装。
4. 仅具备分析权限的角色允许访问。

### 2.4 回归与稳定性基线

第一版回归基线不引入单独的 e2e 框架，直接基于 Go 测试补关键链路：

1. 覆盖成功路径：注册 -> 报名 -> 审核 -> 签到 -> 签退 -> 工时 -> 通知。
2. 覆盖一条权限拒绝路径，验证越权请求会被稳定拒绝。
3. CI 继续执行 `go test ./...`，把新增主链路测试纳入发布前固定检查。

## 3. 接口与模块落点

### 3.1 审核中心

涉及模块：

1. `internal/api/audit.proto`
2. `internal/service/audit.go`
3. `internal/repository/audit.go`
4. `internal/handler/audit.go`
5. `internal/router/audit.go`

变更方向：

1. 在待审查询请求中增加多目标类型字段。
2. 在 service 层完成关键词过滤和权限过滤。
3. 保持现有详情、单条审核和批量审核接口不拆分。

### 3.2 导入中心

涉及模块：

1. 新增 `internal/api/attachment.proto`
2. 新增 `internal/service/attachment.go`
3. 在现有 volunteer/activity repository 中补导入用方法
4. 新增 `internal/handler/attachment.go`
5. 新增 `internal/router/attachment.go`
6. 新增 `pkg/util` 下的文件解析与错误回执工具

变更方向：

1. 上传接口接收文件并识别格式。
2. 解析层统一输出结构化行数据。
3. 业务层执行校验、写入、去重和错误收集。
4. 错误回执统一导出成 `csv`。

### 3.3 运营看板

涉及模块：

1. `internal/api/analytics.proto`
2. `internal/service/analytics.go`
3. `internal/repository/analytics.go`
4. `internal/handler/analytics.go`

变更方向：

1. 在现有组织漏斗基础上新增看板汇总接口。
2. 复用组织权限校验模式。
3. 统一统计口径到 repository 层。

### 3.4 回归基线

涉及模块：

1. `internal/service/*_test.go`
2. 必要时新增共享测试构造工具
3. `.github/workflows/ci.yml`

变更方向：

1. 为新增和关键现有能力补服务级测试。
2. 保持 CI 命令简单，继续使用 `go test ./...`。

## 4. 非目标

本轮明确不做以下内容：

1. 不新增导入任务持久化表。
2. 不做导入历史查询和失败回放。
3. 不做复杂调度中心或分布式任务系统。
4. 不做完整 BI 看板或趋势图明细。
5. 不实现 P1/P2 的证书、通知增强、推荐或 AI 扩展。

## 5. 风险与应对

1. Proto 变更会联动生成代码，必须保证 `.pb.go` 与接口实现同步更新。
2. 导入同步处理可能受文件大小影响，需要设置大小和行数上限。
3. 历史数据口径可能不完整，运营看板需优先保证定义一致，而非追求绝对全量。
4. 当前测试基础薄弱，需要先搭出最小可复用测试模式，再逐步补充覆盖。
