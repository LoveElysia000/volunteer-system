# 前端联调接口清单（一页版）

> 适合直接发给前端同学快速联调
>
> 完整版见：`docs/frontend-api-integration.md`

## 1. 通用约定

- 基础前缀：`/api`
- 鉴权方式：`Authorization: Bearer <accessToken>`
- 除 `登录 / 注册 / 刷新令牌 / 登出` 外，其余接口默认都需要登录
- JSON 接口统一返回：

```json
{
  "code": 200,
  "msg": "OK",
  "data": {}
}
```

- 前端统一先判断最外层 `code`
- 时间字段大多为字符串，常见格式：`YYYY-MM-DD HH:mm:ss`
- `expiresAt` 为 Unix 时间戳
- 导出接口返回文件流，不走 JSON 包装
- 导入接口使用 `multipart/form-data`，文件字段名固定为 `file`
- AI 流式接口返回 `text/event-stream`

### 1.1 身份口径

- `志愿者`：志愿者端用户
- `组织管理者`：组织侧账号，通常可管理自己所属组织的数据
- `超管`：平台全局管理账号
- 如果某接口写的是 `组织管理者/超管`，前端可以理解为“组织侧后台使用，超管也可用”
- 如果某接口写的是 `已登录用户`，表示志愿者和组织管理者都能用

## 2. 注册登录

| 接口 | 给谁用 | 核心入参 | 核心返回 |
| --- | --- | --- | --- |
| `POST /api/volunteer/register` | 游客 | `name:string, phone:string, email:string, password:string, age:int32, gender:string, userName:string` | `data:{}` |
| `POST /api/organization/register` | 游客 | `name:string, phone:string, email:string, password:string, organizationName:string, code:string, userName:string` | `data:{}` |
| `POST /api/login` | 游客 | `loginType:string(email/phone), identifier:string, password:string, identity:string` | `success:bool, message:string, accessToken:string, refreshToken:string, expiresAt:int64, userInfo:UserInfo` |
| `POST /api/logout` | 游客/已登录用户 | `token:string` | `success:bool, message:string` |
| `POST /api/refresh` | 游客/已登录用户 | `refreshToken:string` | `success:bool, message:string, token:string, refreshToken:string, expiresAt:int64, userInfo:UserInfo` |

## 3. 志愿者相关

| 接口 | 给谁用 | 核心入参 | 核心返回 |
| --- | --- | --- | --- |
| `POST /api/volunteers/list` | 有组织管理权限的账号 | `keyword:string, page:int32, pageSize:int32` | `total:int32, list:VolunteerListItem[]` |
| `GET /api/volunteers/detail/:id` | 当前志愿者本人，或有组织管理权限且能覆盖该志愿者的账号 | 路径参数 `id` | `volunteer` |
| `GET /api/volunteers/my/profile/:id` | 志愿者 | 路径参数 `id` | `volunteer` |
| `GET /api/volunteers/home/summary` | 志愿者 | 无 | `nickname:string, level:int32, stats:VolunteerHomeSummaryStats, monthlyGrowth:double, needHoursToNextLevel:double` |
| `PUT /api/volunteers/:id` | 当前志愿者本人，或有组织管理权限且能覆盖该志愿者的账号 | `realName:string, gender:int32, birthday:string, avatarUrl:string, introduction:string` | `data:{}` |
| `POST /api/volunteers/real-name/submit` | 志愿者 | `realName:string, idCard:string` | `auditId:int64, status:int32` |

## 4. 组织相关

| 接口 | 给谁用 | 核心入参 | 核心返回 |
| --- | --- | --- | --- |
| `POST /api/organizations/list` | 有组织管理权限的账号 | `keyword:string, status:int32[], organizationType:string, region:string, page:int32, pageSize:int32` | `total:int32, list:OrganizationListItem[]` |
| `GET /api/organizations/:id` | 有组织管理权限且能管理该组织的账号 | 路径参数 `id` | `organization` |
| `POST /api/organizations/create` | 已登录用户 | `name:string, organizationCode:string, contactPerson:string, contactPhone:string, email:string, address:string, organizationType:string, region:string, description:string, websiteUrl:string, logoUrl:string` | `id:int64, message:string` |
| `PUT /api/organizations/:id` | 有组织管理权限且能管理该组织的账号 | 路径参数 `id` + 可更新字段 | `message` |
| `DELETE /api/organizations/:id` | 有组织管理权限且能管理该组织的账号 | 路径参数 `id` | `message` |
| `POST /api/organizations/:id/disable` | 有组织管理权限且能管理该组织的账号 | 路径参数 `id` + `reason` | `message` |
| `POST /api/organizations/:id/enable` | 有组织管理权限且能管理该组织的账号 | 路径参数 `id` + `reason` | `message` |
| `POST /api/organizations/search` | 有组织管理权限的账号 | `keyword:string, status:int32[], organizationType:string, region:string, startDate:string, endDate:string, page:int32, pageSize:int32` | `total:int32, list:OrganizationListItem[]` |
| `POST /api/organizations/bulk-delete` | 有组织管理权限且能管理目标组织的账号 | `ids:int64[]` | `successCount:int32, failedCount:int32, message:string` |
| `POST /api/organizations/batch-disable` | 有组织管理权限且能管理目标组织的账号 | `ids:int64[], reason:string` | `successCount:int32, failedIds:int64[], message:string` |
| `POST /api/organizations/batch-enable` | 有组织管理权限且能管理目标组织的账号 | `ids:int64[], reason:string` | `successCount:int32, failedIds:int64[], message:string` |

## 5. 成员关系

| 接口 | 给谁用 | 核心入参 | 核心返回 |
| --- | --- | --- | --- |
| `POST /api/memberships/join` | 志愿者 | `volunteerId:int64(当前本人), organizationId:int64` | `membershipId:int64, status:int32, message:string` |
| `POST /api/memberships/leave` | 志愿者 | `membershipId:int64, reason:string` | `message:string` |
| `GET /api/organizations/:organizationId/members` | 有成员管理权限且能管理该组织的账号 | 查询参数 `status:int32, role:int32, keyword:string, page:int32, pageSize:int32` | `total:int32, list:MemberInfo[]` |
| `GET /api/volunteers/:volunteerId/organizations` | 志愿者 | 查询参数 `status:int32, page:int32, pageSize:int32` | `total:int32, list:OrganizationMemberInfo[]` |
| `POST /api/memberships/status/update` | 有成员管理权限且能管理该组织的账号 | `membershipId:int64, status:int32, reviewComment:string` | `message:string` |
| `GET /api/memberships/stats` | 有成员管理权限的账号 | 查询参数 `organizationId:int64` | `pendingCount:int64, activeCount:int64, inactiveCount:int64, suspendedCount:int64, totalCount:int64` |

## 6. 活动相关

### 6.1 志愿者侧

| 接口 | 给谁用 | 核心入参 | 核心返回 |
| --- | --- | --- | --- |
| `POST /api/activities` | 已登录用户 | `page:int32, pageSize:int32, status:int32, keyword:string, startFrom:string, startTo:string, sortBy:string, sortOrder:string` | `total:int32, list:ActivityItem[]` |
| `GET /api/activities/:id` | 已登录用户 | 路径参数 `id` | `activity` |
| `POST /api/activities/signup` | 志愿者 | `activityId` | `success` |
| `POST /api/activities/cancel` | 志愿者 | `activityId` | `success` |
| `POST /api/activities/my` | 志愿者 | `page:int32, pageSize:int32, status:int32` | `total:int32, list:MyActivityItem[]` |
| `POST /api/activities/checkin` | 志愿者 | `activityId:int64, checkInCode:string` | `success:bool, checkInTime:string` |
| `POST /api/activities/checkout` | 志愿者 | `activityId:int64, checkOutCode:string` | `success:bool, checkOutTime:string, grantedHours:double` |

### 6.2 组织侧

| 接口 | 给谁用 | 核心入参 | 核心返回 |
| --- | --- | --- | --- |
| `POST /api/activities/create` | 有组织管理权限且能管理该组织的账号 | `orgId:int64, title:string, description:string, coverUrl:string, startTime:string, endTime:string, location:string, address:string, duration:double, maxPeople:int32` | `id:int64, message:string` |
| `PUT /api/activities/:id` | 有组织管理权限且能管理该活动所属组织的账号 | 路径参数 `id` + 可更新字段 | `message` |
| `DELETE /api/activities/:id` | 有组织管理权限且能管理该活动所属组织的账号 | 路径参数 `id` | `message` |
| `POST /api/activities/cancel/:id` | 有组织管理权限且能管理该活动所属组织的账号 | 路径参数 `id` + `reason` | `message` |
| `POST /api/activities/finish/:id` | 有组织管理权限且能管理该活动所属组织的账号 | 路径参数 `id` | `message` |
| `POST /api/activities/attendance-codes/generate/:id` | 有组织管理权限且能管理该活动所属组织的账号 | 路径参数 `id:int64` + `checkInValidMinutes:int32, checkOutValidMinutes:int32` | `success:bool, checkInCode:string, checkOutCode:string, attendanceCodeVersion:int64, attendanceCodeUpdatedAt:string, checkInExpireAt:string, checkOutExpireAt:string` |
| `POST /api/activities/attendance-codes/reset/:id` | 有组织管理权限且能管理该活动所属组织的账号 | 路径参数 `id:int64` + `codeType:int32, validMinutes:int32` | `success:bool, codeType:int32, code:string, expireAt:string, attendanceCodeVersion:int64, attendanceCodeUpdatedAt:string` |
| `GET /api/activities/attendance-codes/:id` | 有组织管理权限且能管理该活动所属组织的账号 | 路径参数 `id:int64` | `success:bool, checkInCode:string, checkOutCode:string, checkInExpireAt:string, checkOutExpireAt:string, attendanceCodeVersion:int64, attendanceCodeUpdatedAt:string` |
| `POST /api/activities/supplement-attendance` | 有组织管理权限且能管理该活动所属组织的账号 | `activityId:int64, volunteerId:int64, checkInTime:string, checkOutTime:string, reason:string` | `success:bool, checkInTime:string, checkOutTime:string, grantedHours:double` |

## 7. 审核中心

| 接口 | 给谁用 | 核心入参 | 核心返回 |
| --- | --- | --- | --- |
| `POST /api/audits/pending` | 有审核权限的账号 | `targetTypes:int32[], status:int32[], keyword:string, page:int32, pageSize:int32, createdFrom:string, createdTo:string, slaHours:int32` | `total:int32, list:PendingAuditItem[]` |
| `POST /api/audits/approval` | 有审核权限且能审核该记录的账号 | `id:int64, reason:string` | `data:{}` |
| `POST /api/audits/rejection` | 有审核权限且能审核该记录的账号 | `id:int64, reason:string` | `data:{}` |
| `POST /api/audits/batch-decision` | 有审核权限且能审核目标记录的账号 | `ids:int64[], action:int32, reason:string` | `successCount:int32, failedIds:int64[]` |
| `GET /api/audits/records/:id` | 有审核权限且能查看该记录的账号 | 路径参数 `id` | `record` |

## 8. 工时流水

| 接口 | 给谁用 | 核心入参 | 核心返回 |
| --- | --- | --- | --- |
| `POST /api/work-hours/list` | 当前志愿者本人，或有组织管理权限的账号 | `page:int32, pageSize:int32, activityId:int64, signupId:int64, operationType:int32` | `total:int32, list:WorkHourLogItem[]` |
| `POST /api/work-hours/void` | 有组织管理权限且能管理该报名所属活动的账号 | `signupId:int64, reason:string, idempotencyKey:string` | `success:bool, workHourLogId:int64` |
| `POST /api/work-hours/recalculate` | 有组织管理权限且能管理该报名所属活动的账号 | `signupId:int64, hours:double, reason:string, idempotencyKey:string` | `success:bool, workHourLogId:int64, grantedHours:double` |

## 9. 通知中心

| 接口 | 给谁用 | 核心入参 | 核心返回 |
| --- | --- | --- | --- |
| `GET /api/notifications` | 已登录用户 | 查询参数 `page:int32, pageSize:int32, unreadOnly:bool` | `total:int32, list:NotificationItem[]` |
| `POST /api/notifications/read` | 已登录用户 | `ids:int64[]` | `updated:int32` |

## 10. 看板统计

| 接口 | 给谁用 | 核心入参 | 核心返回 |
| --- | --- | --- | --- |
| `GET /api/analytics/org/funnel` | 有统计查看权限且能访问该组织的账号 | 查询参数 `orgId:int64, start:string, end:string` | `registrationCount:int64, membershipCount:int64, signupCount:int64, attendanceCount:int64, workhourCount:int64, registrationToMembershipRate:double, membershipToSignupRate:double, signupToAttendanceRate:double, attendanceToWorkhourRate:double, start:string, end:string` |
| `GET /api/analytics/org/dashboard` | 有统计查看权限且能访问该组织的账号 | 查询参数 `orgId:int64, start:string, end:string` | `signupCount:int64, approvedSignupCount:int64, attendanceCount:int64, attendanceRate:double, grantedWorkHours:double, start:string, end:string` |

## 11. RBAC 权限

| 接口 | 给谁用 | 核心入参 | 核心返回 |
| --- | --- | --- | --- |
| `GET /api/authz/roles` | 有 RBAC 全局管理权限的账号 | 查询参数 `keyword:string, includeDisabled:bool, page:int32, pageSize:int32` | `total:int32, list:RoleInfo[]` |
| `POST /api/authz/roles` | 有 RBAC 全局管理权限的账号 | `roleCode:string, roleName:string, description:string` | `id:int64` |
| `PUT /api/authz/roles/:id` | 有 RBAC 全局管理权限的账号 | 路径参数 `id:int64` + `roleName:string, description:string` | `message:string` |
| `POST /api/authz/roles/:id/status` | 有 RBAC 全局管理权限的账号 | 路径参数 `id` + `status` | `message` |
| `GET /api/authz/permissions` | 有 RBAC 全局管理权限的账号 | 查询参数 `keyword:string, onlyEnabled:bool` | `list:PermissionInfo[]` |
| `GET /api/authz/roles/:roleId/permissions` | 有 RBAC 全局管理权限的账号 | 路径参数 `roleId:int64` | `roleId:int64, permissions:RolePermissionItem[]` |
| `POST /api/authz/roles/:roleId/permissions/set` | 有 RBAC 全局管理权限的账号 | 路径参数 `roleId` + `permissionIds` | `message` |
| `GET /api/authz/grants` | 有 RBAC 全局管理权限的账号 | 查询参数 `accountId:int64, scopeType:string, scopeId:int64, onlyActive:bool, page:int32, pageSize:int32` | `total:int32, list:AccountRoleBindingInfo[]` |
| `POST /api/authz/grants` | 有 RBAC 全局管理权限的账号 | `accountId:int64, roleId:int64, scopeType:string, scopeId:int64, expiresAt:string, remark:string` | `message:string` |
| `POST /api/authz/grants/:bindingId/revoke` | 有 RBAC 全局管理权限的账号 | 路径参数 `bindingId` + `remark` | `message` |
| `GET /api/authz/me` | 已登录用户 | 无 | `accountId:int64, roles:AccountRoleBindingInfo[], permissions:MyPermissionScope[]` |

## 12. 导入导出

### 12.1 导出

| 接口 | 给谁用 | 核心入参 | 返回 |
| --- | --- | --- | --- |
| `POST /api/admin/export/volunteers` | 有导出权限的账号；若存在多个组织作用域则此接口可能无法直接确定组织 | `idList:int64[], keyword:string, auditStatus:int32, status:int32` | Excel 文件流 |
| `POST /api/admin/export/activities` | 有导出权限的账号；若存在多个组织作用域则此接口可能无法直接确定组织 | `idList:int64[], keyword:string, status:int32, startFrom:string, startTo:string` | Excel 文件流 |
| `POST /api/admin/export/ops-report` | 有导出权限且能访问指定组织的账号 | `periodType:string, orgId:int64, start:string, end:string` | Excel 文件流 |

说明：

- 前端请求建议使用 `blob`
- 文件名从响应头 `Content-Disposition` 获取

### 12.2 导入

| 接口 | 给谁用 | 请求格式 | 核心返回 |
| --- | --- | --- | --- |
| `POST /api/admin/import/volunteers` | 有组织管理权限的账号 | `multipart/form-data(file:binary)` | `totalCount:int32, successCount:int32, failedCount:int32, errorFileName:string, errorFileContentType:string, errorFileContent:bytes(base64), failures:ImportFailureItem[]` |
| `POST /api/admin/import/activities` | 有组织管理权限且能管理导入数据所属组织的账号 | `multipart/form-data(file:binary)` | `totalCount:int32, successCount:int32, failedCount:int32, errorFileName:string, errorFileContentType:string, errorFileContent:bytes(base64), failures:ImportFailureItem[]` |

## 13. AI 助手

| 接口 | 给谁用 | 核心入参 | 核心返回 |
| --- | --- | --- | --- |
| `POST /api/assistant/sessions` | 已登录用户 | `scene:string, title:string` | `session_id:int64` |
| `POST /api/assistant/chat` | 已登录用户 | `session_id:int64, message:string, stream:bool` | `reply:string, tool_calls:AssistantToolCall[], usage:AssistantUsage` |
| `GET /api/assistant/sessions/:id/messages` | 已登录用户 | 路径参数 `id` | `list[]` |
| `POST /api/assistant/actions/activity-draft` | 已登录用户；实际能否生成具体组织草案取决于该用户是否拥有对应组织成员/可访问组织范围 | `session_id:int64, topic:string, target_people:string, location:string` | `session_id:int64, result:AssistantChatResponse` |
| `POST /api/assistant/chat/stream` | 已登录用户 | `session_id:int64, message:string` | SSE 事件流 |

SSE 事件类型：

- `start`
- `delta`
- `tool`
- `usage`
- `done`

## 14. 联调优先顺序

1. 登录 / 刷新令牌
2. 志愿者首页 / 组织列表 / 活动列表
3. 活动报名 / 取消 / 签到 / 签退
4. 组织侧活动管理 / 成员管理 / 审核
5. 通知 / 看板 / 工时 / 导入导出 / AI

补充说明：
- 志愿者注册里的 `gender` 当前只接受 `男`、`女`、`未知`
- 登录返回里的 `displayName` 当前默认等于 `userName`
- 组织列表/详情中的 `email`、`organizationType`、`region`、`websiteUrl` 当前通常为空字符串
- 组织创建/更新虽然接收 `email`、`organizationType`、`region`、`websiteUrl`，但当前后端未持久化这些字段
- `/api/assistant/chat/stream` 内部会强制将 `stream` 设为 `false`，前端直接调用流式路由时可不传这个字段
