# 环保志愿者服务平台

> 一个连接环保志愿者与环保活动的平台，促进环保志愿服务的社会化、规范化管理。

[![Go Version](https://img.shields.io/badge/Go-1.24.0+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## 项目概述

环保志愿者服务平台是一个基于 Go 语言开发的全栈 Web 应用，旨在为环保组织、志愿者和管理员提供一个统一的平台，实现志愿者活动的发布、报名、管理和统计功能。

## 功能模块

### 志愿者管理
- 志愿者注册与个人信息管理
- 技能标签系统和志愿服务记录
- 积分类别和等级评定

### 活动管理
- 环保活动发布与管理
- 线上报名与审核机制
- 活动成果记录与展示

### 组织管理
- 环保组织注册与认证
- 组织内成员管理
- 活动发布权限控制

### 成员关系与审核
- 志愿者加入/退出组织申请
- 成员状态变更与统计分析
- 审核记录快照与审批留痕

### 数据导入导出
- 志愿者信息批量导入/导出
- 活动数据批量处理
- Excel/CSV 格式支持

### 证书生成
- 志愿者服务证书自动生成
- 证书模板管理
- PDF 格式证书下载

## 技术栈

### 后端

| 组件 | 说明 |
|------|------|
| **框架** | [Hertz v0.10.3](https://github.com/cloudwego/hertz) - CloudWeGo 高性能 HTTP 框架 |
| **数据库** | MySQL 8.0+ + [GORM v1.26.0](https://gorm.io/) |
| **缓存** | Redis + [go-redis v9.5.1](https://github.com/redis/go-redis/) |
| **认证** | JWT (jsonwebtoken v5.3.0) + Token 轮换机制 |
| **API** | Protocol Buffers + OpenAPI |
| **日志** | 自研日志库 (`pkg/logger`) - 文件+控制台双输出 |

### 前端 (规划中)

| 组件 | 说明 |
|------|------|
| **框架** | Vue.js + Tailwind CSS |
| **地图服务** | 地理位置API集成 |
| **图表** | 数据可视化组件 |

## 📁 项目结构

```
volunteer-system/
├── cmd/                    # 命令行入口
│   ├── main.go            # 主程序入口
│   └── cli/               # CLI命令
│       └── server.go      # 服务器启动逻辑
├── config/                # 配置管理
│   ├── config.go          # 配置结构体定义
│   └── config.yaml        # 运行时配置文件
├── internal/              # 内部业务包
│   ├── api/               # Protobuf API 定义
│   ├── dao/               # 数据访问层 (GORM生成)
│   ├── handler/           # HTTP 请求处理器
│   ├── middleware/        # 中间件 (auth、recovery、cors)
│   ├── model/             # 数据模型
│   ├── repository/        # 仓储模式实现
│   ├── response/          # 统一响应封装
│   ├── router/            # 路由定义
│   └── service/           # 业务逻辑层
├── pkg/                   # 公共可复用包
│   ├── auth/              # JWT 认证管理器
│   ├── database/          # 数据库连接管理
│   │   ├── mysql/         # MySQL 连接
│   │   └── redis/         # Redis 连接
│   ├── logger/            # 日志工具 (文件+控制台)
│   ├── util/              # 通用工具函数
│   └── validator/         # 输入验证
├── proto/                 # Protobuf 定义
├── sql/                   # 数据库脚本
│   ├── ddl/               # 数据定义语言
│   └── dml/               # 数据操作语言
├── logs/                  # 日志文件目录
├── docs/                  # 文档
└── Makefile               # 构建配置
```

## 🔧 快速开始

### 前提条件

| 要求 | 版本 |
|------|------|
| Go | 1.24.0+ |
| MySQL | 8.0+ |
| Redis | 5.0+ (可选) |
| Protobuf编译器 | 最新版 |

### 1. 克隆项目

```bash
git clone <repository-url>
cd volunteer-system
```

### 2. 安装依赖

```bash
make install
```

安装的开发依赖工具：
- `gorm.io/gen/tools/gentool` - GORM 模型生成
- `protoc-gen-go` - Protobuf Go 代码生成
- `protoc-gen-openapi` - OpenAPI 文档生成
- `protoc-go-inject-tag` - Protobuf tag 注入

### 3. 配置环境

编辑 `config/config.yaml` 文件：

```yaml
app:
  name: "Volunteer System"
  env: "development"
  host: "0.0.0.0"
  port: 1109

mysql:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "your-password"
  database: "volunteer_system"

redis:
  host: "127.0.0.1"
  port: 6379

logging:
  level: "info"
  console: true
  file: "./logs/app.log"
```

### 4. 生成代码

```bash
# 生成 API 代码 (Protobuf)
make api

# 生成数据库模型代码 (GORM)
make models
```

### 5. 构建运行

```bash
# 构建项目
make build

# 运行服务
make run
```

服务默认运行在 `http://localhost:1109`

## API 文档

API 文档使用 OpenAPI 规范生成，可通过以下方式访问：

| 方式 | 说明 |
|------|------|
| **静态文档** | 查看 `docs/openapi.yaml` 文件 |
| **Swagger UI** (开发中) | 访问 `http://localhost:1109/swagger/` |

## 会员关系与审核（2026-02）

### 认证与响应约定

- 除登录/注册外，`/api/*` 默认需要携带 `Authorization: Bearer <access_token>`。
- 接口响应统一为：

```json
{
  "code": 200,
  "msg": "OK",
  "data": {}
}
```

### 关键业务流程

1. 志愿者提交加入组织：`POST /api/memberships/join`  
   服务端会创建 `audit_records` 待审核记录（`target_type=3, operation_type=1, status=1`）。
2. 组织账号审核加入申请：`POST /api/audits/approval` 或 `POST /api/audits/rejection`。  
   审核通过后才会真正写入 `org_members`。
3. 志愿者提交退出组织：`POST /api/memberships/leave`。  
   服务端会创建退出审核记录（`operation_type=3, status=1`）。
4. 组织账号审核退出申请通过后，成员状态更新为 `已退出`（`status=4`）。
5. 组织管理者可直接更新成员状态：`POST /api/memberships/{membershipId}/status`，仅允许 `2/3/4`（不能直接改为待审核）。

### 状态枚举

| 类型 | 值 | 说明 |
|------|----|------|
| 成员状态 | `1` | 待审核 |
| 成员状态 | `2` | 正式成员（已通过） |
| 成员状态 | `3` | 已拒绝 |
| 成员状态 | `4` | 已退出 |
| 成员角色 | `1` | 普通成员 |
| 成员角色 | `2` | 管理员 |
| 成员角色 | `3` | 负责人 |
| 审核状态 | `1` | 待审核 |
| 审核状态 | `2` | 审核通过 |
| 审核状态 | `3` | 审核拒绝 |
| 审核操作类型 | `1` | 新增 |
| 审核操作类型 | `2` | 更新 |
| 审核操作类型 | `3` | 删除 |

### 会员与审核 API 索引

| 接口 | 方法 | 说明 | 主要权限约束 |
|------|------|------|-------------|
| `/api/memberships/join` | `POST` | 志愿者提交加入组织申请 | 仅志愿者本人可提交 |
| `/api/memberships/leave` | `POST` | 志愿者提交退出组织申请 | 仅该成员关系对应志愿者本人 |
| `/api/organizations/{organizationId}/members` | `GET` | 组织成员列表 | 仅该组织管理者 |
| `/api/volunteers/{volunteerId}/organizations` | `GET` | 志愿者组织列表 | 仅志愿者本人可查自己 |
| `/api/memberships/{membershipId}/status` | `POST` | 组织侧更新成员状态 | 仅该组织管理者；状态仅 `2/3/4` |
| `/api/memberships/stats` | `GET` | 成员统计 | 组织管理者；`organizationId` 可选 |
| `/api/audits/volunteer-join-org/pending` | `POST` | 待审核列表 | 需登录 |
| `/api/audits/approval` | `POST` | 审核通过 | 组织账号 |
| `/api/audits/rejection` | `POST` | 审核驳回 | 组织账号 |
| `/api/audits/records/:id` | `GET` | 审核记录详情 | 需登录 |

## 数据库迁移

- 本次版本新增 `sql/ddl/ddl_v1.1.6.sql`，用于将 `org_members` 的 `(org_id, volunteer_id)` 升级为唯一索引，避免同一组织出现重复成员关系。
- 建议按版本顺序执行 DDL 脚本（`sql/ddl/ddl_v1.1.0.sql` -> 最新版本）。
- 执行示例：

```bash
mysql -h127.0.0.1 -uroot -p volunteer_system < sql/ddl/ddl_v1.1.6.sql
```

## 用户权限体系

平台支持 2 种角色：

| 角色 | 权限 |
|------|------|
| **志愿者** | 浏览和报名活动、管理个人信息、查看服务记录和积分 |
| **组织方** | 发布和管理本组织活动、审核志愿者报名、查看统计数据 |

## 日志系统

项目内置日志工具 `pkg/logger`，支持：

- **日志级别**: DEBUG, INFO, WARN, ERROR
- **双输出**: 同时写入文件和控制台
- **线程安全**: 使用互斥锁保护并发写入

### 使用示例

```go
import "volunteer-system/pkg/logger"

// 初始化 (在启动时调用一次)
logger.Init("info", true, "./logs/app.log")

// 获取logger实例
log := logger.GetLogger()

// 写入日志
log.Info("服务启动成功")
log.Error("连接失败: %v", err)
log.Warn("内存使用率较高")
log.Debug("调试信息")
```

### 日志格式

```
2026-01-29 15:30:45 [INFO] 服务启动成功
2026-01-29 15:30:46 [ERROR] 连接失败: connection refused
```

## 测试

```bash
make test
```

## Make 命令说明

| 命令 | 说明 |
|------|------|
| `make install` | 安装开发依赖工具 |
| `make api` | 批量生成API代码 |
| `make api-single file=<file>` | 生成单个proto文件代码 |
| `make api-single-mac file=<file>` | Mac/Linux 下生成单个proto文件代码 |
| `make build` | 构建可执行文件 |
| `make build-mac` | 构建 Mac/Linux 可执行文件 |
| `make run` | 运行服务 |
| `make clean` | 清理编译产物 |
| `make test` | 运行测试 |
| `make fmt` | 格式化代码 |
| `make mod` | 整理依赖 |
| `make models` | 生成数据库模型代码 |
| `make docker-build` | 构建Docker镜像 |
| `make docker-tag` | 镜像打 tag 到 Harbor 地址 |
| `make docker-push` | 推送镜像到 Harbor |

## 部署

### Docker 部署

```bash
# 构建镜像
make docker-build

# 推送镜像
make docker-push
```

### 手动部署

1. 构建项目：`make build`
2. 配置生产环境：
   - 设置环境变量 `VOLUNTEER_APP_ENV=production`
   - 配置生产数据库连接
   - 设置SSL证书（如需要）
3. 启动服务：`./volunteer-system.exe -c server`

### 环境变量配置

| 前缀 | 说明 |
|------|------|
| `VOLUNTEER_APP_*` | 应用相关配置 |
| `VOLUNTEER_MYSQL_*` | MySQL连接配置 |
| `VOLUNTEER_REDIS_*` | Redis连接配置 |
| `VOLUNTEER_AUTH_JWT_*` | JWT认证配置 |

## 贡献指南

我们欢迎社区贡献！请遵循以下步骤：

1. Fork 项目
2. 创建功能分支：`git checkout -b feature/AmazingFeature`
3. 提交更改：`git commit -m 'Add some AmazingFeature'`
4. 推送分支：`git push origin feature/AmazingFeature`
5. 创建 Pull Request

## 许可证

MIT License - 查看 [LICENSE](LICENSE) 文件了解详情。

## 致谢

- [CloudWeGo](https://www.cloudwego.io/) - 提供高性能的 Hertz 框架
- [GORM](https://gorm.io/) - 优秀的 Go ORM 库
- 所有为环保事业贡献的志愿者
