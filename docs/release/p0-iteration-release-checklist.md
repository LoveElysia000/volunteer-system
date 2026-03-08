# P0 Iteration Release Checklist

## 1. Schema and Seed

- [ ] Apply incremental DDL: `sql/ddl/ddl_v1.3.1.sql`
- [ ] Apply RBAC seed DML: `sql/dml/dml_v1.3.1.sql`
- [ ] Confirm full schema includes RBAC tables in `deploy/ddl.sql`

## 2. API Artifacts

- [ ] Run `make api`
- [ ] Confirm updated API files:
  - `internal/api/activities.pb.go`
  - `internal/api/analytics.pb.go`
  - `internal/api/assistant.pb.go`
  - `internal/api/audit.pb.go`
  - `internal/api/export.pb.go`
  - `internal/api/membership.pb.go`
- [ ] Confirm `docs/openapi.yaml` is updated

## 3. Functional Verification

- [ ] RBAC authorization checks
  - `organization.manage`
  - `membership.manage`
  - `audit.review`
  - `analytics.org.read`
  - `export.manage`
- [ ] Audit batch decision returns partial success payload
- [ ] Notification event matrix covers:
  - `signup_rejected`
  - `activity_canceled`
- [ ] Email channel can be disabled via config without side effects
- [ ] Activity list filters/sorting work (keyword/time/sort)
- [ ] Analytics funnel endpoint returns stage counts + conversion rates
- [ ] Ops report export returns fixed template columns
- [ ] SSE stream endpoint emits `start/delta/tool/usage/done/error`

## 4. Build and Tests

- [ ] Run `go test ./...`
- [ ] Run `make build-mac`
- [ ] Start server: `go run cmd/main.go -c server`

## 5. Smoke Test Endpoints

- [ ] `/api/audits/batch-decision`
- [ ] `/api/analytics/org/funnel`
- [ ] `/api/admin/export/ops-report`
- [ ] `/api/assistant/chat/stream`
