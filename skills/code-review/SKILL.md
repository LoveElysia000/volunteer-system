---
name: code-review
description: Review completed code changes in volunteer-system to find bugs, behavioral regressions, layering violations, unsafe AI/runtime changes, and missing tests before merge. 当需求开发完成后、提交前自检、联调前验收、或用户明确要求“review/代码审查/检查风险”时使用。
---

# Volunteer System Code Review

## Goal

Produce a risk-focused code review report that prioritizes defects and regressions over style feedback.

## Review Workflow

### 1) Define review scope first

- Identify changed files and focus on modified hunks first.
- Ignore unrelated dirty files unless they directly affect reviewed behavior.
- Read requirements/intent (if available) and compare implementation against expected behavior.

### 2) Review by severity, not by file order

- Prioritize:
  - functional correctness bugs,
  - data consistency and concurrency risks,
  - security/permission leaks,
  - API contract and compatibility regressions,
  - missing tests for high-risk logic.
- Leave low-impact style notes for the end only if they matter.

### 3) Enforce project layering boundaries

- `handler`: bind/validate input, call service, return response only.
- `service`: business orchestration only; call repo/pkg methods, do not inline SQL or transport formatting.
- `repository`: DB access and persistence composition only.
- `router`/`middleware`: wiring and cross-cutting concerns only.
- `pkg`: generic reusable utilities, not endpoint-specific business workflows.

### 4) Verify generated-file and contract consistency

- If `internal/api/*.proto` changed, verify generated API artifacts were regenerated and not hand-edited (`*.pb.go`, `docs/openapi.yaml`).
- Treat `internal/model/*.gen.go` and `internal/dao/*.gen.go` as generated files; review for consistency, not manual business edits.
- Check handler/service field usage matches updated proto contract.

### 5) Run AI-specific checks when assistant modules are touched

For changes under `assistant_*` files, verify:
- session ownership/status validation is preserved,
- request quota consumption and usage aggregation semantics remain atomic,
- message ordering (`seq_no`) and retry behavior are not broken,
- runtime failure still returns safe fallback output,
- tool planner, executor, and adapter remain consistent after tool changes,
- config changes update both struct and example yaml.

### 6) Validate with commands when possible

Run representative checks (adapt to scope):
```bash
go test ./internal/service -run 'Assistant|Runtime|Eino|Tool'
go test ./...
```
If generation changed:
```bash
make api
```
Build check:
```bash
make build-mac
```
or Windows:
```bash
make build
```

## Report Format

Return findings first, ordered by severity:
- Include: severity, concise title, impact, and exact file/line reference.
- Keep each finding actionable (what is wrong and why it matters).
- If no findings, explicitly state no defects found and list residual risks/testing gaps.

Use this structure:
1. `Findings` (highest severity first)
2. `Open Questions / Assumptions` (only if blocking certainty)
3. `Change Summary` (brief)
4. `Validation Executed` (commands run and notable outcomes)

## Severity Guide

- `P0`: data loss/corruption, critical security issue, or production outage risk.
- `P1`: clear functional bug, permission bypass, significant regression.
- `P2`: medium risk maintainability/edge-case issue that can become a bug.
- `P3`: low-risk improvement or clarity issue.

## Review Checklist

- [ ] Scope is clear and unrelated workspace noise is excluded.
- [ ] Business behavior matches intended requirement and API contract.
- [ ] Layer boundaries are preserved (`handler/service/repository/router/pkg`).
- [ ] Generated files are treated as generated (no manual business edits).
- [ ] AI invariants are checked for assistant-related changes.
- [ ] High-risk paths have corresponding tests or explicit test-gap notes.
- [ ] Final report lists findings first with precise file references.
