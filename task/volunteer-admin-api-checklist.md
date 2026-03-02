# 志愿者端与管理员端接口清单（待补功能版）

更新时间：2026-02-27
适用范围：`/Users/Ein/project2/volunteer-system`

## 1. 说明

本文将待补功能拆成可执行接口任务，按 `P0/P1` 标记优先级，并给出：
- API 路径与方法
- 请求/响应关键字段
- 权限要求
- 数据表与迁移建议
- 代码落点建议

---

## 2. 志愿者端 P0 接口清单

### 2.1 首页概览（仪表盘）

| 项目 | 内容 |
| --- | --- |
| 优先级 | P0 |
| 实现状态 | 🟡 部分完成（接口与核心字段已实现；`records` 历史回填/来源追踪仍待补） |
| 新增 API | `GET /api/volunteers/home/summary` |
| 请求参数 | 无（从 token 获取当前账号） |
| 响应字段 | `nickname` `level` `stats.points` `stats.hours` `stats.activityCount` `monthlyGrowth` `needHoursToNextLevel` |
| 权限 | 仅志愿者本人 |
| 依赖表 | `volunteers`（已有） `records`（新增） `level_rules`（新增） |
| 迁移建议 | 新增 `sql/ddl/ddl_v1.2.0.sql`：创建 `records`、`level_rules`，并补 `volunteers.level_id/total_points` |
| 代码落点 | `internal/api/volunteer.proto` `internal/service/volunteer.go` `internal/router/volunteer.go` |

---

### 2.2 活动列表动态报名状态

| 项目 | 内容 |
| --- | --- |
| 优先级 | P0 |
| 实现状态 | ✅ 已实现（`internal/service/activities.go` 动态计算 `isRegistered`） |
| 现有 API | `POST /api/activities`（升级） |
| 请求参数 | 保持现有分页/状态 |
| 响应补强 | `ActivityItem.isRegistered` 必须按当前用户动态返回 |
| 权限 | 登录用户（志愿者/组织都可看列表，是否带 `isRegistered` 由身份决定） |
| 依赖表 | `activity_signups` |
| 迁移建议 | 无 |
| 代码落点 | `internal/service/activities.go` `ActivityList`，复用 `GetUserSignupMap` |

---

### 2.3 我的活动审核状态补全

| 项目 | 内容 |
| --- | --- |
| 优先级 | P0 |
| 实现状态 | ✅ 已实现（返回 `signupStatus`、`auditReason`） |
| 现有 API | `POST /api/activities/my`（升级） |
| 请求参数 | 保持现有 |
| 响应补强 | 增加 `signupStatus` `auditReason`（驳回原因） |
| 权限 | 仅志愿者本人 |
| 依赖表 | `activity_signups` `audit_records` |
| 迁移建议 | 如 `activity_signups` 无驳回原因字段，则仅从 `audit_records.reject_reason` 取最近审核结果 |
| 代码落点 | `internal/api/activities.proto` `internal/service/activities.go` |

---

### 2.4 志愿者资料变更提审（替代直改）

| 项目 | 内容 |
| --- | --- |
| 优先级 | P0 |
| 实现状态 | ✅ 已实现（复用 `audit_records`，按 `scene=volunteer_profile_update` 区分） |
| 新增 API | `POST /api/volunteers/profile-change/submit` |
| 请求参数 | `realName` `gender` `birthday` `avatarUrl` `introduction`（按需） |
| 响应字段 | `auditId` `status` |
| 权限 | 仅志愿者本人 |
| 依赖表 | 直接复用 `audit_records` |
| 迁移建议 | 使用 `target_type=volunteer` + `operation_type=update` 区分资料变更审核；不新增专表 |
| 代码落点 | `internal/api/volunteer.proto` `internal/service/volunteer.go` `internal/service/audit.go` |

---

### 2.5 志愿者实名认证提审

| 项目 | 内容 |
| --- | --- |
| 优先级 | P0 |
| 实现状态 | ✅ 已实现（复用 `audit_records`，`scene=volunteer_real_name_verify`） |
| 新增 API | `POST /api/volunteers/real-name/submit` |
| 请求参数 | `realName` `idCard` |
| 响应字段 | `auditId` `status` |
| 权限 | 仅志愿者本人 |
| 依赖表 | `audit_records` `volunteers` |
| 迁移建议 | 无新增表；审核提交时将 `volunteers.audit_status` 置为 `pending` |
| 代码落点 | `internal/api/volunteer.proto` `internal/service/volunteer.go` `internal/service/audit.go` |

---

## 3. 管理员端 P0 接口清单

### 3.1 统一待审核列表（多目标）

| 项目 | 内容 |
| --- | --- |
| 优先级 | P0 |
| 实现状态 | ✅ 已实现（新增统一多目标待审接口：`POST /api/audits/pending`） |
| 新增 API | `POST /api/audits/pending` |
| 请求参数 | `targetType[]` `status[]` `keyword` `page` `pageSize` |
| 响应字段 | `id` `targetType` `targetId` `title` `subTitle` `creatorId` `createdAt` |
| 权限 | 组织管理员 |
| 依赖表 | `audit_records` |
| 迁移建议 | 视查询压力补索引：`(target_type,status,created_at)`、`(creator_id,status)` |
| 代码落点 | `internal/api/audit.proto` `internal/service/audit.go` `internal/router/audit.go` |

---

### 3.2 活动报名审核批处理

| 项目 | 内容 |
| --- | --- |
| 优先级 | P0 |
| 实现状态 | ❎ 不实施（保留单条审核策略：`POST /api/audits/approval`、`POST /api/audits/rejection`） |
| 新增 API | 无（不新增批处理接口） |
| 请求参数 | 复用现有单条审核参数：`id` `reason` |
| 响应字段 | 复用现有单条审核响应 |
| 权限 | 组织管理员，且仅可操作本组织活动报名审核单（单条） |
| 依赖表 | `audit_records` `activity_signups` `activities` |
| 迁移建议 | 无 |
| 代码落点 | 维持现有实现：`internal/api/audit.proto` `internal/service/audit.go` |
| 决策说明 | 审核采用单条操作，降低误审风险并简化权限与失败处理 |

---

### 3.3 组织停启与账号状态联动

| 项目 | 内容 |
| --- | --- |
| 优先级 | P0 |
| 实现状态 | ✅ 已实现（`disable/enable` 已联动 `sys_accounts.status`） |
| 现有 API | `POST /api/organizations/:id/disable` `POST /api/organizations/:id/enable`（升级） |
| 行为要求 | 停用组织时同步设置 `sys_accounts.status=0`；启用时恢复 `1` |
| 权限 | 组织管理员/平台管理员（按你当前权限模型收敛） |
| 依赖表 | `organizations` `sys_accounts` |
| 迁移建议 | 无 |
| 代码落点 | `internal/service/organization.go` `internal/repository/organization.go` `internal/repository/user.go` |

---

### 3.4 活动自动归档任务

| 项目 | 内容 |
| --- | --- |
| 优先级 | P0 |
| 实现状态 | ❎ 不实施（当前阶段不纳入） |
| 任务类型 | 定时任务（非 HTTP） |
| 触发周期 | 每小时一次 |
| 逻辑 | `start_time < NOW() and status=1` -> 更新为 `status=2` |
| 权限 | 系统任务 |
| 依赖表 | `activities` |
| 迁移建议 | 无 |
| 代码落点 | `cmd/main.go`（注册任务）+ `internal/service/activities.go`（归档方法） |
| 决策说明 | 当前版本优先保障核心业务链路，自动归档任务后置 |

---

## 4. P1 接口清单（运营增强）

### 4.1 导入导出

| 项目 | 内容 |
| --- | --- |
| 优先级 | P1 |
| 实现状态 | 🟡 部分完成（仅导出已实现，且为 `POST /api/admin/export/*`；导入与任务查询未实现） |
| 新增 API | `POST /api/admin/import/volunteers` `POST /api/admin/import/activities` `GET /api/admin/import/tasks/:id` `GET /api/admin/export/volunteers` `GET /api/admin/export/activities` |
| 请求参数 | 文件上传/筛选条件 |
| 响应字段 | 导入任务ID、失败行回执、导出下载链接 |
| 依赖表 | 建议新增 `import_tasks` `import_task_details` |
| 代码落点 | 新增 `internal/api/import_export.proto` 与对应 handler/service/repository |

---

### 4.2 证书系统

| 项目 | 内容 |
| --- | --- |
| 优先级 | P1 |
| 实现状态 | ❌ 未实现 |
| 新增 API | `POST /api/certificates/templates` `PUT /api/certificates/templates/:id` `POST /api/certificates/generate` `GET /api/certificates/:id/download` |
| 请求参数 | 模板内容、活动/志愿者范围 |
| 响应字段 | `certificateId` `downloadUrl` |
| 依赖表 | 建议新增 `certificate_templates` `certificates` |
| 代码落点 | 新增 `internal/api/certificate.proto` 与对应模块 |

---

### 4.3 通知中心

| 项目 | 内容 |
| --- | --- |
| 优先级 | P1 |
| 实现状态 | ❌ 未实现 |
| 新增 API | `GET /api/notifications` `POST /api/notifications/read` |
| 请求参数 | 分页、是否未读 |
| 响应字段 | `title` `content` `bizType` `bizId` `readStatus` `createdAt` |
| 依赖表 | 建议新增 `notifications` |
| 代码落点 | 新增 `internal/api/notification.proto` 与对应模块 |

---

### 4.4 推荐能力

| 项目 | 内容 |
| --- | --- |
| 优先级 | P1 |
| 实现状态 | ❌ 未实现 |
| 新增 API | `GET /api/activities/recommend` |
| 请求参数 | `page` `pageSize` |
| 响应字段 | 活动列表 + `recommendReason` |
| 依赖表 | 轻量规则版可先不改表；AI版需 `activities.embedding_vector` |
| 代码落点 | `internal/service/activities.go` 或独立 `recommend_service.go` |

---

### 4.5 组织批量停启（运营增强）

| 项目 | 内容 |
| --- | --- |
| 优先级 | P1 |
| 实现状态 | ❌ 未实现 |
| 新增 API | `POST /api/organizations/batch-disable` `POST /api/organizations/batch-enable` |
| 请求参数 | `ids[]` `reason` |
| 响应字段 | `successCount` `failedIds[]` |
| 权限 | 组织管理员/平台管理员（按权限模型收敛） |
| 依赖表 | `organizations` `sys_accounts` |
| 迁移建议 | 无 |
| 代码落点 | `internal/api/organization.proto` `internal/service/organization.go` `internal/repository/organization.go` `internal/repository/user.go` |

---

## 5. 数据迁移打包建议

按版本拆分，避免一次迁移过大：
1. `ddl_v1.2.0.sql`：仪表盘统计相关（`records`、`level_rules`、`volunteers` 补字段）
2. `ddl_v1.2.1.sql`：审核中心索引增强与志愿者资料审核方案（复用 `audit_records` 的索引/字段补强）
3. `ddl_v1.2.2.sql`：通知表
4. `ddl_v1.2.3.sql`：导入导出任务表
5. `ddl_v1.2.4.sql`：证书表

---

## 6. 联调与验收最小集

### 6.1 志愿者端
1. 活动列表同一活动在“已报名/未报名”账号下返回不同 `isRegistered`
2. 我的活动可看到待审核与驳回原因
3. 志愿者提交资料变更后，审核前主档案不变

### 6.2 管理员端
1. 审核中心可按目标类型筛选，并可完成审批
2. 单条审批报名记录后，活动人数与报名状态一致
3. 停用组织后账号权限立即受限，启用后恢复
4. 定时归档任务不纳入当前版本验收
