---
name: go-backend
description: Implement and modify backend features for the volunteer-system Go service (Hertz + GORM + Proto/OpenAPI). 当需求涉及新增接口、改接口、分层改造、AI 助手模块、proto 契约变更、路由/中间件调整、或需要执行 API 生成与后端验证命令时使用。
---

# Volunteer System Go Backend

## Goal

Deliver safe backend changes in this repository with consistent layering, protocol compatibility, and low regression risk.

## Project Map

- Protocol contract: `internal/api/*.proto`
- Generated API artifacts: `internal/api/*.pb.go`, `docs/openapi.yaml`
- Request layer: `internal/handler/`
- Business layer: `internal/service/`
- Data access layer: `internal/repository/`
- Route registration: `internal/router/`
- Shared middleware and helpers: `internal/middleware/`, `pkg/`

## Workflow

### 1) Understand current implementation before coding

- Read existing route, handler, service, and repository files related to the feature before editing.
- Confirm each touched file's responsibility first, then decide where to place new code.
- Reuse existing patterns for request binding, response format, and error handling unless there is a clear exception.

### 2) Confirm scope and affected layers

- Read the related route entry in `internal/router/` and matching request/response models in `internal/api/*.proto`.
- Identify whether the change touches contract, behavior, persistence, or only wiring.
- Prefer smallest possible blast radius before coding.

### 3) Update contract first when API changes

- Edit `internal/api/*.proto` before changing handlers/services if request/response fields or enums change.
- Regenerate artifacts:
```bash
make api
```
or for a single proto based on OS:
```bash
if [ "$(go env GOHOSTOS)" = "windows" ]; then
  make api-single file=internal/api/<name>.proto
else
  make api-single-mac file=internal/api/<name>.proto
fi
```
- Do not manually edit generated API files (`internal/api/*.pb.go`); regenerate from proto and commit together with logic changes.
- Treat model/dao generated files (`internal/model/*.gen.go`, `internal/dao/*.gen.go`) as generated output. When field changes depend on local DB schema, have the project owner run:
```bash
gentool -c "./gen.yaml"
```
then continue development against regenerated files.

### 4) Implement in layer order

- Keep handlers thin in `internal/handler/`: bind/validate request, call service, map response.
- Put business rules and orchestration in `internal/service/`.
- In `service`, call encapsulated methods from `repository`/`pkg`/other layers instead of writing their implementation details inline.
- If a required method does not exist, add it in the proper layer first, then call it from `service`.
- Keep SQL/GORM access in `internal/repository/` and preserve context usage (`db.WithContext(...)`).
- Add/update router wiring in `internal/router/` only after handler exists.
- Follow existing response style (`response.Success`, `response.Fail`, `response.FailWithCode`).

### 5) Respect module-specific invariants

- For AI assistant flows (`assistant_*` files), preserve:
  - session ownership and status checks,
  - message ordering (`seq_no`) guarantees,
  - request quota and usage accounting semantics,
  - fallback behavior when model/runtime fails.
- Keep concurrency-sensitive updates atomic where existing code already enforces atomicity.

### 6) Validate before finishing

- Run formatting and tests:
```bash
go fmt ./...
go test ./...
```
- Build to catch integration issues:
```bash
make build-mac
```
or on Windows:
```bash
make build
```

## Layer Responsibilities

- `internal/api/`: define request/response structures and RPC contracts for frontend-backend interaction.
- `internal/handler/`: handle routing entry, bind/validate request, call service, and return standardized response.
- `internal/model/`: store database model structures (usually generated). Avoid manual edits except allowed constants updates (for example `consts` additions).
- `internal/repository/`: handle database access only (query/insert/update/delete and persistence-level composition).
- `internal/service/`: handle business logic and orchestration only; call repository/other layers through exposed methods.
- `internal/router/`: register route groups, endpoint mappings, and middleware wiring only.
- `internal/middleware/`: implement cross-cutting concerns (auth, CORS, recovery, logging, request context) only.
- `pkg/`: provide reusable utilities not tied to specific business logic (for example deduplication, reverse ordering, time formatting, generic helpers).

## Layer Boundaries (Do Not Cross)

- Do not put business rule branching or workflow orchestration in `handler`; keep it in `service`.
- Do not write raw SQL/GORM query logic directly inside `service`.
- Do not perform request/response DTO binding, protocol mapping, or HTTP formatting inside `service`; keep transport concerns in `handler`/`api`.
- Do not implement repository- or utility-layer logic inline in `service` even if the code is short; extract/add the method in the owning layer and call it.
- Do not place complete business workflow logic inside `repository`.
- Do not return HTTP-layer response objects directly from `repository`; return domain/persistence data only.
- Do not put persistence logic directly in `handler`.
- Do not add business semantics into `model` structures; keep them as schema-oriented data models (except explicit constants files).
- Do not treat `pkg/` as a business module; keep it domain-agnostic.
- For new `pkg` utilities, avoid introducing `internal/*` business dependencies; keep legacy dependencies unchanged unless explicitly refactoring them.
- Do not bypass `repository` from other layers for direct DB calls.
- Do not couple `api` contract definitions with service internals; keep contract evolution explicit via proto changes.
- Do not manually edit generated files (`*.pb.go`, `*.gen.go`) to implement business changes; update sources and regenerate instead.
- Do not put endpoint-specific business rules in `router` or `middleware`.
- Prefer calling encapsulated methods across layers instead of duplicating full logic in the wrong layer.

## Change Checklist

- [ ] Existing code in related modules was read first and responsibilities were confirmed.
- [ ] Route registration matches auth/public intent.
- [ ] Proto + generated files stay in sync when contract changes.
- [ ] Generated files (`*.pb.go`, `*.gen.go`) were not manually business-edited.
- [ ] `service` only orchestrates and calls encapsulated methods; missing capabilities were added in owning layers first.
- [ ] Handler/service/repository/model/pkg responsibilities stay separated.
- [ ] Router/middleware remain cross-cutting and wiring-only (no endpoint business workflow).
- [ ] No cross-layer complete logic exists in the wrong layer.
- [ ] Added logic has tests or at least package-level `go test` coverage run.
- [ ] No unrelated file edits included.
