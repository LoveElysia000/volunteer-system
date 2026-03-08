# P0 Product Iteration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Deliver the 4-week P0 roadmap by hardening authorization, upgrading audit/notification/discovery, and adding operations visibility.

**Architecture:** Keep current layered structure (`router -> handler -> service -> repository`) and evolve each capability behind backward-compatible APIs where possible. Authorization is RBAC-first: permissions are defined by `resource.action`, assigned to roles, and granted to accounts by scope (`global/org`) with `super_admin` as global governance role. For AI streaming, add a dedicated SSE endpoint (`/api/assistant/chat/stream`) and keep existing synchronous `/api/assistant/chat` unchanged. Enforce TDD for each behavior change and ship in small commits.

**Tech Stack:** Go, Hertz, GORM, MySQL, Redis, Protobuf/OpenAPI, Go test

---

Execution rules for this plan:
- Use `@test-driven-development` for every behavior change.
- Run `@verification-before-completion` at the end of every task.
- Keep commits small and focused (one task = one commit).

### RBAC Governance Model

- `rbac_permissions`: Defines permission points where `resource + action` is unique (for example, `organization.manage`).
- `rbac_role_permissions`: Defines the permission set owned by each role.
- `rbac_account_roles`: Defines role bindings for an account under a scope (`scope_type=global/org`) with `status` and `expires_at` support.
- `rbac_roles`: Role definitions (this iteration includes `super_admin`, `org_owner`, `org_manager`, and `volunteer`).

Permission check flow:
1. Handler gets the operator `account_id` from token/middleware and does not trust client-passed account IDs.
2. Service calls shared `requireOrgPermission/requireGlobalPermission` helper methods.
3. Repository checks RBAC relations by `account_id + scope + resource + action`.
4. If `super_admin` is matched with `global` scope, allow globally; otherwise enforce exact org-scope checks.
5. If no permission is matched, return a unified authorization error code.

`super_admin` responsibilities:
- Manage role-to-permission mappings (who can operate which features).
- Handle cross-organization governance operations and emergency actions.
- Own platform-level capabilities for audit, export, analytics view, and organization management.

### Registration RBAC Resilience (No New Table)

Goal: prevent RBAC dependency failures from blocking account registration while still guaranteeing eventual default-role availability.

Implementation policy:
- Keep registration transaction focused on core business entities only (`sys_accounts`, `volunteers` / `organizations`).
- Move default RBAC binding out of the registration transaction and execute it as best-effort with short retries.
- If RBAC binding still fails, registration remains successful and the failure is logged for follow-up.
- On login, run default-role self-heal: if the expected default role binding is missing, auto-upsert it.
- Provide a reconcile entry to scan active accounts and backfill missing default bindings in batches.

Scope and storage decision:
- Reuse existing RBAC tables (`rbac_roles`, `rbac_account_roles`) and existing business tables.
- Do not add new DDL or new tables for this resilience upgrade.

### Task 0: Add RBAC Schema Migration Artifacts

**Files:**
- Create: `sql/ddl/ddl_v1.3.1.sql`
- Modify: `deploy/ddl.sql`

**Step 1: Define DDL scope and naming**

- DDL version: `v1.3.1`
- New tables: `rbac_roles`, `rbac_permissions`, `rbac_role_permissions`, `rbac_account_roles`
- Keep alignment with existing directory conventions (`sql/ddl` incremental + `deploy/ddl.sql` full schema).

**Step 2: Generate incremental DDL file**

- Generate `sql/ddl/ddl_v1.3.1.sql`
- Only write table-creation and index statements; do not execute scripts.

**Step 3: Sync full schema file**

- Sync the same 4 tables into `deploy/ddl.sql`
- Keep table comments, column comments, and index naming style consistent with existing files.

**Step 4: Record Change**

- Record changed files for this task, then proceed to the next task.
- This task does not execute any SQL or any other scripts.


### Task 1: Add RBAC Role/Permission Seeds and Service-Level Authorizer

**Files:**
- Create: `sql/dml/dml_v1.3.1.sql`
- Create: `internal/repository/rbac.go`
- Create: `internal/service/rbac_helpers.go`
- Test: `internal/service/authz_helpers_test.go`
- Modify: `internal/model/consts.go`

**Step 1: Write the failing test**

```go
func TestHasPermissionByScope(t *testing.T) {
    t.Run("super admin global allow", func(t *testing.T) { /* expect allow */ })
    t.Run("org role allow in same org", func(t *testing.T) { /* expect allow */ })
    t.Run("org role deny in other org", func(t *testing.T) { /* expect deny */ })
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestHasPermissionByScope -v`
Expected: FAIL with undefined RBAC helper symbols.

**Step 3: Write minimal implementation**

- Add permission constants in `internal/model/consts.go` (for example: `organization.manage`, `membership.manage`, `audit.review`, `export.manage`, `analytics.org.read`).
- Generate seed data in `sql/dml/dml_v1.3.1.sql` for `super_admin`, `org_owner`, `org_manager`, and `volunteer`, plus role-permission mappings.
- Add repository queries in `internal/repository/rbac.go` to evaluate permissions by `account_id + scope + resource + action`.
- Add service-level authorization helpers in `internal/service/rbac_helpers.go` for reuse across services.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run TestHasPermissionByScope -v`
Expected: PASS.

**Step 5: Record Change**

- Record changed files and verification results for this task, then proceed to the next task.


### Task 2: Remove Client-Controlled `accountId` and Enforce `membership.manage`

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
    // expect service uses middleware user id and checks membership.manage permission
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestUpdateMemberStatus_UsesTokenUserNotBodyAccountID -v`
Expected: FAIL because service still depends on request `accountId`.

**Step 3: Write minimal implementation**

```go
operatorID, err := middleware.GetUserIDInt(s.c)
if err != nil { return nil, err }
// delete req.AccountId dependency and enforce RBAC permission
if err := s.requireOrgPermission(operatorID, member.OrgID, "membership", "manage"); err != nil {
    return nil, err
}
```

**Step 4: Regenerate API artifacts and rerun tests**

Run: `make api-single-mac file=membership.proto`
Run: `go test ./internal/service -run TestUpdateMemberStatus_UsesTokenUserNotBodyAccountID -v`
Expected: PASS.

**Step 5: Record Change**

- Record changed files and verification results for this task, then proceed to the next task.


### Task 3: Apply RBAC on Organization Write APIs (`organization.manage`)

**Files:**
- Modify: `internal/service/organization.go`
- Test: `internal/service/organization_permission_test.go`

**Step 1: Write the failing test**

```go
func TestOrganizationWriteOps_ByRBAC(t *testing.T) {
    // super_admin allowed globally
    // org_owner/org_manager allowed in own org
    // volunteer denied
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestOrganizationWriteOps_ByRBAC -v`
Expected: FAIL because current methods are still owner/identity based.

**Step 3: Write minimal implementation**

```go
userID, _ := middleware.GetUserIDInt(s.c)
if err := s.requireOrgPermission(userID, organization.ID, "organization", "manage"); err != nil {
    return nil, err
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run TestOrganizationWriteOps_ByRBAC -v`
Expected: PASS.

**Step 5: Record Change**

- Record changed files and verification results for this task, then proceed to the next task.


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
- enforce `audit.review` permission by org scope (super_admin global bypass)
- process each id in transaction-safe path
- aggregate `successCount` and `failedIds`
- compute `isOverdue` in list response by SLA hours

**Step 4: Regenerate artifacts and run tests**

Run: `make api-single-mac file=audit.proto`
Run: `go test ./internal/service -run TestAuditBatchDecision_PartialSuccess -v`
Expected: PASS.

**Step 5: Record Change**

- Record changed files and verification results for this task, then proceed to the next task.


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

- Record changed files and verification results for this task, then proceed to the next task.


### Task 6: Add Email Delivery Channel (Config-Gated)

**Files:**
- Create: `pkg/notify/email_sender.go`
- Modify: `internal/service/notification.go` (includes email channel dispatch method)
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

- Record changed files and verification results for this task, then proceed to the next task.


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

- Record changed files and verification results for this task, then proceed to the next task.


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
- enforce `analytics.org.read` permission by scope

**Step 4: Generate artifacts and run tests**

Run: `make api-single-mac file=analytics.proto`
Run: `go test ./internal/service -run TestOrgFunnelSummary_ComputesConversionRates -v`
Expected: PASS.

**Step 5: Record Change**

- Record changed files and verification results for this task, then proceed to the next task.


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
- enforce `export.manage` permission by scope (super_admin global or org role)

**Step 4: Generate artifacts and run tests**

Run: `make api-single-mac file=export.proto`
Run: `go test ./internal/service -run TestExportOpsReport_MonthlyTemplateColumns -v`
Expected: PASS.

**Step 5: Record Change**

- Record changed files and verification results for this task, then proceed to the next task.


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

- Record changed files and verification results for this task, then proceed to the next task.


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
- RBAC seed data (`sql/dml/dml_v1.3.1.sql`) and role matrix (`super_admin/org_owner/org_manager/volunteer`)
- AI stream endpoint (`/api/assistant/chat/stream`) and client usage notes

**Step 4: Re-run smoke verification**

Run:
- `go test ./...`
- `go run cmd/main.go -c server` and manual smoke on new endpoints

Expected: clean tests + server starts without runtime errors.

**Step 5: Record Change**

- Record changed files and verification results for this task, then proceed to the next task.
