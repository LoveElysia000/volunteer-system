# 前端联调接口文档

> 适用对象：前端开发、联调测试
>
> 数据来源：`internal/api/*.proto`、`internal/router/*.go`、`internal/handler/*.go`
>
> 说明：
> 1. 本文件用于前端联调，重点关注“接口做什么、怎么传、会返回什么”
> 2. 文中“给谁用”偏后端真实权限口径
> 3. 如果是直接转发给前端同学快速浏览，优先看 `docs/frontend-api-one-page.md`

## 1. 通用约定

建议阅读顺序：

1. 先看本章的通用规则、状态枚举、错误说明、必填字段速查
2. 再看对应模块的接口清单
3. 最后直接用文末“核心联调示例”发请求验证

### 1.1 基础信息

- 接口前缀：`/api`
- 文档原始定义：`docs/openapi.yaml`
- 除 `登录 / 注册 / 刷新令牌 / 登出` 外，其余接口默认都需要登录态

### 1.2 认证方式

请求头统一携带：

```http
Authorization: Bearer <accessToken>
```

### 1.3 响应格式

绝大多数 JSON 接口统一返回：

```json
{
  "code": 200,
  "msg": "OK",
  "data": {}
}
```

联调时注意：

- `code = 200` 表示成功
- 非 `200` 表示业务失败或鉴权失败
- 前端主要关注 `data`
- 导出接口是文件流下载，不走上面的 JSON 包装

### 1.4 时间字段

- 大多数时间字段是字符串，常见格式为 `YYYY-MM-DD HH:mm:ss`
- 登录/刷新接口里的 `expiresAt` 是 Unix 时间戳

### 1.5 身份说明

- `游客`：未登录用户
- `已登录用户`：只要登录即可调用
- `当前志愿者本人`：仅当前登录志愿者可操作自己的数据
- `有组织管理权限的账号`：拥有组织维度或全局 `organization.manage` 权限
- `有成员管理权限的账号`：拥有组织维度或全局 `membership.manage` 权限
- `有审核权限的账号`：拥有组织维度或全局 `audit.review` 权限
- `有统计查看权限的账号`：拥有组织维度或全局 `analytics.org_read` 权限
- `有导出权限的账号`：拥有组织维度或全局 `export.manage` 权限
- `有 RBAC 全局管理权限的账号`：拥有全局 `rbac.manage` 权限

### 1.6 常用枚举

前端联调时最常用的枚举如下：

| 字段 | 枚举值 |
| --- | --- |
| `organization.status` | `0=停用`，`1=正常` |
| `volunteer.status` | `1=活跃`，`2=非活跃`，`3=其他` |
| `volunteer.auditStatus` | `0=未认证`，`1=审核中`，`2=已通过`，`3=已驳回` |
| `member.status` | `1=待审核`，`2=正式成员`，`3=已拒绝`，`4=已退出` |
| `member.role` | `1=普通成员`，`2=管理员`，`3=负责人` |
| `activity.status` | `1=报名中`，`2=已结束`，`3=已取消` |
| `signup.status` | `1=待审核`，`2=报名成功`，`3=报名驳回`，`4=已取消` |
| `checkInStatus` | `0=未签到`，`1=已签到` |
| `checkOutStatus` | `0=未签退`，`1=已签退` |
| `workHourStatus` | `0=未结算`，`1=已发放`，`2=已作废` |
| `workHour.operationType` | `1=发放`，`2=作废`，`3=重发/重算` |
| `notification.readStatus` | `0=未读`，`1=已读` |
| `audit.targetType` | `1=志愿者实名审核`，`2=组织审核`，`3=加入组织审核`，`4=活动报名审核` |
| `audit.status` | `1=待审核`，`2=审核通过`，`3=审核拒绝` |
| `audit.auditResult` | `1=通过`，`2=驳回` |
| `attendance codeType` | `1=签到码`，`2=签退码` |

使用建议：

- 前端建议把这些枚举统一收口到一个 `constants` / `enum map` 文件里
- 文案展示以这份文档为准，后续如果后端扩枚举，需要同步更新前端映射

### 1.7 常见错误

接口虽然统一返回 HTTP 200，但业务失败时最外层 `code` 不等于 `200`。联调时最常见的失败类型：

| 场景 | 常见提示 |
| --- | --- |
| 未登录 / token 失效 | `未提供认证令牌`、`认证令牌格式错误`、`认证令牌已过期，请刷新` |
| 权限不足 | `无权操作该组织`、`无权执行该操作`、`无权查看工时流水`、`无权限执行审核` |
| 参数错误 | `请求不能为空`、`组织ID不能为空`、`活动ID不能为空`、`状态值不合法` |
| 时间格式错误 | `开始时间格式错误`、`结束时间格式错误`、`签到时间格式错误`、`签退时间格式错误` |
| 业务状态不允许 | `活动不存在`、`组织不存在`、`成员关系不存在`、`已取消活动不允许签到`、`结束时间不能早于开始时间` |

联调建议：

- 所有请求统一拦截最外层 `code`
- 如果 `code !== 200`，优先展示 `msg`
- 对鉴权失败建议统一跳登录 / 触发刷新 token 逻辑

### 1.8 字段规则

为了减少来回确认，字段口径统一如下：

- 表格或说明里明确写了 `必填` 的字段，前端联调时必须传
- 没写 `必填` 的字段，默认按 `可选` 处理
- 路径参数如 `:id`、`:roleId`、`:bindingId` 一律视为必填
- 查询列表接口如果带 `page`、`pageSize`，联调时建议始终传
- 时间字段如果文档里给了格式示例，联调时按示例格式传
- 导入接口不走 JSON，必须使用 `multipart/form-data`
- 导出接口返回文件流，前端不要按普通 JSON 解析

### 1.9 高频接口必填速查

前端最常先联调下面这些接口，这里给一份“最小必填字段”速查表：

| 接口 | 最小必填字段 |
| --- | --- |
| `POST /api/login` | `loginType:string, identifier:string, password:string, identity:string` |
| `POST /api/refresh` | `refreshToken` |
| `POST /api/volunteers/list` | `page:int32, pageSize:int32` |
| `POST /api/organizations/list` | `page:int32, pageSize:int32` |
| `POST /api/activities` | `page:int32, pageSize:int32` |
| `GET /api/activities/:id` | 路径参数 `id` |
| `POST /api/activities/signup` | `activityId:int64` |
| `POST /api/activities/cancel` | `activityId:int64` |
| `POST /api/activities/checkin` | `activityId:int64, checkInCode:string` |
| `POST /api/activities/checkout` | `activityId:int64, checkOutCode:string` |
| `POST /api/activities/create` | `orgId:int64, title:string, startTime:string, endTime:string, location:string, duration:double, maxPeople:int32` |
| `PUT /api/activities/:id` | 路径参数 `id` |
| `POST /api/audits/approval` | `id:int64` |
| `POST /api/audits/rejection` | `id:int64` |
| `POST /api/audits/batch-decision` | `ids:int64[], action:int32` |
| `POST /api/memberships/join` | `volunteerId:int64, organizationId:int64` |
| `POST /api/memberships/leave` | `membershipId:int64` |
| `POST /api/memberships/status/update` | `membershipId:int64, status:int32` |
| `POST /api/work-hours/void` | `signupId:int64, reason:string, idempotencyKey:string` |
| `POST /api/work-hours/recalculate` | `signupId:int64, reason:string, idempotencyKey:string` |
| `POST /api/notifications/read` | `ids:int64[]` |
| `POST /api/admin/import/volunteers` | `form-data(file)` |
| `POST /api/admin/import/activities` | `form-data(file)` |
| `POST /api/admin/export/ops-report` | `periodType:string, orgId:int64` |
| `POST /api/assistant/sessions` | `scene` |
| `POST /api/assistant/chat` | `session_id:int64, message:string` |
| `POST /api/assistant/actions/activity-draft` | `topic` |

## 2. 接口总览

这一节用于先判断某个页面应该看哪一组接口。

| 模块 | 主要给谁用 |
| --- | --- |
| 注册登录 | 游客 |
| 志愿者信息 | 当前志愿者本人、有组织管理权限的账号 |
| 组织管理 | 已登录用户、有组织管理权限的账号 |
| 成员关系 | 当前志愿者本人、有成员管理权限的账号 |
| 活动管理 | 已登录用户、当前志愿者本人、有组织管理权限的账号 |
| 审核中心 | 有审核权限的账号 |
| 工时流水 | 当前志愿者本人、有组织管理权限的账号 |
| 通知中心 | 已登录用户 |
| 看板统计 | 有统计查看权限的账号 |
| RBAC 权限 | 有 RBAC 全局管理权限的账号 |
| 导入导出 | 有组织管理权限/导出权限的账号 |
| AI 助手 | 已登录用户 |

---

## 3. 注册登录

### 3.1 志愿者注册

- 接口：`POST /api/volunteer/register`
- 给谁用：`游客`
- 请求体字段类型：
  `name:string, phone:string, email:string, password:string, age:int32, gender:string, userName:string`
- 请求体：

```json
{
  "name": "张三",
  "phone": "13800000000",
  "email": "a@test.com",
  "password": "123456",
  "age": 20,
  "gender": "男",
  "userName": "zhangsan"
}
```

- 返回 `data`：`{}`

### 3.2 组织管理员注册

- 接口：`POST /api/organization/register`
- 给谁用：`游客`
- 请求体字段类型：
  `name:string, phone:string, email:string, password:string, organizationName:string, code:string, userName:string`
- 请求体：

```json
{
  "name": "李四",
  "phone": "13800000001",
  "email": "org@test.com",
  "password": "123456",
  "organizationName": "环保协会",
  "code": "9133XXXXXXXXXXXXXX",
  "userName": "green-org"
}
```

- 返回 `data`：`{}`

### 3.3 登录

- 接口：`POST /api/login`
- 给谁用：`游客`
- 请求体字段类型：
  `loginType:string, identifier:string, password:string, identity:string`
- 请求体：

```json
{
  "loginType": "email",
  "identifier": "a@test.com",
  "password": "123456",
  "identity": "volunteer"
}
```

- 返回 `data`：

```json
{
  "success": true,
  "message": "登录成功",
  "accessToken": "xxx",
  "refreshToken": "xxx",
  "expiresAt": 1710000000,
  "userInfo": {
    "userId": "1",
    "userName": "zhangsan",
    "email": "a@test.com",
    "phone": "13800000000",
    "displayName": "zhangsan",
    "avatarUrl": "",
    "identity": "volunteer",
    "createdAt": 1710000000,
    "updatedAt": 1710000000
  }
}
```

- 说明：
  `loginType` 当前只支持 `email` 和 `phone`
  `phone` 为解密后的手机号；若底层解密失败则返回空字符串
  `displayName` 当前默认取用户名
  `userInfo` 字段类型：
  `userId:string, userName:string, email:string, phone:string, displayName:string, avatarUrl:string, identity:string, createdAt:int64, updatedAt:int64`

### 3.4 登出

- 接口：`POST /api/logout`
- 给谁用：`游客/已登录用户`
- 请求体字段类型：
  `token:string`
- 请求体：

```json
{
  "token": "当前 accessToken"
}
```

- 返回 `data`：

```json
{
  "success": true,
  "message": "登出成功"
}
```

- 返回字段类型：
  `success:bool, message:string`

### 3.5 刷新令牌

- 接口：`POST /api/refresh`
- 给谁用：`游客/已登录用户`
- 请求体字段类型：
  `refreshToken:string`
- 请求体：

```json
{
  "refreshToken": "xxx"
}
```

- 返回 `data`：

```json
{
  "success": true,
  "message": "刷新成功",
  "token": "newAccessToken",
  "refreshToken": "newRefreshToken",
  "expiresAt": 1710000000,
  "userInfo": {}
}
```

- 返回字段类型：
  `success:bool, message:string, token:string, refreshToken:string, expiresAt:int64, userInfo:UserInfo`

---

## 4. 志愿者信息

### 4.1 接口清单

| 接口 | 给谁用 | 请求结构 | 返回 data 结构 |
| --- | --- | --- | --- |
| `POST /api/volunteers/list` | 有组织管理权限的账号 | `{ keyword:string, page:int32, pageSize:int32 }` | `{ total:int32, list: VolunteerListItem[] }` |
| `GET /api/volunteers/detail/:id` | 当前志愿者本人，或有组织管理权限且能覆盖该志愿者的账号 | 路径参数 `id:int64` | `{ volunteer: VolunteerInfo }` |
| `GET /api/volunteers/my/profile/:id` | 当前志愿者本人 | 路径参数 `id:int64` | `{ volunteer: VolunteerInfo }` |
| `GET /api/volunteers/home/summary` | 当前志愿者本人 | 无 | `{ nickname:string, level:int32, stats: VolunteerHomeSummaryStats, monthlyGrowth:double, needHoursToNextLevel:double }` |
| `PUT /api/volunteers/:id` | 当前志愿者本人，或有组织管理权限且能覆盖该志愿者的账号 | 路径参数 `id:int64` + `{ realName:string, gender:int32, birthday:string, avatarUrl:string, introduction:string }` | `{}` |
| `POST /api/volunteers/real-name/submit` | 当前志愿者本人 | `{ realName:string, idCard:string }` | `{ auditId:int64, status:int32 }` |

### 4.2 关键返回结构

`VolunteerListItem`

字段类型：
`id:int64, accountId:int64, realName:string, gender:int32, avatarUrl:string, totalHours:double, serviceCount:int32, creditScore:int32, auditStatus:int32, createdAt:string, updatedAt:string, status:int32`

```json
{
  "id": 1,
  "accountId": 1,
  "realName": "张三",
  "gender": 1,
  "avatarUrl": "",
  "totalHours": 12.5,
  "serviceCount": 6,
  "creditScore": 100,
  "auditStatus": 2,
  "createdAt": "2026-03-20 10:00:00",
  "updatedAt": "2026-03-20 10:00:00",
  "status": 1
}
```

`VolunteerInfo`

字段类型：
`id:int64, accountId:int64, realName:string, gender:int32, birthday:string, idCard:string, avatarUrl:string, introduction:string, totalHours:double, serviceCount:int32, creditScore:int32, auditStatus:int32, createdAt:string, updatedAt:string, status:int32`

```json
{
  "id": 1,
  "accountId": 1,
  "realName": "张三",
  "gender": 1,
  "birthday": "2000-01-01",
  "idCard": "310xxxxxxxxxxxxxxx",
  "avatarUrl": "",
  "introduction": "个人简介",
  "totalHours": 12.5,
  "serviceCount": 6,
  "creditScore": 100,
  "auditStatus": 2,
  "createdAt": "2026-03-20 10:00:00",
  "updatedAt": "2026-03-20 10:00:00",
  "status": 1
}
```

`VolunteerHomeSummaryStats` 字段类型：
`points:int32, hours:double, activityCount:int32`

---

## 5. 组织管理

### 5.1 接口清单

| 接口 | 给谁用 | 请求结构 | 返回 data 结构 |
| --- | --- | --- | --- |
| `POST /api/organizations/list` | 有组织管理权限的账号 | `{ keyword:string, status:int32[], organizationType:string, region:string, page:int32, pageSize:int32 }` | `{ total:int32, list: OrganizationListItem[] }` |
| `GET /api/organizations/:id` | 有组织管理权限且能管理该组织的账号 | 路径参数 `id:int64` | `{ organization: OrganizationInfo }` |
| `POST /api/organizations/create` | 已登录用户 | `{ name:string, organizationCode:string, contactPerson:string, contactPhone:string, email:string, address:string, organizationType:string, region:string, description:string, websiteUrl:string, logoUrl:string }` | `{ id:int64, message:string }` |
| `PUT /api/organizations/:id` | 有组织管理权限且能管理该组织的账号 | 路径参数 `id:int64` + 可更新字段 `{ name:string, organizationCode:string, contactPerson:string, contactPhone:string, email:string, address:string, organizationType:string, region:string, description:string, websiteUrl:string, logoUrl:string }` | `{ message:string }` |
| `DELETE /api/organizations/:id` | 有组织管理权限且能管理该组织的账号 | 路径参数 `id:int64` | `{ message:string }` |
| `POST /api/organizations/:id/disable` | 有组织管理权限且能管理该组织的账号 | 路径参数 `id:int64` + `{ reason:string }` | `{ message:string }` |
| `POST /api/organizations/:id/enable` | 有组织管理权限且能管理该组织的账号 | 路径参数 `id:int64` + `{ reason:string }` | `{ message:string }` |
| `POST /api/organizations/search` | 有组织管理权限的账号 | `{ keyword:string, status:int32[], organizationType:string, region:string, startDate:string, endDate:string, page:int32, pageSize:int32 }` | `{ total:int32, list: OrganizationListItem[] }` |
| `POST /api/organizations/bulk-delete` | 有组织管理权限且能管理目标组织的账号 | `{ ids:int64[] }` | `{ successCount:int32, failedCount:int32, message:string }` |
| `POST /api/organizations/batch-disable` | 有组织管理权限且能管理目标组织的账号 | `{ ids:int64[], reason:string }` | `{ successCount:int32, failedIds:int64[], message:string }` |
| `POST /api/organizations/batch-enable` | 有组织管理权限且能管理目标组织的账号 | `{ ids:int64[], reason:string }` | `{ successCount:int32, failedIds:int64[], message:string }` |

### 5.2 关键返回结构

`OrganizationListItem`

字段类型：
`id:int64, name:string, organizationCode:string, contactPerson:string, contactPhone:string, email:string, address:string, status:int32, organizationType:string, region:string, createdAt:string`

```json
{
  "id": 1,
  "name": "环保协会",
  "organizationCode": "ORG001",
  "contactPerson": "李四",
  "contactPhone": "13800000001",
  "email": "",
  "address": "上海",
  "status": 1,
  "organizationType": "",
  "region": "",
  "createdAt": "2026-03-20 10:00:00"
}
```

`OrganizationInfo`

字段类型：
`id:int64, accountId:int64, name:string, organizationCode:string, contactPerson:string, contactPhone:string, email:string, address:string, status:int32, organizationType:string, region:string, description:string, websiteUrl:string, logoUrl:string, createdAt:string, updatedAt:string`

```json
{
  "id": 1,
  "accountId": 10,
  "name": "环保协会",
  "organizationCode": "ORG001",
  "contactPerson": "李四",
  "contactPhone": "13800000001",
  "email": "",
  "address": "上海",
  "status": 1,
  "organizationType": "",
  "region": "",
  "description": "组织简介",
  "websiteUrl": "",
  "logoUrl": "",
  "createdAt": "2026-03-20 10:00:00",
  "updatedAt": "2026-03-20 10:00:00"
}
```

- 说明：
  `POST /api/organizations/create` 当前任何已登录账号都可调用
  当前组织列表/详情返回里的 `email`、`organizationType`、`region`、`websiteUrl` 默认为空字符串
  `create/update` 接口虽然接收 `email`、`organizationType`、`region`、`websiteUrl`，但当前后端未持久化这些字段，前端联调时不要依赖回显

---

## 6. 成员关系

| 接口 | 给谁用 | 请求结构 | 返回 data 结构 |
| --- | --- | --- | --- |
| `POST /api/memberships/join` | 当前志愿者本人 | `{ volunteerId:int64, organizationId:int64 }` | `{ membershipId:int64, status:int32, message:string }` |
| `POST /api/memberships/leave` | 当前志愿者本人 | `{ membershipId:int64, reason:string }` | `{ message:string }` |
| `GET /api/organizations/:organizationId/members` | 有成员管理权限且能管理该组织的账号 | 路径参数 `organizationId:int64` + 查询参数 `{ status:int32, role:int32, keyword:string, page:int32, pageSize:int32 }` | `{ total:int32, list: MemberInfo[] }` |
| `GET /api/volunteers/:volunteerId/organizations` | 当前志愿者本人 | 路径参数 `volunteerId:int64` + 查询参数 `{ status:int32, page:int32, pageSize:int32 }` | `{ total:int32, list: OrganizationMemberInfo[] }` |
| `POST /api/memberships/status/update` | 有成员管理权限且能管理该组织的账号 | `{ membershipId:int64, status:int32, reviewComment:string }` | `{ message:string }` |
| `GET /api/memberships/stats` | 有成员管理权限的账号 | 查询参数 `{ organizationId:int64 }` | `{ pendingCount:int64, activeCount:int64, inactiveCount:int64, suspendedCount:int64, totalCount:int64 }` |

`MemberInfo` 主要字段：
`membershipId:int64, volunteerId:int64, volunteerName:string, volunteerCode:string, organizationId:int64, organizationName:string, status:int32, role:int32, position:string, motivation:string, expectedHours:string, joinDate:string, reviewDate:string, reviewComment:string, leaveDate:string, leaveReason:string, createdAt:string, updatedAt:string`

`OrganizationMemberInfo` 主要字段：
`membershipId:int64, organizationId:int64, organizationName:string, organizationCode:string, status:int32, role:int32, position:string, joinDate:string, reviewDate:string, reviewComment:string, createdAt:string, updatedAt:string`

- 说明：
  `POST /api/memberships/join` 里的 `volunteerId` 需要传当前登录志愿者本人 ID；若传其他 ID 会被拒绝

---

## 7. 活动管理

### 7.1 志愿者侧

| 接口 | 给谁用 | 请求结构 | 返回 data 结构 |
| --- | --- | --- | --- |
| `POST /api/activities` | 已登录用户 | `{ page:int32, pageSize:int32, status:int32, keyword:string, startFrom:string, startTo:string, sortBy:string, sortOrder:string }` | `{ total:int32, list: ActivityItem[] }` |
| `GET /api/activities/:id` | 已登录用户 | 路径参数 `id:int64` | `{ activity: ActivityInfo }` |
| `POST /api/activities/signup` | 当前志愿者本人 | `{ activityId:int64 }` | `{ success:bool }` |
| `POST /api/activities/cancel` | 当前志愿者本人 | `{ activityId:int64 }` | `{ success:bool }` |
| `POST /api/activities/my` | 当前志愿者本人 | `{ page:int32, pageSize:int32, status:int32 }` | `{ total:int32, list: MyActivityItem[] }` |
| `POST /api/activities/checkin` | 当前志愿者本人 | `{ activityId:int64, checkInCode:string }` | `{ success:bool, checkInTime:string }` |
| `POST /api/activities/checkout` | 当前志愿者本人 | `{ activityId:int64, checkOutCode:string }` | `{ success:bool, checkOutTime:string, grantedHours:double }` |

### 7.2 组织侧

| 接口 | 给谁用 | 请求结构 | 返回 data 结构 |
| --- | --- | --- | --- |
| `POST /api/activities/create` | 有组织管理权限且能管理该组织的账号 | `{ orgId:int64, title:string, description:string, coverUrl:string, startTime:string, endTime:string, location:string, address:string, duration:double, maxPeople:int32 }` | `{ id:int64, message:string }` |
| `PUT /api/activities/:id` | 有组织管理权限且能管理该活动所属组织的账号 | 路径参数 `id:int64` + 可更新字段 `{ title:string, description:string, coverUrl:string, startTime:string, endTime:string, location:string, address:string, duration:double, maxPeople:int32 }` | `{ message:string }` |
| `DELETE /api/activities/:id` | 有组织管理权限且能管理该活动所属组织的账号 | 路径参数 `id:int64` | `{ message:string }` |
| `POST /api/activities/cancel/:id` | 有组织管理权限且能管理该活动所属组织的账号 | 路径参数 `id:int64` + `{ reason:string }` | `{ message:string }` |
| `POST /api/activities/finish/:id` | 有组织管理权限且能管理该活动所属组织的账号 | 路径参数 `id:int64` | `{ message:string }` |
| `POST /api/activities/attendance-codes/generate/:id` | 有组织管理权限且能管理该活动所属组织的账号 | 路径参数 `id:int64` + `{ checkInValidMinutes:int32, checkOutValidMinutes:int32 }` | `{ success:bool, checkInCode:string, checkOutCode:string, attendanceCodeVersion:int64, attendanceCodeUpdatedAt:string, checkInExpireAt:string, checkOutExpireAt:string }` |
| `POST /api/activities/attendance-codes/reset/:id` | 有组织管理权限且能管理该活动所属组织的账号 | 路径参数 `id:int64` + `{ codeType:int32, validMinutes:int32 }` | `{ success:bool, code:string, codeType:int32, attendanceCodeVersion:int64, attendanceCodeUpdatedAt:string, expireAt:string }` |
| `GET /api/activities/attendance-codes/:id` | 有组织管理权限且能管理该活动所属组织的账号 | 路径参数 `id:int64` | `{ success:bool, checkInCode:string, checkOutCode:string, attendanceCodeVersion:int64, attendanceCodeUpdatedAt:string, checkInExpireAt:string, checkOutExpireAt:string }` |
| `POST /api/activities/supplement-attendance` | 有组织管理权限且能管理该活动所属组织的账号 | `{ activityId:int64, volunteerId:int64, checkInTime:string, checkOutTime:string, reason:string }` | `{ success:bool, checkInTime:string, checkOutTime:string, grantedHours:double }` |

### 7.3 关键返回结构

`ActivityItem` 主要字段：
`id:int64, title:string, description:string, coverUrl:string, startTime:string, endTime:string, location:string, duration:double, maxPeople:int32, currentPeople:int32, status:int32, isRegistered:bool, isFull:bool`

`ActivityInfo` 主要字段：
`id:int64, orgId:int64, orgName:string, title:string, description:string, coverUrl:string, startTime:string, endTime:string, location:string, address:string, duration:double, maxPeople:int32, currentPeople:int32, status:int32, isRegistered:bool, createdAt:string, checkInStatus:int32, checkInTime:string, checkOutStatus:int32, checkOutTime:string, workHourStatus:int32, grantedHours:double`

`MyActivityItem` 主要字段：
`id:int64, orgId:int64, orgName:string, title:string, description:string, coverUrl:string, startTime:string, endTime:string, location:string, duration:double, maxPeople:int32, currentPeople:int32, status:int32, signupTime:string, checkInStatus:int32, checkInTime:string, checkOutStatus:int32, checkOutTime:string, workHourStatus:int32, grantedHours:double, signupStatus:int32, auditReason:string`

---

## 8. 审核中心

| 接口 | 给谁用 | 请求结构 | 返回 data 结构 |
| --- | --- | --- | --- |
| `POST /api/audits/pending` | 有审核权限的账号 | `{ targetTypes:int32[], status:int32[], keyword:string, page:int32, pageSize:int32, createdFrom:string, createdTo:string, slaHours:int32 }` | `{ total:int32, list: PendingAuditItem[] }` |
| `POST /api/audits/approval` | 有审核权限且能审核该记录的账号 | `{ id:int64, reason:string }` | `{}` |
| `POST /api/audits/rejection` | 有审核权限且能审核该记录的账号 | `{ id:int64, reason:string }` | `{}` |
| `POST /api/audits/batch-decision` | 有审核权限且能审核目标记录的账号 | `{ ids:int64[], action:int32, reason:string }` | `{ successCount:int32, failedIds:int64[] }` |
| `GET /api/audits/records/:id` | 有审核权限且能查看该记录的账号 | 路径参数 `id:int64` | `{ record: AuditRecordDetail }` |

`PendingAuditItem` 主要字段：
`id:int64, targetType:int32, targetId:int64, title:string, subTitle:string, creatorId:int64, createdAt:string, isOverdue:bool`

`AuditRecordDetail` 主要字段：
`id:int64, targetType:int32, targetId:int64, auditorId:int64, status:int32, oldContent:string, newContent:string, auditResult:int32, rejectReason:string, auditTime:string, createdAt:string`

---

## 9. 工时流水

| 接口 | 给谁用 | 请求结构 | 返回 data 结构 |
| --- | --- | --- | --- |
| `POST /api/work-hours/list` | 当前志愿者本人，或有组织管理权限的账号 | `{ page:int32, pageSize:int32, activityId:int64, signupId:int64, operationType:int32 }` | `{ total:int32, list: WorkHourLogItem[] }` |
| `POST /api/work-hours/void` | 有组织管理权限且能管理该报名所属活动的账号 | `{ signupId:int64, reason:string, idempotencyKey:string }` | `{ success:bool, workHourLogId:int64 }` |
| `POST /api/work-hours/recalculate` | 有组织管理权限且能管理该报名所属活动的账号 | `{ signupId:int64, hours:double, reason:string, idempotencyKey:string }` | `{ success:bool, workHourLogId:int64, grantedHours:double }` |

`WorkHourLogItem` 主要字段：
`id:int64, volunteerId:int64, activityId:int64, signupId:int64, operationType:int32, hoursDelta:double, serviceCountDelta:int64, beforeTotalHours:double, afterTotalHours:double, beforeServiceCount:int64, afterServiceCount:int64, workHourVersion:int64, refLogId:int64, reason:string, operatorId:int64, idempotencyKey:string, createdAt:string`

---

## 10. 通知中心

| 接口 | 给谁用 | 请求结构 | 返回 data 结构 |
| --- | --- | --- | --- |
| `GET /api/notifications` | 已登录用户 | 查询参数 `{ page:int32, pageSize:int32, unreadOnly:bool }` | `{ total:int32, list: NotificationItem[] }` |
| `POST /api/notifications/read` | 已登录用户 | `{ ids:int64[] }` | `{ updated:int32 }` |

`NotificationItem` 主要字段：
`inboxId:int64, notificationId:int64, eventType:string, bizType:string, bizId:int64, title:string, content:string, readStatus:int32, readAt:string, createdAt:string`

---

## 11. 看板统计

| 接口 | 给谁用 | 请求结构 | 返回 data 结构 |
| --- | --- | --- | --- |
| `GET /api/analytics/org/funnel` | 有统计查看权限且能访问该组织的账号 | 查询参数 `{ orgId:int64, start:string, end:string }` | `{ registrationCount:int64, membershipCount:int64, signupCount:int64, attendanceCount:int64, workhourCount:int64, registrationToMembershipRate:double, membershipToSignupRate:double, signupToAttendanceRate:double, attendanceToWorkhourRate:double, start:string, end:string }` |
| `GET /api/analytics/org/dashboard` | 有统计查看权限且能访问该组织的账号 | 查询参数 `{ orgId:int64, start:string, end:string }` | `{ signupCount:int64, approvedSignupCount:int64, attendanceCount:int64, attendanceRate:double, grantedWorkHours:double, start:string, end:string }` |

---

## 12. RBAC 权限

| 接口 | 给谁用 | 请求结构 | 返回 data 结构 |
| --- | --- | --- | --- |
| `GET /api/authz/roles` | 有 RBAC 全局管理权限的账号 | 查询参数 `{ keyword:string, includeDisabled:bool, page:int32, pageSize:int32 }` | `{ total:int32, list: RoleInfo[] }` |
| `POST /api/authz/roles` | 有 RBAC 全局管理权限的账号 | `{ roleCode:string, roleName:string, description:string }` | `{ id:int64 }` |
| `PUT /api/authz/roles/:id` | 有 RBAC 全局管理权限的账号 | 路径参数 `id:int64` + `{ roleName:string, description:string }` | `{ message:string }` |
| `POST /api/authz/roles/:id/status` | 有 RBAC 全局管理权限的账号 | 路径参数 `id:int64` + `{ status:int32 }` | `{ message:string }` |
| `GET /api/authz/permissions` | 有 RBAC 全局管理权限的账号 | 查询参数 `{ keyword:string, onlyEnabled:bool }` | `{ list: PermissionInfo[] }` |
| `GET /api/authz/roles/:roleId/permissions` | 有 RBAC 全局管理权限的账号 | 路径参数 `roleId:int64` | `{ roleId:int64, permissions: RolePermissionItem[] }` |
| `POST /api/authz/roles/:roleId/permissions/set` | 有 RBAC 全局管理权限的账号 | 路径参数 `roleId:int64` + `{ permissionIds:int64[] }` | `{ message:string }` |
| `GET /api/authz/grants` | 有 RBAC 全局管理权限的账号 | 查询参数 `{ accountId:int64, scopeType:string, scopeId:int64, onlyActive:bool, page:int32, pageSize:int32 }` | `{ total:int32, list: AccountRoleBindingInfo[] }` |
| `POST /api/authz/grants` | 有 RBAC 全局管理权限的账号 | `{ accountId:int64, roleId:int64, scopeType:string, scopeId:int64, expiresAt:string, remark:string }` | `{ message:string }` |
| `POST /api/authz/grants/:bindingId/revoke` | 有 RBAC 全局管理权限的账号 | 路径参数 `bindingId:int64` + `{ remark:string }` | `{ message:string }` |
| `GET /api/authz/me` | 已登录用户 | 无 | `{ accountId:int64, roles: AccountRoleBindingInfo[], permissions: MyPermissionScope[] }` |

`RoleInfo` 字段类型：
`id:int64, roleCode:string, roleName:string, description:string, status:int32, createdAt:string, updatedAt:string`

`PermissionInfo` 字段类型：
`id:int64, resource:string, action:string, description:string, status:int32`

`RolePermissionItem` 字段类型：
`permissionId:int64, resource:string, action:string, description:string`

`AccountRoleBindingInfo` 字段类型：
`bindingId:int64, accountId:int64, roleId:int64, roleCode:string, roleName:string, scopeType:string, scopeId:int64, status:int32, grantedBy:int64, expiresAt:string, createdAt:string, updatedAt:string`

`MyPermissionScope` 字段类型：
`resource:string, action:string, scopeType:string, scopeId:int64`

---

## 13. 导入导出

### 13.1 导出接口

这 3 个接口返回的是文件流，不是 JSON：

| 接口 | 给谁用 | 请求结构 | 返回 |
| --- | --- | --- | --- |
| `POST /api/admin/export/volunteers` | 有导出权限的账号；若存在多个组织作用域则此接口可能无法直接确定组织 | `{ idList:int64[], keyword:string, auditStatus:int32, status:int32 }` | Excel 文件流 |
| `POST /api/admin/export/activities` | 有导出权限的账号；若存在多个组织作用域则此接口可能无法直接确定组织 | `{ idList:int64[], keyword:string, status:int32, startFrom:string, startTo:string }` | Excel 文件流 |
| `POST /api/admin/export/ops-report` | 有导出权限且能访问指定组织的账号 | `{ periodType:string, orgId:int64, start:string, end:string }` | Excel 文件流 |

前端处理建议：

- `responseType` 设为 `blob`
- 从响应头 `Content-Disposition` 取文件名

### 13.2 导入接口

这 2 个接口使用 `multipart/form-data` 上传文件，字段名固定为 `file`：

| 接口 | 给谁用 | 请求格式 | 返回 data 结构 |
| --- | --- | --- | --- |
| `POST /api/admin/import/volunteers` | 有组织管理权限的账号 | `form-data(file:binary)` | `{ totalCount:int32, successCount:int32, failedCount:int32, errorFileName:string, errorFileContentType:string, errorFileContent:bytes(base64), failures: ImportFailureItem[] }` |
| `POST /api/admin/import/activities` | 有组织管理权限且能管理导入数据所属组织的账号 | `form-data(file:binary)` | `{ totalCount:int32, successCount:int32, failedCount:int32, errorFileName:string, errorFileContentType:string, errorFileContent:bytes(base64), failures: ImportFailureItem[] }` |

`ImportFailureItem` 结构：
`rowNumber:int32, reason:string, raw:string`

---

## 14. AI 助手

### 14.1 普通接口

| 接口 | 给谁用 | 请求结构 | 返回 data 结构 |
| --- | --- | --- | --- |
| `POST /api/assistant/sessions` | 已登录用户 | `{ scene:string, title:string }` | `{ session_id:int64 }` |
| `POST /api/assistant/chat` | 已登录用户（且仅能对自己的会话发消息） | `{ session_id:int64, message:string, stream:bool }` | `{ reply:string, tool_calls: AssistantToolCall[], usage: AssistantUsage }` |
| `GET /api/assistant/sessions/:id/messages` | 已登录用户（且仅能查看自己的会话） | 路径参数 `id:int64` | `{ list: AssistantMessageItem[] }` |
| `POST /api/assistant/actions/activity-draft` | 已登录用户；实际能否生成具体组织草案取决于该用户是否拥有对应组织成员/可访问组织范围 | `{ session_id:int64, topic:string, target_people:string, location:string }` | `{ session_id:int64, result: AssistantChatResponse }` |

`AssistantToolCall` 字段类型：
`tool_name:string, success:bool, error_code:string, error_msg:string, latency_ms:int32, input:string, output:string`

`AssistantUsage` 字段类型：
`model:string, token_in:int32, token_out:int32, latency_ms:int32`

`AssistantMessageItem` 字段类型：
`id:int64, session_id:int64, seq_no:int32, role:int32, content:string, model:string, finish_reason:int32, token_in:int32, token_out:int32, latency_ms:int32, request_id:string, created_at:string`

`AssistantChatResponse` 字段类型：
`reply:string, tool_calls:AssistantToolCall[], usage:AssistantUsage`

### 14.2 SSE 流式接口

- 接口：`POST /api/assistant/chat/stream`
- 给谁用：`登录用户通用`
- 请求体：

```json
{
  "session_id": 1,
  "message": "帮我生成一个环保活动方案",
  "stream": false
}
```

- 返回类型：`text/event-stream`
- 事件流示例：

```text
event: start
data: {"session_id":1}

event: delta
data: {"text":"第一段文本"}

event: tool
data: {"tool_name":"xxx","success":true}

event: usage
data: {"model":"xxx","token_in":10,"token_out":20,"latency_ms":800}

event: done
data: {"finish_reason":"stop"}
```

---

## 15. 前端联调建议

### 15.1 建议优先联调顺序

1. 注册/登录/刷新令牌
2. 当前用户信息、组织列表、活动列表
3. 报名/取消报名/签到/签退
4. 组织侧活动管理、成员管理、审核
5. 通知、统计、导出导入、AI 助手

### 15.2 前端需要特别注意

- 所有 JSON 接口都先判断最外层 `code`
- 导出接口要按文件流处理
- 导入接口要用 `multipart/form-data`
- 大多数列表接口带分页：`page`、`pageSize`
- 多数字段是可选的，联调时建议只传当前页面真实需要的字段
- 活动、审核、工时等模块里有较多状态码，前端最好统一做枚举映射

---

## 16. 文档边界说明

这份文档是给前端联调使用的“简版说明”，重点是：

- 这个接口给谁用
- 需要传什么
- 会返回什么
- 有没有鉴权/文件流/SSE 这类特殊处理

---

## 17. 核心联调示例

下面这些示例尽量选“最小可用请求”，前端可以直接照着调。

### 17.1 登录

请求：

```http
POST /api/login
Content-Type: application/json
```

```json
{
  "loginType": "phone",
  "identifier": "13800000000",
  "password": "123456",
  "identity": "volunteer"
}
```

成功响应：

```json
{
  "code": 200,
  "msg": "OK",
  "data": {
    "success": true,
    "message": "登录成功",
    "accessToken": "xxx",
    "refreshToken": "xxx",
    "expiresAt": 1710000000,
    "userInfo": {
      "userId": "1",
      "userName": "zhangsan",
      "identity": "volunteer"
    }
  }
}
```

### 17.2 活动列表

请求：

```http
POST /api/activities
Authorization: Bearer <accessToken>
Content-Type: application/json
```

```json
{
  "page": 1,
  "pageSize": 10,
  "status": 1,
  "keyword": "",
  "sortBy": "start_time",
  "sortOrder": "asc"
}
```

### 17.3 创建活动

请求：

```http
POST /api/activities/create
Authorization: Bearer <accessToken>
Content-Type: application/json
```

```json
{
  "orgId": 1,
  "title": "社区环保日",
  "description": "社区清洁和垃圾分类宣传",
  "coverUrl": "",
  "startTime": "2026-03-28 09:00:00",
  "endTime": "2026-03-28 12:00:00",
  "location": "社区广场",
  "address": "上海市XX路XX号",
  "duration": 3,
  "maxPeople": 30
}
```

成功响应：

```json
{
  "code": 200,
  "msg": "OK",
  "data": {
    "id": 1001,
    "message": "创建活动成功"
  }
}
```

### 17.4 活动报名审核通过

请求：

```http
POST /api/audits/approval
Authorization: Bearer <accessToken>
Content-Type: application/json
```

```json
{
  "id": 5001,
  "reason": "审核通过"
}
```

### 17.5 通知批量已读

请求：

```http
POST /api/notifications/read
Authorization: Bearer <accessToken>
Content-Type: application/json
```

```json
{
  "ids": [101, 102, 103]
}
```

成功响应：

```json
{
  "code": 200,
  "msg": "OK",
  "data": {
    "updated": 3
  }
}
```

### 17.6 导入志愿者

请求方式：

```http
POST /api/admin/import/volunteers
Authorization: Bearer <accessToken>
Content-Type: multipart/form-data
```

表单字段：

- `file`：Excel 文件

### 17.7 导出活动

请求：

```http
POST /api/admin/export/activities
Authorization: Bearer <accessToken>
Content-Type: application/json
```

```json
{
  "keyword": "",
  "status": 1,
  "startFrom": "2026-03-01 00:00:00",
  "startTo": "2026-03-31 23:59:59"
}
```

说明：

- 返回是文件流
- 前端要用 `blob` 方式接收
- 文件名从 `Content-Disposition` 读取

如果需要更完整的字段定义和类型细节，以以下文件为准：

- `internal/api/*.proto`
- `docs/openapi.yaml`
