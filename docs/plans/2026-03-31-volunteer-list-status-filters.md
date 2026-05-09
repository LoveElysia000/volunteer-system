# Volunteer List Status Filters Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `auditStatuses` and `statuses` array filters to the volunteer list request so admin users can filter volunteers by audit status and volunteer activity status.

**Architecture:** Extend the volunteer list protobuf request, validate the two repeated enum-code fields in the service layer, and translate them into `IN` filters reused by the existing repository query-map pattern. Keep the response unchanged and add focused unit coverage around the new filter-building helper.

**Tech Stack:** Go, Protocol Buffers, Hertz, GORM, OpenAPI generation

---

### Task 1: Add a failing test for filter validation and query construction

**Files:**
- Create: `internal/service/volunteer_test.go`
- Modify: `internal/service/volunteer.go`

**Step 1: Write the failing test**

Add tests that expect:
- valid `auditStatuses` and `statuses` to become `v.audit_status IN ?` and `v.status IN ?`
- invalid audit status to return an error
- invalid volunteer status to return an error

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run "TestBuildVolunteerListFilters" -count=1`

Expected: FAIL because the helper does not exist yet.

**Step 3: Write minimal implementation**

Add a helper in `internal/service/volunteer.go` that validates enum values and builds the repository query map.

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run "TestBuildVolunteerListFilters" -count=1`

Expected: PASS

### Task 2: Wire the new request fields through the volunteer list API

**Files:**
- Modify: `internal/api/volunteer.proto`
- Modify: `internal/api/volunteer.pb.go`
- Modify: `docs/openapi.yaml`
- Modify: `internal/service/volunteer.go`

**Step 1: Extend the protobuf request**

Add:
- `repeated int32 auditStatuses`
- `repeated int32 statuses`

to `VolunteerListRequest`, keeping existing fields intact.

**Step 2: Regenerate generated artifacts**

Run the single-proto generation flow for `internal/api/volunteer.proto` so `internal/api/volunteer.pb.go` and `docs/openapi.yaml` reflect the new request schema.

**Step 3: Hook request fields into service logic**

Replace the ad-hoc keyword-only query-map construction with the validated helper so keyword and both status filters can combine.

**Step 4: Run targeted tests**

Run: `go test ./internal/service -run "TestBuildVolunteerListFilters" -count=1`

Expected: PASS

### Task 3: Verify end-to-end compile safety

**Files:**
- Modify: none expected

**Step 1: Run package tests**

Run: `go test ./internal/service -count=1`

Expected: PASS

**Step 2: Run broader build or test if needed**

Run: `go test ./...`

Expected: PASS, or capture unrelated failures explicitly if present.
