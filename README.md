# 环保志愿者服务平台（后端）

> 基于 Go + Hertz 的志愿者管理后端服务，覆盖注册登录、组织/活动、审核、工时、导出、通知与 AI 助手能力。

[![Go Version](https://img.shields.io/badge/Go-1.24.0+-00ADD8?style=flat&logo=go)](https://golang.org/)

## 功能范围

- 账号与身份：志愿者/组织注册登录、账号状态管理
- 组织与成员：成员申请、审核、角色与状态流转
- 活动闭环：活动发布、报名、签到签退、工时发放/作废
- 审核中心：统一审核记录与审核结果回写
- 通知中心：通知投递、收件箱、已读/归档
- 数据导出：活动与志愿者数据导出
- AI 助手：会话、消息、工具调用与用量统计

API 协议定义以 `internal/api/*.proto` 为准，OpenAPI 产物见 `docs/openapi.yaml`。

## 技术栈

| 组件 | 说明 |
|------|------|
| Web 框架 | [Hertz](https://github.com/cloudwego/hertz) |
| ORM | [GORM](https://gorm.io/) + `gorm/gen` |
| 数据库 | MySQL |
| 缓存 | Redis（可选，初始化失败不阻断服务） |
| 认证 | JWT |
| API 定义 | Protocol Buffers + OpenAPI |

## 环境要求

- Go `1.24+`
- MySQL `8.0+`
- Redis（可选）
- `protoc`（仅在你需要重新生成 pb/openapi 时）

## 快速启动

### 1. 准备配置

```bash
cp config/config.yaml.example config/config.yaml
```

至少确认以下配置项：

- `app.host` / `app.port`
- `mysql.host` / `mysql.port` / `mysql.user` / `mysql.password` / `mysql.database`
- `auth.jwt.secret`
- `ai.*`（如启用 AI 助手）

默认端口来自 `config/config.yaml`，示例配置为 `1109`。

### 2. 初始化数据库

新环境推荐直接使用全量建表脚本：

```bash
mysql -h127.0.0.1 -uroot -p volunteer_system < deploy/ddl.sql
```

说明：

- `deploy/ddl.sql`：当前项目完整表结构（适合新库初始化）
- `sql/ddl/*.sql`：历史版本增量脚本（适合已有库按版本升级）

### 3. 启动服务

```bash
make build
make run
```

或直接：

```bash
go run cmd/main.go -c server
```

默认监听：`http://localhost:1109`

## 常用开发命令

| 命令 | 说明 |
|------|------|
| `make install` | 安装开发工具（gentool / protoc 插件等） |
| `make api` | 全量生成 pb.go + OpenAPI |
| `make api-single file=<file>` | Windows：生成单个 proto 的 pb.go，并全量重建 OpenAPI |
| `make api-single-mac file=<file>` | Mac/Linux：生成单个 proto 的 pb.go，并全量重建 OpenAPI |
| `make models` | 生成 `gorm/gen` DAO 代码 |
| `make build` | 构建 `volunteer-system.exe` |
| `make build-mac` | 构建 `volunteer-system` |
| `make run` | 启动服务 |
| `make test` | 运行测试 |
| `make fmt` | 代码格式化 |
| `make mod` | 整理依赖 |

## 目录结构

```text
volunteer-system/
├── cmd/                  # 启动入口
├── config/               # 配置加载与模板
├── deploy/               # 部署脚本（全量 DDL）
├── docs/                 # OpenAPI 文档
├── internal/
│   ├── api/              # proto 与生成代码
│   ├── handler/          # HTTP 处理层
│   ├── service/          # 业务层
│   ├── repository/       # 数据访问层
│   ├── router/           # 路由注册
│   └── model/dao/        # 数据模型与 DAO
├── pkg/                  # 通用组件（logger/db/auth/util）
├── proto/                # 三方 proto 依赖
├── sql/                  # 历史 DDL / DML 增量脚本
├── test/                 # 测试目录（集成测试/fixtures/脚本）
├── Makefile
└── go.mod
```

## 文档与接口

- OpenAPI 文件：`docs/openapi.yaml`
- 推荐生成方式：`make api`

当前仓库未内置 Swagger UI 路由，如需在线查看可自行接入 Swagger UI/ReDoc。
