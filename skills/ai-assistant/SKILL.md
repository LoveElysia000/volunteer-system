---
name: ai-assistant
description: Implement and evolve AI assistant features in volunteer-system, including assistant API contract, handler/router wiring, runtime orchestration (Eino), tool planning/execution, session and message persistence, quota and usage accounting, fallback strategy, and AI config changes. 当需求涉及 AI 会话、对话、工具调用、模型 provider 切换、checkpoint、token/成本统计、配额、或 AI 专项测试时使用。
---

# Volunteer System Ai Assistant

## Goal

Deliver AI-assistant changes that keep conversation reliability, clear layering, and stable observability.

## AI Surface Map

- Contract: `internal/api/assistant.proto`
- Generated API artifacts: `internal/api/assistant.pb.go`, `docs/openapi.yaml`
- Route and transport: `internal/router/assistant.go`, `internal/handler/assistant.go`
- Orchestration: `internal/service/assistant_service.go`
- Runtime: `internal/service/assistant_runtime_*.go`
- Tool chain: `internal/service/assistant_tool_service.go`, `internal/service/assistant_tool_eino_adapter.go`
- Persistence: `internal/repository/assistant.go`
- AI config: `config/config.go`, `config/config.yaml.example`

## Workflow

### 1) Understand scope before editing

- Classify the request first: contract change, orchestration change, runtime/provider change, tool change, persistence/metrics change, or config-only change.
- Read the full chain for the touched capability: router -> handler -> service -> runtime/tool/repository.
- Reuse current error style and response patterns unless a new requirement explicitly changes them.

### 2) Update assistant API contract first when needed

- Edit `internal/api/assistant.proto` for request/response, enum, or endpoint contract changes.
- Regenerate generated API files:
```bash
make api
```
or single-file generation based on host OS:
```bash
if [ "$(go env GOHOSTOS)" = "windows" ]; then
  make api-single file=internal/api/assistant.proto
else
  make api-single-mac file=internal/api/assistant.proto
fi
```
- Do not manually edit `internal/api/*.pb.go`.

### 3) Keep orchestration invariants in `assistant_service.go`

- Validate input and reject unsupported stream mode unless the API and runtime are both upgraded to true streaming.
- Keep session ownership and active-status checks before chat execution.
- Consume daily quota before runtime call.
- Keep outcome accounting reliable: request quota and success/failure token/cost metrics must remain consistent even on early returns.
- Keep message ordering guarantees via `appendAiMessage` retry flow on duplicate `seq_no`.
- Preserve fallback behavior: runtime failure should still produce user-visible safe output and should not leak raw provider internals.
- If adding a new scene, update scene constants, normalization, default title, and system prompt mapping together.

### 4) Keep runtime files framework-focused and persistence-free

- Restrict `assistant_runtime_*.go` to runtime concerns: model init, tool binding, retries, timeout, checkpoint, event parsing, and standardized output.
- Keep runtime output shape in `runtimeChatOutput`; avoid returning handler/repository types from runtime.
- Preserve partial tool-call capture on runtime failure so fallback can use collected tool results.
- Keep provider and key resolution centralized in runtime config helpers; do not hardcode keys or scatter provider branches.
- Preserve checkpoint semantics: TTL `< 0` disables checkpoint, `0` uses default.

### 5) Change tools through both planner and adapter paths

- Add tool name and plan entry in `PlanTools`.
- Add execution branch in `executeOnce` and implement tool-specific function.
- Register Eino tool in `assistant_tool_eino_adapter.go` with typed input struct and normalized map payload.
- Keep timeout/retry/error classification behavior and permission checks in tool service.
- Keep service layer calling tool service methods; do not inline tool-implementation logic into orchestration.

### 6) Keep repository atomicity and auditability

- Keep `internal/repository/assistant.go` focused on DB access only.
- Preserve atomic quota/update SQL semantics in `ConsumeAiRequestQuota` and `AppendAiUsageOutcome`.
- Keep tool-call logging and assistant-message binding flow consistent (`CreateAiToolCall` -> `UpdateAiToolCallMessageID`).

### 7) Update config when runtime behavior changes

- If adding AI options, update both `config/config.go` structs and `config/config.yaml.example`.
- Prefer bounded config helpers (min/max clamping) for timeout, retries, and context limits.

### 8) Validate before completion

- Run formatting and tests:
```bash
go fmt ./...
go test ./internal/service -run 'Assistant|Runtime|Eino|Tool'
go test ./...
```
- Build binary to catch integration errors:
```bash
make build-mac
```
or on Windows:
```bash
make build
```

## Layer Boundaries (Do Not Cross)

- Do not place business workflow decisions in `handler` or `router`.
- Do not write SQL/GORM queries in `assistant_service.go` or runtime files.
- Do not perform HTTP response formatting inside repository or runtime.
- Do not put persistence writes in runtime adapters (`assistant_runtime_*`, `assistant_tool_eino_adapter.go`).
- Do not implement repository/pkg logic inline in service; add methods in owning layer and call them.
- Do not manually business-edit generated files (`*.pb.go`, `*.gen.go`).

## Generated Files Policy

- Treat `internal/api/*.pb.go`, `internal/model/*.gen.go`, and `internal/dao/*.gen.go` as generated outputs.
- Regenerate API outputs from proto changes via `make api` / `make api-single*`.
- If schema changes require regenerated model/dao fields, let the project owner run local DB generation:
```bash
gentool -c "./gen.yaml"
```
then continue coding against regenerated files.

## Change Checklist

- [ ] The full assistant chain was read before editing (router/handler/service/runtime/tool/repository).
- [ ] Contract changes in `assistant.proto` were regenerated; no manual edits in `assistant.pb.go`.
- [ ] Service keeps quota, message ordering, fallback, and usage-accounting invariants intact.
- [ ] Runtime changes keep standardized input/output and preserve partial tool results on failure.
- [ ] Tool changes updated planner, executor, and Eino adapter together.
- [ ] Repository changes preserve atomic quota/usage semantics and tool-call audit flow.
- [ ] Generated files (`*.pb.go`, `*.gen.go`) were updated by generation commands only.
- [ ] AI config struct and example yaml were updated together when adding config fields.
- [ ] AI-focused tests and package tests were run before completion.
