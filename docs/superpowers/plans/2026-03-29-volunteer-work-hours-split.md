# Volunteer Work Hours Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a volunteer-domain work-hour list endpoint without changing the existing organization work-hour list endpoint.

**Architecture:** Keep the current organization work-hour query path intact and add a separate volunteer-facing API contract, handler, router entry, and service method. The volunteer path will enforce current-volunteer scope and return a trimmed DTO suited for volunteer clients.

**Tech Stack:** Go, Hertz, protobuf/OpenAPI generation, GORM

---

### Task 1: Add volunteer-facing API contract

**Files:**
- Modify: `internal/api/volunteer.proto`
- Modify: generated `internal/api/volunteer.pb.go`
- Modify: generated `docs/openapi.yaml`
- Test: `internal/service/volunteer_work_hours_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestVolunteerWorkHourListRequiresAccountContext(t *testing.T) {
    svc := &VolunteerService{Service: Service{ctx: context.Background(), c: app.NewContext(0), repo: &repository.Repository{}}}
    _, err := svc.VolunteerWorkHourList(&api.VolunteerWorkHourListRequest{Page: 1, PageSize: 20})
    if err == nil {
        t.Fatal("expected missing account context to return an error")
    }
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestVolunteerWorkHourListRequiresAccountContext -count=1`
Expected: FAIL because `VolunteerWorkHourList` and/or volunteer work-hour API types do not exist yet

- [ ] **Step 3: Write minimal implementation**

Add a volunteer-domain RPC and messages in `internal/api/volunteer.proto`, generate protocol artifacts, and add the new service method signature needed by the test.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run TestVolunteerWorkHourListRequiresAccountContext -count=1`
Expected: PASS

### Task 2: Wire volunteer handler and route

**Files:**
- Modify: `internal/handler/volunteer.go`
- Modify: `internal/router/volunteer.go`
- Test: `internal/service/volunteer_work_hours_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestVolunteerWorkHourListRejectsNonVolunteerAccount(t *testing.T) {
    // create request context with organization account id in middleware context
    // expect VolunteerWorkHourList to return an identity error
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestVolunteerWorkHourListRejectsNonVolunteerAccount -count=1`
Expected: FAIL because volunteer-only identity validation is not implemented yet

- [ ] **Step 3: Write minimal implementation**

Add volunteer handler/router wiring and implement volunteer identity validation in the new service method.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run TestVolunteerWorkHourListRejectsNonVolunteerAccount -count=1`
Expected: PASS

### Task 3: Return volunteer-scoped work-hour data

**Files:**
- Modify: `internal/service/volunteer.go`
- Modify: `internal/repository/work_hour.go` (only if query support is needed)
- Modify: `docs/backend-api-alignment.backup.md`
- Test: `internal/service/volunteer_work_hours_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestVolunteerWorkHourListTrimsSensitiveFieldsAtContractLevel(t *testing.T) {
    item := &api.VolunteerWorkHourItem{}
    _ = item
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run TestVolunteerWorkHourListTrimsSensitiveFieldsAtContractLevel -count=1`
Expected: FAIL until the volunteer DTO exists

- [ ] **Step 3: Write minimal implementation**

Implement `VolunteerWorkHourList`, scope results to the current volunteer, map query results into the volunteer DTO, and update docs to point volunteer clients at `/api/volunteers/work-hours`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run 'TestVolunteerWorkHourList.*' -count=1`
Expected: PASS
