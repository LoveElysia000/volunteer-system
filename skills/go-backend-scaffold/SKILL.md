---
name: go-backend-scaffold
description: Scaffold a new Go backend project from scratch based on the volunteer-system engineering layout (cmd/config/internal/pkg/proto/docs/sql/test), including project structure, base configuration, router/handler/service/repository skeleton, and startup-oriented make/go command setup, without business logic implementation or derived artifact generation. 当用户说“新建一个后端go项目”“从零搭建go后端脚手架”或要求初始化/搭建 Go 后端项目骨架时使用。
---

# Volunteer System Go Backend Scaffold

## Goal

Create a compilable, layered Go backend skeleton that mirrors the current project style while intentionally excluding business-domain implementation.

## Scope

- Focus on structure and engineering baseline only.
- Include: directories, entrypoint, config loading, route wiring skeleton, middleware skeleton, response wrapper, startup-oriented Makefile commands, and README basics.
- Exclude: business rules, real database schema design, production credentials, and feature-specific SQL logic.

## Recommended Structure

```text
<project>/
├── cmd/main.go
├── config/
│   ├── config.go
│   └── config.yaml.example
├── deploy/ddl.sql
├── docs/
├── internal/
│   ├── api/
│   ├── dao/
│   ├── handler/
│   ├── middleware/
│   ├── model/
│   ├── repository/
│   ├── response/
│   ├── router/
│   └── service/
├── pkg/
│   ├── logger/
│   ├── database/
│   └── util/
├── proto/
├── sql/
│   └── ddl/
├── test/
├── Makefile
├── gen.yaml
├── README.md
├── go.mod
└── go.sum
```

## Workflow

### 1) Bootstrap project metadata

- Decide module path, binary name, and default service port.
- Initialize module:
```bash
go mod init <module>
```
- Create base directories and placeholder files before writing detailed code.

### 2) Build minimal runnable skeleton

- `cmd/main.go`: load config, init core dependencies, start HTTP server.
- `config/config.go`: define config structs and `LoadConfig`/`GetConfig`.
- `config/config.yaml.example`: provide safe template values only.
- `internal/router/`: register global middleware and a minimal route group (for example `/api`).
- `internal/handler/`, `internal/service/`, `internal/repository/`: create skeleton interfaces/constructors and one smoke-test endpoint path.
- `internal/response/`: provide unified success/fail response helpers.

### 3) Add tooling workflow (scaffold stage)

- Add and maintain `gen.yaml` in project root as a future placeholder for `gorm/gen`.
- Add Makefile targets aligned with this repository style:
  - required now: `build`, `build-mac`, `run`, `test`, `fmt`, `mod`
  - optional placeholders for future: `install`, `api`, `api-single`, `api-single-mac`, `models`
- Do not execute proto/model generation commands in scaffold stage.
- If generation-related targets are present, keep them documented as future steps only.

### 4) Keep layering strict even in skeleton stage

- `handler`: bind/validate and return response.
- `service`: business orchestration only.
- `repository`: DB access only.
- `router`/`middleware`: wiring and cross-cutting concerns only.
- `pkg`: reusable generic utilities, not endpoint-specific business logic.

### 5) Validate scaffold output

Run baseline checks:
```bash
go mod tidy
go fmt ./...
go test ./...
```
Build check:
```bash
make build-mac
```
or Windows:
```bash
make build
```
Startup smoke check:
```bash
make run
```

## Code Generation Boundary

- This skill should generate project structure and scaffold code files.
- This skill should not generate derived artifacts (for example `*.pb.go`, `docs/openapi.yaml`, `*.gen.go`) in scaffold stage.
- This skill should not generate business-domain implementation logic.
- Generation workflows (`make api`, `make models`, `gentool -c "./gen.yaml"`) are out of scope for scaffold execution and can be run later when needed.

## Output Checklist

- [ ] Project compiles and starts with placeholder configuration.
- [ ] Directory layout matches the volunteer-system layering style.
- [ ] Makefile includes core startup/build/test/fmt/mod commands.
- [ ] Derived artifacts are not generated in scaffold stage.
- [ ] No business-domain implementation is added.
