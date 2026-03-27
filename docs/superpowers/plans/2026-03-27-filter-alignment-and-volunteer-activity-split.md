# Filter Alignment And Volunteer Activity Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add backend-side notification keyword filtering, audit queue-state filtering, and split volunteer "my activities" into a dedicated API path without breaking existing behavior.

**Architecture:** Extend the existing proto/service/repository request flow for notifications and audits so filtering happens before pagination totals are computed. For volunteer activities, keep the current shared list behavior compatible while introducing a dedicated `/api/activities/my` entry backed by the existing registered-only logic.

**Tech Stack:** Go, Hertz, protobuf-generated API contracts, GORM, Go test

---

### Task 1: Notification Keyword Filter

**Files:**
- Modify: `internal/api/notification.proto`
- Modify: `internal/service/notification.go`
- Modify: `internal/repository/notification.go`
- Test: `internal/service/filter_alignment_test.go`

- [ ] Step 1: Write a failing test for notification keyword filtering behavior.
- [ ] Step 2: Run the targeted test and verify it fails for missing keyword support.
- [ ] Step 3: Implement minimal proto/service/repository support for `keyword`.
- [ ] Step 4: Run the targeted test and verify it passes.

### Task 2: Audit Queue State Filter

**Files:**
- Modify: `internal/api/audit.proto`
- Modify: `internal/service/audit.go`
- Test: `internal/service/filter_alignment_test.go`

- [ ] Step 1: Write failing tests for `queueState=all|overdue|pending` normalization/filtering.
- [ ] Step 2: Run the targeted tests and verify they fail.
- [ ] Step 3: Implement minimal queue-state validation and filtering logic.
- [ ] Step 4: Run the targeted tests and verify they pass.

### Task 3: Volunteer My Activities Split Endpoint

**Files:**
- Modify: `internal/api/activities.proto`
- Modify: `internal/router` / handler / service files that expose activity list endpoints
- Modify: `internal/service/activities.go`
- Test: `internal/service/filter_alignment_test.go`

- [ ] Step 1: Write failing tests for the new dedicated volunteer-my-activities request path behavior.
- [ ] Step 2: Run the targeted tests and verify they fail.
- [ ] Step 3: Implement the new endpoint by reusing the current registered-only query path.
- [ ] Step 4: Run the targeted tests and verify they pass.

### Task 4: Regenerate Contracts And Verify

**Files:**
- Modify: generated `internal/api/*.pb.go` and docs if generation tooling updates them

- [ ] Step 1: Regenerate generated API files if required by repo workflow.
- [ ] Step 2: Run focused Go tests for the changed service package.
- [ ] Step 3: Run broader verification if the focused suite passes.
