# 环保志愿者服务平台后端（volunteer-system）

> 基于 Go + Hertz 的志愿者管理后端，覆盖注册登录、组织/活动、审核、工时、通知、导出与 AI 助手能力。

[![Go Version](https://img.shields.io/badge/Go-1.24.0+-00ADD8?style=flat&logo=go)](https://golang.org/)

## 功能概览

- 账号与身份：志愿者/组织注册登录、账号状态管理、JWT 鉴权
- 组织与成员：成员申请、审核、角色与状态流转
- 活动闭环：活动发布、报名、签到签退、工时发放/作废
- 审核中心：统一审核记录与审核结果回写
- 通知中心：通知投递、收件箱、已读/归档
- 数据导出：志愿者与活动数据导出，支持运营报表导出
- 权限管理：RBAC 角色（`super_admin`/`org_owner`/`volunteer`）、权限、账号绑定与变更日志
- AI 助手：会话、消息、工具调用、SSE 流式输出与用量统计

接口协议以 `internal/api/*.proto` 为准，OpenAPI 产物见 `docs/openapi.yaml`。

## 技术栈

| 组件 | 说明 |
|------|------|
| Web 框架 | [Hertz](https://github.com/cloudwego/hertz) |
| ORM | [GORM](https://gorm.io/) + `gorm/gen` |
| 数据库 | MySQL 8.0+ |
| 缓存 | Redis（可选，初始化失败不阻断服务） |
| 认证 | JWT |
| API 定义 | Protocol Buffers + OpenAPI |
| AI 能力 | CloudWeGo Eino + OpenAI 兼容模型 |

## 环境要求

- Go `1.24+`
- MySQL `8.0+`
- Redis（可选）
- `protoc`（仅在需要重新生成 pb.go/OpenAPI 时）

## 快速启动

### 1. 准备配置

```bash
cp config/config.yaml.example config/config.yaml
```

建议至少确认以下配置：

- `app.host` / `app.port`
- `mysql.host` / `mysql.port` / `mysql.user` / `mysql.password` / `mysql.database`
- `auth.jwt.secret`
- `ai.*`（启用 AI 助手时）

默认端口来自 `config/config.yaml`，示例为 `1109`。

### 2. 初始化数据库

新库初始化（推荐）：

```bash
mysql -h127.0.0.1 -uroot -p volunteer_system < deploy/ddl.sql
```

脚本说明：

- `deploy/ddl.sql`：当前完整表结构（新库直接使用）
- `sql/ddl/*.sql`：历史增量 DDL（已有库按版本升级）
- `sql/dml/dml_v1.0.0.sql`：唯一索引修复与重复数据检查
- `sql/dml/dml_v1.1.0.sql`：`level_rules` 初始化
- `sql/dml/dml_v1.3.2_rbac.sql`：RBAC 初始化（空库版，建议首次执行一次）

### 3. 启动服务

通用方式（推荐）：

```bash
go run cmd/main.go -c server
```

按平台构建后启动：

```bash
# Windows
make build
make run

# macOS / Linux
make build-mac
./volunteer-system -c server
```

默认监听地址：`http://127.0.0.1:1109`

## Docker 部署（单机/手动）

用于本地或手动部署（镜像本地构建）：

```bash
cp .env.example .env
docker compose up -d --build
```

首次启动会自动执行 `deploy/ddl.sql` 初始化数据库。  
默认 `MySQL/Redis` 仅绑定到服务器本机（`127.0.0.1`），不会直接暴露到公网。

## CI/CD 自动化部署（GitHub Actions + GHCR + SSH）

### 1. 服务器首次准备

1. 安装 Docker 与 Docker Compose 插件
2. 克隆仓库到固定目录（示例：`/opt/volunteer-system`）
3. 准备生产变量：

```bash
cd /opt/volunteer-system
cp .env.example .env
```

`APP_IMAGE` 请改为实际镜像地址，例如：`ghcr.io/<github-username>/volunteer-system`。  
其余至少配置：

- `MYSQL_ROOT_PASSWORD`
- `MYSQL_PASSWORD`
- `APP_SECRET_KEY`
- `JWT_SECRET`

### 2. GitHub 仓库 Secrets

在仓库 Settings -> Secrets and variables -> Actions 中新增：

- `SERVER_HOST`：服务器公网或内网地址
- `SERVER_PORT`：SSH 端口（通常 `22`）
- `SERVER_USER`：SSH 用户
- `SERVER_SSH_KEY`：私钥内容（PEM）
- `SERVER_APP_DIR`：服务器项目路径（如 `/opt/volunteer-system`）
- `GHCR_USERNAME`：用于服务器拉镜像的 GitHub 用户名
- `GHCR_TOKEN`：用于服务器 `docker login ghcr.io` 的 Token（建议 PAT，含 `read:packages`）

### 3. 工作流说明

- CI：`.github/workflows/ci.yml`
  - 在 `pull_request` / `push main` 触发
  - 运行 `go test ./...`
  - 构建镜像做可构建性检查（不推送）
- CD：`.github/workflows/cd.yml`
  - 在 `push main` 或手动触发
  - 构建并推送镜像到 GHCR（`latest` + `git sha`）
  - SSH 到服务器执行 `./scripts/deploy.sh`

### 4. 部署脚本与生产编排

- 生产编排：`docker-compose.prod.yml`（使用远端镜像，不在服务器本地构建）
- 生产配置模板：`config/config.prod.yaml`
- 部署脚本：`scripts/deploy.sh`
  - 支持从环境变量读取 `APP_IMAGE` 与 `IMAGE_TAG`
  - 执行 `docker compose pull app` + `up -d`

服务器手动部署（用于首发或回滚）：

```bash
cd /opt/volunteer-system
export IMAGE_TAG=latest
./scripts/deploy.sh
```

## 常用开发命令

| 命令 | 说明 |
|------|------|
| `make install` | 安装开发工具（gentool / protoc 插件等） |
| `make api` | 全量生成 `pb.go` + OpenAPI |
| `make api-single file=<file>` | Windows：生成单个 proto 并重建 OpenAPI |
| `make api-single-mac file=<file>` | macOS/Linux：生成单个 proto 并重建 OpenAPI |
| `make models` | 生成 `gorm/gen` DAO 代码 |
| `make build` | 构建 Windows 可执行文件 `volunteer-system.exe` |
| `make build-mac` | 构建 macOS/Linux 可执行文件 `volunteer-system` |
| `make run` | Windows 下启动服务（依赖 `volunteer-system.exe`） |
| `make test` | 运行测试 |
| `make fmt` | 代码格式化 |
| `make mod` | 整理依赖 |

## 接口与文档

- 协议定义：`internal/api/*.proto`
- OpenAPI 文件：`docs/openapi.yaml`
- 推荐重新生成方式：`make api`

Swagger UI：

- 默认在 `app.env=development` 时启用
- 启动服务后访问：`http://127.0.0.1:1109/swagger/`
- 读取文档源：`/swagger/openapi.yaml`（对应仓库文件 `docs/openapi.yaml`）

## 项目结构

```text
volunteer-system/
├── cmd/                  # 启动入口
├── config/               # 配置加载与配置模板
├── deploy/               # 部署脚本（全量 DDL）
├── docs/                 # OpenAPI、发布/计划文档
├── internal/
│   ├── api/              # proto 与生成代码
│   ├── handler/          # HTTP 处理层
│   ├── service/          # 业务服务层
│   ├── repository/       # 数据访问封装
│   ├── dao/              # gorm/gen 生成 DAO
│   ├── model/            # 领域模型
│   ├── middleware/       # 中间件
│   └── router/           # 路由注册
├── pkg/                  # 通用组件（logger/db/auth/util/ai）
├── proto/                # 第三方 proto 依赖
├── sql/                  # 版本化 DDL / DML
├── test/                 # 测试目录
├── Makefile
└── go.mod
```

## 最近迭代（P0，2026-03）

- 权限模型：新增 RBAC 表结构与种子数据  
  `sql/ddl/ddl_v1.3.1.sql`、`sql/dml/dml_v1.3.1.sql`
- 运营分析：新增漏斗接口  
  `GET /api/analytics/org/funnel?orgId=&start=&end=`
- 导出增强：新增周/月运营模板导出  
  `POST /api/admin/export/ops-report`
- 通知增强：新增 `signup_rejected`、`activity_canceled` 事件，支持可配置邮件通道
- AI 助手：新增流式接口  
  `POST /api/assistant/chat/stream`

SSE 使用说明：

- 客户端建议设置 `Accept: text/event-stream`
- 事件类型：`start`、`delta`、`tool`、`usage`、`done`、`error`
