# P0 Operations Closure Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 逐项落地 P0 运营闭环能力，并在每个任务内坚持 TDD 与小步验证。

**Architecture:** 基于现有分层架构做增量扩展。先完成审核中心升级，随后补同步导入、运营看板，最后用测试把主链路串起来，确保可持续迭代。

**Tech Stack:** Go, Hertz, GORM, MySQL, Protobuf/OpenAPI, Go test

---

### Task 1: 审核中心升级

**Files:**
- Modify: `internal/api/audit.proto`
- Modify: `internal/api/audit.pb.go`
- Modify: `internal/service/audit.go`
- Modify: `internal/repository/audit.go`
- Test: `internal/service/audit_test.go`

**Step 1: Write the failing test**

为以下场景补测试：

1. 待审列表支持多目标类型。
2. 关键词只返回标题或副标题匹配项。
3. 无效目标类型会返回错误。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run "TestPendingAuditList" -count=1`

Expected: FAIL，原因是新字段和新过滤行为尚未实现。

**Step 3: Write minimal implementation**

1. 扩展 `PendingAuditListRequest` 支持多目标类型。
2. repository 支持 `target_type IN ?` 查询。
3. service 在组装标题后做关键词过滤，并保持权限过滤优先正确。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run "TestPendingAuditList" -count=1`

Expected: PASS

**Step 5: Verify broader impact**

Run: `go test ./internal/service -run "TestAudit" -count=1`

Expected: PASS

### Task 2: 志愿者导入（同步）

**Files:**
- Create: `internal/api/import.proto`
- Create: `internal/service/import.go`
- Create: `internal/handler/import.go`
- Create: `internal/router/import.go`
- Create: `internal/service/import_test.go`
- Modify: `internal/repository/volunteer.go`
- Modify: `pkg/util/...`

**Step 1: Write the failing test**

补测试覆盖：

1. `csv` 志愿者导入成功写入。
2. 同一文件中的非法行会进入错误回执。
3. 合法行和非法行可部分成功。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run "TestImportVolunteers" -count=1`

Expected: FAIL，原因是导入服务和错误回执尚未实现。

**Step 3: Write minimal implementation**

1. 完成文件格式识别和 `csv/xlsx` 解析。
2. 把失败行转换为统一错误结构。
3. 返回统计结果和错误回执 `csv` 字节。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run "TestImportVolunteers" -count=1`

Expected: PASS

### Task 3: 活动导入（同步）

**Files:**
- Modify: `internal/api/import.proto`
- Modify: `internal/service/import.go`
- Modify: `internal/service/import_test.go`
- Modify: `internal/repository/activities.go`

**Step 1: Write the failing test**

补测试覆盖：

1. 活动 `xlsx` 导入成功。
2. 时间格式错误会进入错误回执。
3. 重复活动或非法组织作用域被拒绝。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run "TestImportActivities" -count=1`

Expected: FAIL

**Step 3: Write minimal implementation**

实现活动导入的行校验、组织权限校验和批量写入。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run "TestImportActivities" -count=1`

Expected: PASS

### Task 4: 运营看板 V1

**Files:**
- Modify: `internal/api/analytics.proto`
- Modify: `internal/api/analytics.pb.go`
- Modify: `internal/service/analytics.go`
- Modify: `internal/repository/analytics.go`
- Test: `internal/service/analytics_test.go`

**Step 1: Write the failing test**

补测试覆盖：

1. 返回报名数、通过数、出勤数、出勤率和工时发放量。
2. 非法时间区间返回错误。
3. 无权限用户被拒绝。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run "TestOpsDashboardSummary" -count=1`

Expected: FAIL

**Step 3: Write minimal implementation**

1. 新增看板接口定义。
2. repository 聚合核心指标。
3. service 做参数解析和权限校验。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run "TestOpsDashboardSummary" -count=1`

Expected: PASS

### Task 5: 回归与稳定性基线

**Files:**
- Create: `internal/service/regression_test.go`
- Modify: `.github/workflows/ci.yml`

**Step 1: Write the failing test**

补主链路回归和权限拒绝测试。

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run "TestCoreRegression|TestPermissionRegression" -count=1`

Expected: FAIL

**Step 3: Write minimal implementation**

复用前面任务的服务能力和测试夹具，串联出核心业务链路。

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run "TestCoreRegression|TestPermissionRegression" -count=1`

Expected: PASS

### Task 6: 整体验证

**Files:**
- Verify only

**Step 1: Run targeted test suites**

Run: `go test ./internal/service -count=1`

Expected: PASS

**Step 2: Run full verification**

Run: `go test ./...`

Expected: PASS
