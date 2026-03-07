# P0 Product Iteration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Deliver the 4-week P0 roadmap by hardening authorization, upgrading audit/notification/discovery, and adding operations visibility.

**Architecture:** Keep current layered structure (`router -> handler -> service -> repository`) and evolve each capability behind backward-compatible APIs where possible. Introduce centralized authorization helpers and event-driven notification fan-out without rewriting existing business modules. For AI streaming, add a dedicated SSE endpoint (`/api/assistant/chat/stream`) and keep existing synchronous `/api/assistant/chat` unchanged. Enforce TDD for each behavior change and ship in small commits.

**Tech Stack:** Go, Hertz, GORM, MySQL, Redis, Protobuf/OpenAPI, Go test

---

Execution rules for this plan:
- Use `@test-driven-development` for every behavior change.
- Run `@verification-before-completion` at the end of every task.
- Keep commits small and focused (one task = one commit).

### Task 0: Add RBAC Schema Migration Artifacts

**Files:**
- Create: `sql/ddl/ddl_v1.3.1.sql`
- Modify: `deploy/ddl.sql`

**Step 1: Define DDL scope and naming**

- DDL 版本号：`v1.3.1`
- 新增表：`rbac_roles`, `rbac_permissions`, `rbac_role_permissions`, `rbac_account_roles`
- 与现有目录规范保持一致（`sql/ddl` 增量 + `deploy/ddl.sql` 全量）

**Step 2: Generate incremental DDL file**

- 生成 `sql/ddl/ddl_v1.3.1.sql`
- 仅编写建表语句和索引语句，不执行脚本

**Step 3: Sync full schema file**

- 将同样的 4 张表同步到 `deploy/ddl.sql`
- 保持表注释、字段注释、索引命名风格与现有文件一致

**Step 4: Record Change**

- 记录本任务改动文件，进入下一任务。
- 本任务不执行任何 SQL 或其他脚本。


### Task 1: Add Centralized Authorization Policy

**Files:**
- Create: `internal/authz/policy.go`
- Test: `internal/authz/policy_test.go`
- Modify: `internal/model/consts.go`

**Step 1: Write the failing test**

```go
func TestCanManageOrganization(t *testing.T) {
    t.Run("owner can manage", func(t *testing.T) { /* expect true */ })
    t.Run("non owner denied", func(t *testing.T) { /* expect false */ })
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/authz -run TestCanManageOrganization -v`
Expected: FAIL with undefined policy symbols.

**Step 3: Write minimal implementation**

```go
func CanManageOrganization(actorAccountID, ownerAccountID int64) bool {
    return actorAccountID > 0 && actorAccountID == ownerAccountID
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/authz -run TestCanManageOrganization -v`
Expected: PASS.

**Step 5: Record Change**

- 记录本任务改动文件与验证结果，进入下一任务。


### Task 2: Remove Client-Controlled `accountId` in Member Status Update

**Files:**
- Modify: `internal/api/membership.proto`
- Modify: `internal/service/membership.go`
- Modify: `internal/handler/memberships.go`
- Modify: `internal/api/membership.pb.go` (generated)
- Modify: `docs/openapi.yaml` (generated)
- Test: `internal/service/membership_update_status_test.go`

**Step 1: Write the failing test**

```go
func TestUpdateMemberStatus_UsesTokenUserNotBodyAccountID(t *testing.T) {
    // given body accountId mismatch
    // expect service uses middleware user id and still enforces owner check
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestUpdateMemberStatus_UsesTokenUserNotBodyAccountID -v`
Expected: FAIL because service still depends on request `accountId`.

**Step 3: Write minimal implementation**

```go
operatorID, err := middleware.GetUserIDInt(s.c)
if err != nil { return nil, err }
// delete req.AccountId dependency
```

**Step 4: Regenerate API artifacts and rerun tests**

Run: `make api-single-mac file=membership.proto`
Run: `go test ./internal/service -run TestUpdateMemberStatus_UsesTokenUserNotBodyAccountID -v`
Expected: PASS.

**Step 5: Record Change**

- 记录本任务改动文件与验证结果，进入下一任务。


### Task 3: Enforce Organization Ownership on All Organization Writes

**Files:**
- Modify: `internal/service/organization.go`
- Test: `internal/service/organization_permission_test.go`

**Step 1: Write the failing test**

```go
func TestOrganizationWriteOps_DenyNonOwner(t *testing.T) {
    // update/delete/disable/enable should fail for non-owner
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestOrganizationWriteOps_DenyNonOwner -v`
Expected: FAIL because current methods do not always check owner.

**Step 3: Write minimal implementation**

```go
userID, _ := middleware.GetUserIDInt(s.c)
if !authz.CanManageOrganization(userID, organization.AccountID) {
    return nil, errors.New("无权操作该组织")
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run TestOrganizationWriteOps_DenyNonOwner -v`
Expected: PASS.

**Step 5: Record Change**

- 记录本任务改动文件与验证结果，进入下一任务。


### Task 4: Audit Center V2 (Batch Decisions + Time Filters + SLA Flag)

**Files:**
- Modify: `internal/api/audit.proto`
- Modify: `internal/router/audit.go`
- Modify: `internal/handler/audit.go`
- Modify: `internal/service/audit.go`
- Modify: `internal/repository/audit.go`
- Modify: `internal/api/audit.pb.go` (generated)
- Modify: `docs/openapi.yaml` (generated)
- Test: `internal/service/audit_batch_test.go`

**Step 1: Write the failing test**

```go
func TestAuditBatchDecision_PartialSuccess(t *testing.T) {
    // mixed pending/non-pending ids returns success_count + failed_ids
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestAuditBatchDecision_PartialSuccess -v`
Expected: FAIL because batch API does not exist.

**Step 3: Write minimal implementation**

```go
rpc AuditBatchDecision(AuditBatchDecisionRequest) returns (AuditBatchDecisionResponse)
```

Implement service flow:
- validate ids and action
- process each id in transaction-safe path
- aggregate `successCount` and `failedIds`
- compute `isOverdue` in list response by SLA hours

**Step 4: Regenerate artifacts and run tests**

Run: `make api-single-mac file=audit.proto`
Run: `go test ./internal/service -run TestAuditBatchDecision_PartialSuccess -v`
Expected: PASS.

**Step 5: Record Change**

- 记录本任务改动文件与验证结果，进入下一任务。


### Task 5: Expand Notification Event Matrix

**Files:**
- Modify: `internal/model/consts.go`
- Modify: `internal/service/activities.go`
- Modify: `internal/service/audit.go`
- Modify: `internal/service/notification.go`
- Test: `internal/service/notification_event_test.go`

**Step 1: Write the failing test**

```go
func TestRenderNotificationMessage_SignupRejected(t *testing.T) {
    // expect title/content for signup rejected event
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestRenderNotificationMessage_SignupRejected -v`
Expected: FAIL because event not defined.

**Step 3: Write minimal implementation**

```go
const NotificationEventSignupRejected = "signup_rejected"
```

Add publish points:
- signup approval/rejection in audit flow
- activity cancellation event in activity service

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run TestRenderNotificationMessage_SignupRejected -v`
Expected: PASS.

**Step 5: Record Change**

- 记录本任务改动文件与验证结果，进入下一任务。


### Task 6: Add Email Delivery Channel (Config-Gated)

**Files:**
- Create: `pkg/notify/email_sender.go`
- Create: `internal/service/notification_channel_email.go`
- Modify: `internal/service/notification_dispatcher.go`
- Modify: `config/config.go`
- Test: `internal/service/notification_channel_email_test.go`

**Step 1: Write the failing test**

```go
func TestEmailChannel_DisabledConfigSkipsSend(t *testing.T) {
    // expect no send and no error when email disabled
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestEmailChannel_DisabledConfigSkipsSend -v`
Expected: FAIL because channel does not exist.

**Step 3: Write minimal implementation**

```go
func SendEmail(cfg *config.EmailConfig, to, subject, body string) error {
    // build smtp client from cfg and send directly
    return nil
}
```

Call email channel after inbox write in dispatcher worker when config enabled.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run TestEmailChannel_DisabledConfigSkipsSend -v`
Expected: PASS.

**Step 5: Record Change**

- 记录本任务改动文件与验证结果，进入下一任务。


### Task 7: Upgrade Activity Discovery Filters and Sorting

**Files:**
- Modify: `internal/api/activities.proto`
- Modify: `internal/repository/activities.go`
- Modify: `internal/service/activities.go`
- Modify: `internal/api/activities.pb.go` (generated)
- Modify: `docs/openapi.yaml` (generated)
- Test: `internal/service/activity_list_filter_test.go`

**Step 1: Write the failing test**

```go
func TestActivityList_FilterByKeywordAndTimeRange(t *testing.T) {
    // expect only matching activities returned
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestActivityList_FilterByKeywordAndTimeRange -v`
Expected: FAIL because request fields/query are missing.

**Step 3: Write minimal implementation**

Add fields in `ActivityListRequest`:
- `keyword`
- `startFrom`
- `startTo`
- `sortBy` (`start_time`, `created_at`)
- `sortOrder` (`asc`, `desc`)

Wire them through repository query builder.

**Step 4: Regenerate artifacts and run tests**

Run: `make api-single-mac file=activities.proto`
Run: `go test ./internal/service -run TestActivityList_FilterByKeywordAndTimeRange -v`
Expected: PASS.

**Step 5: Record Change**

- 记录本任务改动文件与验证结果，进入下一任务。


### Task 8: Add Operations Dashboard API (Funnel + Conversion)

**Files:**
- Create: `internal/api/analytics.proto`
- Create: `internal/router/analytics.go`
- Create: `internal/handler/analytics.go`
- Create: `internal/service/analytics.go`
- Create: `internal/repository/analytics.go`
- Modify: `internal/router/router.go`
- Modify: `docs/openapi.yaml` (generated)
- Test: `internal/service/analytics_funnel_test.go`

**Step 1: Write the failing test**

```go
func TestOrgFunnelSummary_ComputesConversionRates(t *testing.T) {
    // expect registration->membership->signup->attendance->workhour rates
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestOrgFunnelSummary_ComputesConversionRates -v`
Expected: FAIL because analytics service/API does not exist.

**Step 3: Write minimal implementation**

Define endpoint:
- `GET /api/analytics/org/funnel?orgId=&start=&end=`

Response fields:
- stage counts
- conversion percentages
- period metadata

**Step 4: Generate artifacts and run tests**

Run: `make api-single-mac file=analytics.proto`
Run: `go test ./internal/service -run TestOrgFunnelSummary_ComputesConversionRates -v`
Expected: PASS.

**Step 5: Record Change**

- 记录本任务改动文件与验证结果，进入下一任务。


### Task 9: Add Weekly/Monthly Export Templates

**Files:**
- Modify: `internal/api/export.proto`
- Modify: `internal/service/export.go`
- Modify: `internal/repository/export.go`
- Modify: `pkg/util/export_writer.go`
- Modify: `internal/api/export.pb.go` (generated)
- Modify: `docs/openapi.yaml` (generated)
- Test: `internal/service/export_template_test.go`

**Step 1: Write the failing test**

```go
func TestExportOpsReport_MonthlyTemplateColumns(t *testing.T) {
    // expect fixed columns for monthly ops template
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestExportOpsReport_MonthlyTemplateColumns -v`
Expected: FAIL because report template API does not exist.

**Step 3: Write minimal implementation**

Add endpoint:
- `POST /api/admin/export/ops-report`

Request:
- `periodType` (`weekly`/`monthly`)
- `orgId`
- `start`/`end`

Generate XLSX with frozen columns and standard metric headers.

**Step 4: Generate artifacts and run tests**

Run: `make api-single-mac file=export.proto`
Run: `go test ./internal/service -run TestExportOpsReport_MonthlyTemplateColumns -v`
Expected: PASS.

**Step 5: Record Change**

- 记录本任务改动文件与验证结果，进入下一任务。


### Task 10: Introduce AI Streaming Endpoint (SSE) with Backward Compatibility

**Files:**
- Modify: `internal/api/assistant.proto`
- Modify: `internal/router/assistant.go`
- Modify: `internal/handler/assistant.go`
- Modify: `internal/service/assistant_service.go`
- Create: `internal/service/assistant_stream.go`
- Modify: `cmd/cli/server.go`
- Modify: `internal/api/assistant.pb.go` (generated)
- Modify: `docs/openapi.yaml` (generated)
- Test: `internal/service/assistant_stream_test.go`
- Test: `internal/handler/assistant_stream_handler_test.go`

**Step 1: Write the failing test**

```go
func TestAssistantChatStream_EmitsDeltaAndDoneEvents(t *testing.T) {
    // expect SSE frames: start -> delta* -> usage -> done
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestAssistantChatStream_EmitsDeltaAndDoneEvents -v`
Expected: FAIL because stream endpoint is missing.

**Step 3: Write minimal implementation**

- Add new RPC/endpoint for stream chat (`/api/assistant/chat/stream`) using SSE.
- Keep `/api/assistant/chat` unchanged for old clients.
- Stream event schema:
  - `start`
  - `delta`
  - `tool`
  - `usage`
  - `done`
  - `error`
- Add stream safeguards:
  - heartbeat every 15s
  - client disconnect cancellation
  - stream-specific timeout and limits
- Do not use `response.Success` wrapper for streaming responses.

**Step 4: Generate artifacts and run tests**

Run: `make api-single-mac file=assistant.proto`
Run: `go test ./internal/service -run TestAssistantChatStream_EmitsDeltaAndDoneEvents -v`
Run: `go test ./internal/handler -run TestAssistantChatStream -v`
Expected: PASS.

**Step 5: Record Change**

- 记录本任务改动文件与验证结果，进入下一任务。


### Task 11: Full Verification and Release Checklist

**Files:**
- Modify: `README.md`
- Modify: `docs/openapi.yaml`
- Create: `docs/release/p0-iteration-release-checklist.md`

**Step 1: Write the failing check script/test list**

```bash
go test ./...
```

Record all current failures.

**Step 2: Run verification suite**

Run:
- `go test ./...`
- `make build-mac`
- `make api`

Expected: all pass and OpenAPI generated.

**Step 3: Finalize docs**

Update README sections:
- new analytics endpoint
- new export report API
- notification channels and config
- RBAC schema migration (`sql/ddl/ddl_v1.3.1.sql`)
- AI stream endpoint (`/api/assistant/chat/stream`) and client usage notes

**Step 4: Re-run smoke verification**

Run:
- `go test ./...`
- `go run cmd/main.go -c server` and manual smoke on new endpoints

Expected: clean tests + server starts without runtime errors.

**Step 5: Record Change**

- 记录本任务改动文件与验证结果，进入下一任务。
