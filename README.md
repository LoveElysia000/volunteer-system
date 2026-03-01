# 环保志愿者服务平台（后端）

> 基于 Go + Hertz 的志愿者管理后端服务，提供注册登录、组织管理、活动管理、审核流、工时管理、导出与站内通知能力。

[![Go Version](https://img.shields.io/badge/Go-1.24.0+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## 项目概览

本仓库是后端 API 服务，主要面向以下场景：

- 志愿者与组织账号注册、登录、身份管理
- 组织活动发布、报名、签到签退、工时结算
- 成员加入/退出组织审核流
- 数据导出（活动、志愿者等）
- 站内通知（通知列表、标记已读）

当前 API 定义以 `internal/api/*.proto` 和 `docs/openapi.yaml` 为准。

## 技术栈

| 组件 | 说明 |
|------|------|
| Web 框架 | [Hertz](https://github.com/cloudwego/hertz) |
| ORM | [GORM](https://gorm.io/) + `gorm/gen` |
| 数据库 | MySQL |
| 缓存 | Redis（可选） |
| 认证 | JWT |
| API 定义 | Protocol Buffers + OpenAPI |

## 目录结构

```text
volunteer-system/
├── cmd/                  # 启动入口
├── config/               # 配置加载与模板
├── docs/                 # OpenAPI 文档
├── internal/
│   ├── api/              # proto 与生成代码
│   ├── handler/          # HTTP 处理层
│   ├── service/          # 业务层
│   ├── repository/       # 数据访问层
│   ├── router/           # 路由注册
│   └── model/dao/        # 数据模型与DAO
├── pkg/                  # 通用组件（logger/db/auth/util）
├── proto/                # 三方 proto 依赖
├── sql/                  # DDL / DML
├── Makefile
└── go.mod
```

## 快速开始

### 1. 环境要求

- Go `1.24+`
- MySQL `8.0+`
- Redis（可选）
- `protoc`（用于代码生成）

### 2. 安装依赖工具

```bash
make install
```

会安装：

- `protoc-gen-go`
- `protoc-gen-openapi`
- `protoc-go-inject-tag`
- `gorm.io/gen/tools/gentool`

### 3. 配置文件

```bash
cp config/config.yaml.example config/config.yaml
```

按实际环境修改 `config/config.yaml`，最少确认：

- `app.host` / `app.port`
- `mysql.*`
- `auth.jwt.secret`

### 4. 初始化数据库

按版本顺序执行 `sql/ddl` 下脚本，直到最新版本（当前为 `ddl_v1.2.1.sql`）。

示例：

```bash
mysql -h127.0.0.1 -uroot -p volunteer_system < sql/ddl/ddl_v1.2.1.sql
```

### 5. 生成代码

```bash
# 全量生成：pb.go + OpenAPI
make api

# 单个 proto 生成 pb.go，并全量重建 OpenAPI（避免丢失已有接口）
make api-single file=internal/api/notification.proto

# Mac/Linux 对应命令
make api-single-mac file=internal/api/notification.proto

# 生成 gorm/gen 代码
make models
```

### 6. 启动服务

```bash
make build
make run
```

默认监听：`http://localhost:1109`

## OpenAPI 文档

- 文档文件：`docs/openapi.yaml`
- 推荐生成方式：`make api`（全量）
- 若使用 `make api-single`，会自动“单文件 pb.go + 全量重建 openapi”，不做增量追加

## Make 命令

| 命令 | 说明 |
|------|------|
| `make install` | 安装开发工具 |
| `make api` | 全量生成 API 代码与 OpenAPI |
| `make api-single file=<file>` | Windows：生成单个 proto 的 pb.go，并全量重建 OpenAPI |
| `make api-single-mac file=<file>` | Mac/Linux：生成单个 proto 的 pb.go，并全量重建 OpenAPI |
| `make models` | 生成 gorm/gen 代码 |
| `make build` | 构建 Windows 可执行文件 |
| `make build-mac` | 构建 Mac/Linux 可执行文件 |
| `make run` | 启动服务 |
| `make test` | 运行测试 |
| `make fmt` | 格式化代码 |
| `make mod` | 整理依赖 |
| `make clean` | 清理构建产物 |
| `make docker-build` | 构建镜像（依赖本地 Dockerfile） |
| `make docker-tag` | 打 tag 到 Harbor 地址 |
| `make docker-push` | 推送镜像到 Harbor |

## 已知限制

- 当前未内置 Swagger UI 路由，接口文档以 `docs/openapi.yaml` 为主。
- `docker-*` 命令依赖本地存在 Dockerfile 及正确仓库配置。

## 贡献

1. Fork 仓库
2. 创建分支：`git checkout -b feature/xxx`
3. 提交：`git commit -m "feat: xxx"`
4. 推送并发起 PR

## 许可证

MIT，详见 [LICENSE](LICENSE)。
