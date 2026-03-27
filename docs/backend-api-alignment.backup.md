# 后端接口联调整理

> 统计时间：2026-03-27
>
> 整理口径：
>
> - 页面与路由以前端仓库 `src/router/routes.ts` 和 `docs/frontend-api-current-integration.md` 为准
> - 接口路径以当前后端实际注册路由 `internal/router/*.go` 为准
> - 请求/响应结构以 `docs/api/*.md` 与现有接口文档为辅进行整理
> - 主表只保留当前前端页面真正承接到的接口；未进入当前联调范围的能力单独说明
> - 不纳入 RBAC/Authz 这类当前无页面承接、也不面向志愿者或组织管理者使用的后台治理接口

## 使用说明

- 按页面整理，不按后端模块目录整理
- 一个用户动作可对应多个接口
- `前端状态` 取值使用：`已承接`、`已封装未承接`、`未接入`、`待确认`
- 少数历史模块文档中的路径写法与当前 `internal/router` 不一致；本表已按当前实际注册路径修正

## 字段填写约定

- 请求参数列：
  `字段名:类型` 表示字段与类型，`?` 表示可选，未标 `?` 默认为必填
- 响应结构列：
  重点描述统一响应包装里的 `data` 结构；文件流和 `text/event-stream` 不按 JSON 展开
- 常见类型：
  `int64` 主键/ID，`int32` 枚举或分页参数，`double` 工时等浮点值，`string(datetime)` 表示时间字符串
- 路径参数：
  在表格里写成 `路径参数 activityId:int64`
- 查询参数：
  在表格里写成 `query:` 前缀
- 详细对象结构：
  表格中若写 `见“常用 data 结构速查”`，表示关键字段在文末统一展开

## 1. 登录注册

| 页面 | 页面路由 | 用户动作 | 接口用途 | 方法 | 路径 | 请求参数 | 响应结构 | 权限 | 枚举/字典 | 前端状态 | 差异说明 | 负责人 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 登录页 | `/login` | 登录 | 账号登录并签发令牌 | `POST` | `/api/login` | `loginType:string` `identifier:string` `password:string` `identity:string` | `data.accessToken` `data.refreshToken` `data.expiresAt` `data.userInfo` | 无需登录 | `identity` | 已承接 | - | 待定 | 志愿者/组织管理者共用 |
| 注册页 | `/register` | 志愿者注册 | 创建志愿者账号 | `POST` | `/api/volunteer/register` | `name` `phone` `email` `password` `age` `gender` `userName` | `data:{}` | 无需登录 | `gender` | 已承接 | - | 待定 | - |
| 注册页 | `/register` | 组织注册 | 创建组织管理者账号 | `POST` | `/api/organization/register` | `name` `phone` `email` `password` `organizationName` `code` `userName` | `data:{}` | 无需登录 | - | 已承接 | - | 待定 | - |

## 2. 志愿者个人中心

| 页面 | 页面路由 | 用户动作 | 接口用途 | 方法 | 路径 | 请求参数 | 响应结构 | 权限 | 枚举/字典 | 前端状态 | 差异说明 | 负责人 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 志愿者首页 | `/volunteer` | 加载首页摘要 | 获取志愿者首页统计 | `GET` | `/api/volunteers/home/summary` | 无 | `data.nickname` `data.level` `data.stats.points` `data.stats.hours` `data.stats.activityCount` | 志愿者登录 | `level` | 已承接 | - | 待定 | `VolunteerLayout` 进入后会加载 |
| 个人信息页 | `/volunteer/profile` | 加载资料 | 获取我的完整资料 | `GET` | `/api/me/profile` | 无 | `data.volunteer` `data.accountInfo` `data.profile` `data.verification` | 志愿者登录 | `gender` `auditStatus` `status` | 已承接 | 旧模块文档曾写成带 `:id` 路径，当前实际为 `/api/me/profile` | 待定 | 以当前 router 为准 |
| 个人信息页 | `/volunteer/profile` | 保存账户信息 | 更新账号资料 | `PUT` | `/api/me/account` | `userName` `email` `phone` | `data:{}` | 志愿者登录 | - | 已承接 | 旧模块文档曾写成 `/api/volunteers/account` | 待定 | - |
| 个人信息页 | `/volunteer/profile` | 保存个人资料 | 更新志愿者资料 | `PUT` | `/api/me/volunteer-profile` | `gender` `birthday` `avatarUrl` `introduction` | `data:{}` | 志愿者登录 | `gender` | 已承接 | 旧模块文档曾写成 `/api/volunteers/:id` | 待定 | - |
| 个人信息页 | `/volunteer/profile` | 提交实名 | 提交实名认证申请 | `POST` | `/api/volunteers/real-name/submit` | `realName` `idCard` | `data.auditId` `data.status` | 志愿者登录 | `audit.status` | 已承接 | - | 待定 | 会进入审核链路 |

## 3. 志愿者活动

| 页面 | 页面路由 | 用户动作 | 接口用途 | 方法 | 路径 | 请求参数 | 响应结构 | 权限 | 枚举/字典 | 前端状态 | 差异说明 | 负责人 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 活动列表页 | `/volunteer/activities` | 加载活动列表 | 查询可浏览活动 | `POST` | `/api/activities` | `page` `pageSize` `status?` `keyword?` `startFrom?` `startTo?` `registeredOnly=false` | `data.total` `data.list[]` `list[].isRegistered` `list[].isFull` | 志愿者登录 | `activity.status` | 已承接 | 前后端已统一为单接口，不再拆 `/my` | 待定 | “活动大厅”传 `registeredOnly=false/不传` |
| 活动列表页 | `/volunteer/activities` | 点击报名 | 提交活动报名 | `POST` | `/api/activities/signup` | `activityId:int64` | `data.success` | 志愿者登录 | - | 已承接 | - | 待定 | - |
| 活动列表页 | `/volunteer/activities` | 取消报名 | 取消活动报名 | `POST` | `/api/activities/cancel` | `activityId:int64` | `data.success` | 志愿者登录 | - | 已承接 | - | 待定 | - |
| 活动详情页 | `/volunteer/activities/:id` | 加载详情 | 查询活动详情 | `GET` | `/api/activities/:activityId` | 路径参数 `activityId` | `data.activity` `activity.checkInStatus` `activity.checkOutStatus` `activity.grantedHours` | 志愿者登录 | `activity.status` `checkInStatus` `checkOutStatus` `workHourStatus` | 已承接 | 模块文档中的 `:id` 已换成当前 router 参数名理解 | 待定 | 组织端活动详情也复用 |
| 活动详情页 | `/volunteer/activities/:id` | 签到 | 提交签到码 | `POST` | `/api/activities/checkin` | `activityId` `checkInCode` | `data.success` `data.checkInTime` | 志愿者登录 | - | 已承接 | - | 待定 | - |
| 活动详情页 | `/volunteer/activities/:id` | 签退 | 提交签退码并触发工时结算 | `POST` | `/api/activities/checkout` | `activityId` `checkOutCode` | `data.success` `data.checkOutTime` `data.grantedHours` | 志愿者登录 | - | 已承接 | - | 待定 | - |
| 我的报名页 | `/volunteer/activities/my-registrations` | 加载我的报名 | 查询当前志愿者已报名活动 | `POST` | `/api/activities` | `page` `pageSize` `registeredOnly:true` `keyword?` `status?` | `data.total` `data.list[]` | 志愿者登录 | `activity.status` | 已承接 | 原 `/api/activities/my` 已删除，前端已切换到统一接口 | 待定 | 当前仅返回待审核/已报名成功活动 |

## 4. 志愿者组织

| 页面 | 页面路由 | 用户动作 | 接口用途 | 方法 | 路径 | 请求参数 | 响应结构 | 权限 | 枚举/字典 | 前端状态 | 差异说明 | 负责人 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 我的组织页 | `/volunteer/organizations` | 加载公开组织列表 | 查询可加入组织 | `POST` | `/api/organizations/public-list` | `keyword?` `organizationType?` `region?` `page` `pageSize` | `data.total` `data.list[]` | 志愿者登录 | `organization.status` | 已承接 | 当前公开列表只返回正常状态组织 | 待定 | `contactPhone` 不返回真实联系电话 |
| 我的组织页 | `/volunteer/organizations` | 加载我加入的组织 | 查询我的组织关系 | `GET` | `/api/me/organizations` | `status?` `page` `pageSize` | `data.total` `data.list[]` | 志愿者登录 | `membership.status` `membership.role` | 已承接 | 模块文档旧写法为 `/api/volunteers/:volunteerId/organizations`，当前实际为 `/api/me/organizations` | 待定 | 以当前 router 为准 |
| 我的组织页 | `/volunteer/organizations` | 申请加入 | 创建成员申请 | `POST` | `/api/memberships/join` | `volunteerId` `organizationId` | `data.membershipId` `data.status` `data.message` | 志愿者登录 | `membership.status` | 已承接 | - | 待定 | - |
| 我的组织页 | `/volunteer/organizations` | 退出组织 | 解除成员关系 | `POST` | `/api/memberships/leave` | `membershipId` `reason` | `data.message` | 志愿者登录 | - | 已承接 | - | 待定 | - |

## 5. 组织信息管理

| 页面 | 页面路由 | 用户动作 | 接口用途 | 方法 | 路径 | 请求参数 | 响应结构 | 权限 | 枚举/字典 | 前端状态 | 差异说明 | 负责人 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 组织信息页 | `/organization/organization-info` | 加载组织列表 | 查询组织列表 | `POST` | `/api/organizations/list` | `keyword?` `status?[]` `organizationType?` `region?` `page` `pageSize` | `data.total` `data.list[]` | 组织管理者登录 | `organization.status` | 已承接 | - | 待定 | - |
| 组织信息页 | `/organization/organization-info` | 高级搜索 | 按更多条件搜索组织 | `POST` | `/api/organizations/search` | `keyword?` `status?[]` `organizationType?` `region?` `startDate?` `endDate?` `page` `pageSize` | `data.total` `data.list[]` | 组织管理者登录 | `organization.status` | 已承接 | - | 待定 | 与普通列表并存 |
| 组织信息页 | `/organization/organization-info` | 查看组织详情 | 查询单个组织完整信息 | `GET` | `/api/organizations/:organizationId` | 路径参数 `organizationId` | `data.organization` `data.accountInfo` `data.organizationProfile` `data.organizationCertification` | 组织管理者登录 | `organization.status` | 已承接 | - | 待定 | - |
| 组织信息页 | `/organization/organization-info` | 新建组织 | 创建组织 | `POST` | `/api/organizations/create` | `name` `organizationCode` `contactPerson` `contactPhone` `email` `address` `organizationType` `region` `description` `websiteUrl` `logoUrl` | `data.id` `data.message` | 组织管理者登录 | `organizationType` | 已承接 | - | 待定 | - |
| 组织信息页 | `/organization/organization-info` | 修改账户信息 | 更新组织管理者账号 | `PUT` | `/api/organizations/account` | `userName` `email` `phone` | `data:{}` | 组织管理者登录 | - | 已承接 | - | 待定 | - |
| 组织信息页 | `/organization/organization-info` | 修改组织信息 | 更新组织资料 | `PUT` | `/api/organizations/:organizationId` | 路径参数 `organizationId` + 资料字段 | `data.message` | 组织管理者登录 | `organizationType` | 已承接 | - | 待定 | - |
| 组织信息页 | `/organization/organization-info` | 删除组织 | 删除单个组织 | `DELETE` | `/api/organizations/:organizationId` | 路径参数 `organizationId` | `data.message` | 组织管理者登录 | - | 已承接 | - | 待定 | - |
| 组织信息页 | `/organization/organization-info` | 停用组织 | 单个停用 | `POST` | `/api/organizations/:organizationId/disable` | 路径参数 `organizationId` `reason` | `data.message` | 组织管理者登录 | `organization.status` | 已承接 | - | 待定 | - |
| 组织信息页 | `/organization/organization-info` | 启用组织 | 单个启用 | `POST` | `/api/organizations/:organizationId/enable` | 路径参数 `organizationId` `reason` | `data.message` | 组织管理者登录 | `organization.status` | 已承接 | - | 待定 | - |
| 组织信息页 | `/organization/organization-info` | 批量删除 | 批量删除组织 | `POST` | `/api/organizations/bulk-delete` | `ids:int64[]` | `data.successCount` `data.failedCount` `data.message` | 组织管理者登录 | - | 已承接 | - | 待定 | - |
| 组织信息页 | `/organization/organization-info` | 批量停用 | 批量停用组织 | `POST` | `/api/organizations/batch-disable` | `ids:int64[]` `reason` | `data.successCount` `data.failedIds[]` `data.message` | 组织管理者登录 | `organization.status` | 已承接 | - | 待定 | - |
| 组织信息页 | `/organization/organization-info` | 批量启用 | 批量启用组织 | `POST` | `/api/organizations/batch-enable` | `ids:int64[]` `reason` | `data.successCount` `data.failedIds[]` `data.message` | 组织管理者登录 | `organization.status` | 已承接 | - | 待定 | - |

## 6. 志愿者管理

| 页面 | 页面路由 | 用户动作 | 接口用途 | 方法 | 路径 | 请求参数 | 响应结构 | 权限 | 枚举/字典 | 前端状态 | 差异说明 | 负责人 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 志愿者管理页 | `/organization/volunteers` | 加载志愿者列表 | 查询志愿者分页列表 | `POST` | `/api/volunteers/list` | `keyword?` `page` `pageSize` | `data.total` `data.list[]` | 组织管理者登录 | `gender` `auditStatus` `status` | 已承接 | - | 待定 | - |
| 志愿者管理页 | `/organization/volunteers` | 查看志愿者详情 | 查看单个志愿者 | `GET` | `/api/volunteers/:volunteerId` | 路径参数 `volunteerId` | `data.volunteer` | 组织管理者登录 | `gender` `auditStatus` `status` | 已承接 | 模块文档旧路径示例为 `/api/volunteers/detail/:id` | 待定 | 当前 router 为 `/api/volunteers/:volunteerId` |
| 志愿者管理页 | `/organization/volunteers` | 导入志愿者 | 批量导入志愿者 | `POST` | `/api/admin/import/volunteers` | `multipart/form-data` `file` | `data.totalCount` `data.successCount` `data.failedCount` `data.failures[]` | 组织管理者登录 | - | 已承接 | - | 待定 | 返回可包含错误文件内容 |
| 志愿者管理页 | `/organization/volunteers` | 导出志愿者 | 批量导出志愿者 | `POST` | `/api/admin/export/volunteers` | `idList?[]` `keyword?` `auditStatus?` `status?` | 文件流 | 组织管理者登录 | `auditStatus` `status` | 已承接 | 非 JSON 包装 | 待定 | 下载文件 |

## 7. 成员管理

| 页面 | 页面路由 | 用户动作 | 接口用途 | 方法 | 路径 | 请求参数 | 响应结构 | 权限 | 枚举/字典 | 前端状态 | 差异说明 | 负责人 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 成员管理页 | `/organization/members` | 加载成员列表 | 查询组织成员 | `GET` | `/api/organizations/:organizationId/members` | 路径参数 `organizationId` + 查询 `status?` `role?` `keyword?` `page` `pageSize` | `data.total` `data.list[]` | 组织管理者登录 | `membership.status` `membership.role` | 已承接 | - | 待定 | - |
| 成员管理页 | `/organization/members` | 加载成员统计 | 查询成员状态统计 | `GET` | `/api/memberships/stats` | `organizationId:int64` | `data.pendingCount` `data.activeCount` `data.inactiveCount` `data.suspendedCount` `data.totalCount` | 组织管理者登录 | `membership.status` | 已承接 | - | 待定 | - |
| 成员管理页 | `/organization/members` | 更新成员状态 | 审核或调整成员关系 | `POST` | `/api/memberships/status/update` | `membershipId` `status` `reviewComment` | `data.message` | 组织管理者登录 | `membership.status` | 已承接 | - | 待定 | - |

## 8. 活动管理

| 页面 | 页面路由 | 用户动作 | 接口用途 | 方法 | 路径 | 请求参数 | 响应结构 | 权限 | 枚举/字典 | 前端状态 | 差异说明 | 负责人 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 创建活动页 | `/organization/activities/create` | 创建活动 | 提交活动创建 | `POST` | `/api/activities/create` | `orgId` `title` `description` `coverUrl` `startTime` `endTime` `location` `address` `duration` `maxPeople` | `data.id` `data.message` | 组织管理者登录 | `activity.status` | 已承接 | - | 待定 | - |
| 活动管理页 | `/organization/activities` | 加载活动列表 | 查询活动列表 | `POST` | `/api/activities` | `page` `pageSize` `status?` `keyword?` `startFrom?` `startTo?` | `data.total` `data.list[]` | 组织管理者登录 | `activity.status` | 已承接 | 同一接口同时供志愿者端和组织端使用 | 待定 | - |
| 活动管理页 | `/organization/activities` | 更新活动 | 修改活动资料 | `PUT` | `/api/activities/:activityId` | 路径参数 `activityId` + 活动字段 | `data.message` | 组织管理者登录 | `activity.status` | 已承接 | - | 待定 | - |
| 活动管理页 | `/organization/activities` | 删除活动 | 删除活动 | `DELETE` | `/api/activities/:activityId` | 路径参数 `activityId` | `data.message` | 组织管理者登录 | - | 已承接 | - | 待定 | - |
| 活动管理页 | `/organization/activities` | 取消活动 | 标记活动取消 | `POST` | `/api/activities/:activityId/cancel` | 路径参数 `activityId` `reason` | `data.message` | 组织管理者登录 | `activity.status` | 已承接 | 模块文档旧写法顺序不同，当前实际为 `/:activityId/cancel` | 待定 | - |
| 活动管理页 | `/organization/activities` | 结束活动 | 完结活动 | `POST` | `/api/activities/:activityId/finish` | 路径参数 `activityId` | `data.message` | 组织管理者登录 | `activity.status` | 已承接 | 模块文档旧写法顺序不同，当前实际为 `/:activityId/finish` | 待定 | - |
| 活动管理页 | `/organization/activities` | 查看签到码 | 查询签到签退码 | `GET` | `/api/activities/:activityId/attendance-codes` | 路径参数 `activityId` | `data.checkInCode` `data.checkOutCode` `data.checkInExpireAt` `data.checkOutExpireAt` | 组织管理者登录 | `codeType` | 已承接 | - | 待定 | - |
| 活动管理页 | `/organization/activities` | 生成签到码 | 生成签到签退码 | `POST` | `/api/activities/:activityId/attendance-codes/generate` | 路径参数 `activityId` + `checkInValidMinutes` `checkOutValidMinutes` | `data.checkInCode` `data.checkOutCode` `data.attendanceCodeVersion` | 组织管理者登录 | - | 已承接 | - | 待定 | - |
| 活动管理页 | `/organization/activities` | 重置签到码 | 重置签到或签退码 | `POST` | `/api/activities/:activityId/attendance-codes/reset` | 路径参数 `activityId` + `codeType` `validMinutes` | `data.codeType` `data.code` `data.expireAt` | 组织管理者登录 | `codeType` | 已承接 | - | 待定 | - |
| 活动管理页 | `/organization/activities` | 补录考勤 | 补录签到签退并补发工时 | `POST` | `/api/activities/supplement-attendance` | `activityId` `volunteerId` `checkInTime` `checkOutTime` `reason` | `data.success` `data.checkInTime` `data.checkOutTime` `data.grantedHours` | 组织管理者登录 | - | 已承接 | - | 待定 | - |
| 活动管理页 | `/organization/activities` | 导入活动 | 批量导入活动 | `POST` | `/api/admin/import/activities` | `multipart/form-data` `file` | `data.totalCount` `data.successCount` `data.failedCount` `data.failures[]` | 组织管理者登录 | - | 已承接 | - | 待定 | - |
| 活动管理页 | `/organization/activities` | 导出活动 | 批量导出活动 | `POST` | `/api/admin/export/activities` | `idList?[]` `keyword?` `status?` `startFrom?` `startTo?` | 文件流 | 组织管理者登录 | `activity.status` | 已承接 | 非 JSON 包装 | 待定 | 下载文件 |

## 9. 工时流水

| 页面 | 页面路由 | 用户动作 | 接口用途 | 方法 | 路径 | 请求参数 | 响应结构 | 权限 | 枚举/字典 | 前端状态 | 差异说明 | 负责人 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 工时流水页 | `/organization/statistics/financial` | 加载流水 | 查询工时流水列表 | `POST` | `/api/work-hours/list` | `page` `pageSize` `activityId?` `signupId?` `operationType?` | `data.total` `data.list[]` | 组织管理者登录 | `operationType` | 已承接 | - | 待定 | - |
| 工时流水页 | `/organization/statistics/financial` | 作废工时 | 冲销已发放工时 | `POST` | `/api/work-hours/void` | `signupId` `reason` `idempotencyKey` | `data.success` `data.workHourLogId` | 组织管理者登录 | `operationType` | 已承接 | - | 待定 | 建议前端保证幂等键唯一 |
| 工时流水页 | `/organization/statistics/financial` | 重算工时 | 重算并重新发放工时 | `POST` | `/api/work-hours/recalculate` | `signupId` `hours` `reason` `idempotencyKey` | `data.success` `data.workHourLogId` `data.grantedHours` | 组织管理者登录 | `operationType` | 已承接 | - | 待定 | 建议前端保证幂等键唯一 |

## 10. 通知中心

| 页面 | 页面路由 | 用户动作 | 接口用途 | 方法 | 路径 | 请求参数 | 响应结构 | 权限 | 枚举/字典 | 前端状态 | 差异说明 | 负责人 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 通知页 | `/organization/notifications` | 加载通知列表 | 查询当前账号通知 | `GET` | `/api/notifications` | `page` `pageSize` `unreadOnly?` | `data.total` `data.list[]` | 已登录用户 | `readStatus` `eventType` `bizType` | 已承接 | - | 待定 | 志愿者/组织管理者都可用，当前页面主要在组织端 |
| 通知页 | `/organization/notifications` | 标记已读 | 批量通知已读 | `POST` | `/api/notifications/read` | `ids:int64[]` | `data.updated` | 已登录用户 | - | 已承接 | - | 待定 | - |

## 11. 数据统计

| 页面 | 页面路由 | 用户动作 | 接口用途 | 方法 | 路径 | 请求参数 | 响应结构 | 权限 | 枚举/字典 | 前端状态 | 差异说明 | 负责人 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 组织首页 | `/organization` | 加载看板数据 | 获取运营看板汇总 | `GET` | `/api/analytics/org/dashboard` | `orgId` `start` `end` | `data.signupCount` `data.approvedSignupCount` `data.attendanceCount` `data.attendanceRate` `data.grantedWorkHours` | 组织管理者登录 | - | 已承接 | - | 待定 | `Dashboard` 页使用 |
| 统计总览页 | `/organization/statistics` | 加载统计概览 | 获取运营看板汇总 | `GET` | `/api/analytics/org/dashboard` | `orgId` `start` `end` | `data.signupCount` `data.approvedSignupCount` `data.attendanceCount` `data.attendanceRate` `data.grantedWorkHours` | 组织管理者登录 | - | 已承接 | 当前页面入口较弱，但路由已注册且代码在用 | 待定 | - |
| 活动统计页 | `/organization/statistics/activities` | 加载活动统计 | 获取转化漏斗统计 | `GET` | `/api/analytics/org/funnel` | `orgId` `start` `end` | `data.registrationCount` `data.membershipCount` `data.signupCount` `data.attendanceCount` `data.workhourCount` | 组织管理者登录 | - | 已承接 | - | 待定 | - |
| 志愿者统计页 | `/organization/statistics/volunteers` | 加载志愿者统计 | 获取运营看板汇总 | `GET` | `/api/analytics/org/dashboard` | `orgId` `start` `end` | `data.signupCount` `data.approvedSignupCount` `data.attendanceCount` `data.attendanceRate` `data.grantedWorkHours` | 组织管理者登录 | - | 已承接 | 当前后端暂无独立“志愿者统计”接口，页面复用总览统计能力 | 待定 | 待后续是否拆分专用接口 |
| 组织首页/统计页 | `/organization` `/organization/statistics` | 导出报表 | 导出运营报表 | `POST` | `/api/admin/export/ops-report` | `periodType` `orgId` `start` `end` | 文件流 | 组织管理者登录 | `periodType` | 已承接 | 非 JSON 包装 | 待定 | 下载文件 |

## 12. AI 助手

| 页面 | 页面路由 | 用户动作 | 接口用途 | 方法 | 路径 | 请求参数 | 响应结构 | 权限 | 枚举/字典 | 前端状态 | 差异说明 | 负责人 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| AI 助手页 | `/organization/assistant` | 创建会话 | 新建 AI 会话 | `POST` | `/api/assistant/sessions` | `scene` `title` | `data.session_id` | 已登录用户 | `scene` | 已承接 | - | 待定 | 当前组织端页面在用 |
| AI 助手页 | `/organization/assistant` | 加载历史消息 | 查询会话消息 | `GET` | `/api/assistant/sessions/:id/messages` | 路径参数 `id` | `data.list[]` | 已登录用户 | `role` `finish_reason` | 已承接 | - | 待定 | - |
| AI 助手页 | `/organization/assistant` | 发送消息 | 流式 AI 对话 | `POST` | `/api/assistant/chat/stream` | `session_id` `message` `stream` | `text/event-stream` | 已登录用户 | SSE 事件：`start` `message` `error` `done` | 已承接 | 页面当前使用流式接口 | 待定 | - |
| AI 助手页 | `/organization/assistant` | 生成活动草案 | AI 生成活动草案 | `POST` | `/api/assistant/actions/activity-draft` | `session_id` `topic` `target_people` `location` | `data.session_id` `data.result.reply` `data.result.tool_calls[]` | 已登录用户 | - | 已承接 | - | 待定 | - |
| AI 助手页 | `/organization/assistant` | 非流式对话 | 普通 AI 对话 | `POST` | `/api/assistant/chat` | `session_id` `message` `stream` | `data.reply` `data.tool_calls[]` `data.usage` | 已登录用户 | - | 已封装未承接 | 前端 API 已封装，但当前页面未实际调用 | 待定 | 当前代码主要走流式 |

## 13. 导入导出汇总

| 页面 | 页面路由 | 用户动作 | 接口用途 | 方法 | 路径 | 请求参数 | 响应结构 | 权限 | 枚举/字典 | 前端状态 | 差异说明 | 负责人 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 志愿者管理页 | `/organization/volunteers` | 导入志愿者 | 批量导入 | `POST` | `/api/admin/import/volunteers` | `multipart/form-data` `file` | `data.totalCount` `data.successCount` `data.failedCount` | 组织管理者登录 | - | 已承接 | - | 待定 | 同 6 节 |
| 志愿者管理页 | `/organization/volunteers` | 导出志愿者 | 批量导出 | `POST` | `/api/admin/export/volunteers` | `idList?[]` `keyword?` `auditStatus?` `status?` | 文件流 | 组织管理者登录 | `auditStatus` `status` | 已承接 | - | 待定 | 同 6 节 |
| 活动管理页 | `/organization/activities` | 导入活动 | 批量导入活动 | `POST` | `/api/admin/import/activities` | `multipart/form-data` `file` | `data.totalCount` `data.successCount` `data.failedCount` | 组织管理者登录 | - | 已承接 | - | 待定 | 同 8 节 |
| 活动管理页 | `/organization/activities` | 导出活动 | 批量导出活动 | `POST` | `/api/admin/export/activities` | `idList?[]` `keyword?` `status?` `startFrom?` `startTo?` | 文件流 | 组织管理者登录 | `activity.status` | 已承接 | - | 待定 | 同 8 节 |
| 组织首页/统计页 | `/organization` `/organization/statistics` | 导出运营报表 | 导出报表 | `POST` | `/api/admin/export/ops-report` | `periodType` `orgId` `start` `end` | 文件流 | 组织管理者登录 | `periodType` | 已承接 | - | 待定 | 同 11 节 |

## 14. 审核

> 当前前端梳理口径里，审核能力暂不计入“已承接页面主表”，但仓库中已有相关 API 调用和后端稳定接口，因此单独列出。

| 页面 | 页面路由 | 用户动作 | 接口用途 | 方法 | 路径 | 请求参数 | 响应结构 | 权限 | 枚举/字典 | 前端状态 | 差异说明 | 负责人 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 审核能力 | 暂不纳入当前页面承接范围 | 查询审核列表 | 获取待审核数据 | `POST` | `/api/audits/pending` | `targetTypes[]` `status[]` `keyword?` `page` `pageSize` `createdFrom?` `createdTo?` `slaHours?` | `data.total` `data.list[]` | 组织管理者登录 | `targetType` `audit.status` | 未接入 | 代码存在调用，但当前联调主清单未纳入 | 待定 | 单独跟进更合适 |
| 审核能力 | 暂不纳入当前页面承接范围 | 查看审核详情 | 获取审核记录详情 | `GET` | `/api/audits/records/:id` | 路径参数 `id` | `data.record` | 组织管理者登录 | `targetType` `auditResult` `status` | 未接入 | - | 待定 | - |
| 审核能力 | 暂不纳入当前页面承接范围 | 审核通过 | 提交通过动作 | `POST` | `/api/audits/approval` | `id` `reason` | `data:{}` | 组织管理者登录 | - | 未接入 | - | 待定 | - |
| 审核能力 | 暂不纳入当前页面承接范围 | 审核驳回 | 提交驳回动作 | `POST` | `/api/audits/rejection` | `id` `reason` | `data:{}` | 组织管理者登录 | - | 未接入 | - | 待定 | - |
| 审核能力 | 暂不纳入当前页面承接范围 | 批量审核 | 提交批量审核动作 | `POST` | `/api/audits/batch-decision` | `ids[]` `action` `reason` | `data.successCount` `data.failedIds[]` | 组织管理者登录 | `action` | 未接入 | - | 待定 | - |

## 未纳入当前联调主表的后端能力

### 1. 全局会话类接口

| 能力 | 方法 | 路径 | 当前情况 | 备注 |
| --- | --- | --- | --- | --- |
| 注销 | `POST` | `/api/logout` | 有 UI 触发，但不属于某个业务页面主表 | 全局会话动作 |
| 刷新令牌 | `POST` | `/api/refresh` | 已接入请求刷新逻辑，但不属于页面联调 | 拦截器/登录态续期使用 |

## 常用 data 结构速查

### 1. 登录与当前用户

- `LoginResponse.data`
  `accessToken:string`
  `refreshToken:string`
  `expiresAt:int64`
  `userInfo.accountId:string`
  `userInfo.userName:string`
  `userInfo.email:string`
  `userInfo.phone:string`
  `userInfo.displayName:string`
  `userInfo.avatarUrl:string`
  `userInfo.identity:string`

### 2. 志愿者相关

- `MyProfileResponse.data`
  `volunteer.id:int64`
  `volunteer.realName:string`
  `volunteer.gender:int32`
  `volunteer.birthday:string`
  `volunteer.idCard:string`
  `volunteer.avatarUrl:string`
  `volunteer.introduction:string`
  `volunteer.totalHours:double`
  `volunteer.serviceCount:int32`
  `volunteer.creditScore:int32`
  `volunteer.auditStatus:int32`
  `volunteer.status:int32`
  `accountInfo.userName:string`
  `accountInfo.email:string`
  `accountInfo.phone:string`
  `profile.gender:int32`
  `profile.birthday:string`
  `profile.avatarUrl:string`
  `profile.introduction:string`
  `verification.realName:string`
  `verification.idCard:string`
  `verification.auditStatus:int32`

- `VolunteerListResponse.data`
  `total:int32`
  `list[].id:int64`
  `list[].accountId:int64`
  `list[].realName:string`
  `list[].gender:int32`
  `list[].avatarUrl:string`
  `list[].totalHours:double`
  `list[].serviceCount:int32`
  `list[].creditScore:int32`
  `list[].auditStatus:int32`
  `list[].status:int32`
  `list[].createdAt:string(datetime)`

- `VolunteerHomeSummaryResponse.data`
  `nickname:string`
  `level:int32`
  `stats.points:int32`
  `stats.hours:double`
  `stats.activityCount:int32`
  `monthlyGrowth:double`
  `needHoursToNextLevel:double`

### 3. 组织相关

- `OrganizationListResponse.data`
  `total:int32`
  `list[].id:int64`
  `list[].name:string`
  `list[].organizationCode:string`
  `list[].contactPerson:string`
  `list[].contactPhone:string`
  `list[].email:string`
  `list[].address:string`
  `list[].status:int32`
  `list[].organizationType:string`
  `list[].region:string`
  `list[].createdAt:string(datetime)`

- `OrganizationDetailResponse.data`
  `organization.id:int64`
  `organization.accountId:int64`
  `organization.name:string`
  `organization.organizationCode:string`
  `organization.contactPerson:string`
  `organization.contactPhone:string`
  `organization.email:string`
  `organization.address:string`
  `organization.status:int32`
  `organization.organizationType:string`
  `organization.region:string`
  `organization.description:string`
  `organization.websiteUrl:string`
  `organization.logoUrl:string`
  `organization.createdAt:string(datetime)`
  `organization.updatedAt:string(datetime)`
  `accountInfo.userName:string`
  `accountInfo.email:string`
  `accountInfo.phone:string`
  `organizationProfile.name:string`
  `organizationProfile.contactPerson:string`
  `organizationProfile.contactPhone:string`
  `organizationProfile.address:string`
  `organizationProfile.description:string`
  `organizationProfile.logoUrl:string`
  `organizationCertification.organizationCode:string`

### 4. 成员关系相关

- `VolunteerOrganizationsResponse.data`
  `total:int32`
  `list[].membershipId:int64`
  `list[].organizationId:int64`
  `list[].organizationName:string`
  `list[].organizationCode:string`
  `list[].status:int32`
  `list[].role:int32`
  `list[].position:string`
  `list[].joinDate:string(datetime)`
  `list[].reviewDate:string(datetime)`
  `list[].reviewComment:string`

- `OrganizationMembersResponse.data`
  `total:int32`
  `list[].membershipId:int64`
  `list[].volunteerId:int64`
  `list[].volunteerName:string`
  `list[].volunteerCode:string`
  `list[].organizationId:int64`
  `list[].organizationName:string`
  `list[].status:int32`
  `list[].role:int32`
  `list[].position:string`
  `list[].motivation:string`
  `list[].expectedHours:string`
  `list[].joinDate:string(datetime)`
  `list[].reviewDate:string(datetime)`
  `list[].reviewComment:string`
  `list[].leaveDate:string(datetime)`
  `list[].leaveReason:string`

- `MembershipStatsResponse.data`
  `pendingCount:int64`
  `activeCount:int64`
  `inactiveCount:int64`
  `suspendedCount:int64`
  `totalCount:int64`

### 5. 活动相关

- `ActivityListRequest`
  `page:int32`
  `pageSize:int32`
  `status?:int32`
  `keyword?:string`
  `startFrom?:string(datetime)`
  `startTo?:string(datetime)`
  `sortBy?:string`
  `sortOrder?:string`
  `registeredOnly?:bool`

- `ActivityListResponse.data`
  `total:int32`
  `list[].id:int64`
  `list[].title:string`
  `list[].description:string`
  `list[].coverUrl:string`
  `list[].startTime:string(datetime)`
  `list[].endTime:string(datetime)`
  `list[].location:string`
  `list[].duration:double`
  `list[].maxPeople:int32`
  `list[].currentPeople:int32`
  `list[].status:int32`
  `list[].isRegistered:bool`
  `list[].isFull:bool`

- `ActivityDetailResponse.data`
  `activity.id:int64`
  `activity.orgId:int64`
  `activity.orgName:string`
  `activity.title:string`
  `activity.description:string`
  `activity.coverUrl:string`
  `activity.startTime:string(datetime)`
  `activity.endTime:string(datetime)`
  `activity.location:string`
  `activity.address:string`
  `activity.duration:double`
  `activity.maxPeople:int32`
  `activity.currentPeople:int32`
  `activity.status:int32`
  `activity.isRegistered:bool`
  `activity.checkInStatus:int32`
  `activity.checkInTime:string(datetime)`
  `activity.checkOutStatus:int32`
  `activity.checkOutTime:string(datetime)`
  `activity.workHourStatus:int32`
  `activity.grantedHours:double`

- `GenerateAttendanceCodesResponse.data`
  `success:bool`
  `checkInCode:string`
  `checkOutCode:string`
  `attendanceCodeVersion:int64`
  `attendanceCodeUpdatedAt:string(datetime)`
  `checkInExpireAt:string(datetime)`
  `checkOutExpireAt:string(datetime)`

- `ResetAttendanceCodeResponse.data`
  `success:bool`
  `codeType:int32`
  `code:string`
  `expireAt:string(datetime)`
  `attendanceCodeVersion:int64`
  `attendanceCodeUpdatedAt:string(datetime)`

### 6. 工时与通知

- `WorkHourLogListResponse.data`
  `total:int32`
  `list[].id:int64`
  `list[].volunteerId:int64`
  `list[].activityId:int64`
  `list[].signupId:int64`
  `list[].operationType:int32`
  `list[].hoursDelta:double`
  `list[].serviceCountDelta:int64`
  `list[].beforeTotalHours:double`
  `list[].afterTotalHours:double`
  `list[].reason:string`
  `list[].operatorId:int64`
  `list[].idempotencyKey:string`
  `list[].createdAt:string(datetime)`

- `NotificationListResponse.data`
  `total:int32`
  `list[].inboxId:int64`
  `list[].notificationId:int64`
  `list[].eventType:string`
  `list[].bizType:string`
  `list[].bizId:int64`
  `list[].title:string`
  `list[].content:string`
  `list[].readStatus:int32`
  `list[].readAt:string(datetime)`
  `list[].createdAt:string(datetime)`

### 7. 统计与 AI

- `OpsDashboardSummaryResponse.data`
  `signupCount:int64`
  `approvedSignupCount:int64`
  `attendanceCount:int64`
  `attendanceRate:double`
  `grantedWorkHours:double`
  `start:string(datetime)`
  `end:string(datetime)`

- `OrgFunnelSummaryResponse.data`
  `registrationCount:int64`
  `membershipCount:int64`
  `signupCount:int64`
  `attendanceCount:int64`
  `workhourCount:int64`
  `registrationToMembershipRate:double`
  `membershipToSignupRate:double`
  `signupToAttendanceRate:double`
  `attendanceToWorkhourRate:double`
  `start:string(datetime)`
  `end:string(datetime)`

- `AssistantChatResponse.data`
  `reply:string`
  `tool_calls[].tool_name:string`
  `tool_calls[].success:bool`
  `tool_calls[].error_code:string`
  `tool_calls[].error_msg:string`
  `tool_calls[].latency_ms:int32`
  `tool_calls[].input:string`
  `tool_calls[].output:string`
  `usage.model:string`
  `usage.token_in:int32`
  `usage.token_out:int32`
  `usage.latency_ms:int32`

- `AssistantSessionMessagesResponse.data`
  `list[].id:int64`
  `list[].session_id:int64`
  `list[].seq_no:int32`
  `list[].role:int32`
  `list[].content:string`
  `list[].model:string`
  `list[].finish_reason:int32`
  `list[].token_in:int32`
  `list[].token_out:int32`
  `list[].latency_ms:int32`
  `list[].request_id:string`
  `list[].created_at:string(datetime)`

## 对账建议

1. 前端继续以页面承接为准，不要把未承接能力混入主表。
2. 联调时优先核对 `路径`、`方法`、`请求字段`、`data` 结构、`权限`、`枚举`。
3. 如果再出现模块文档和 router 不一致，优先以后端当前实际注册路由为准，再回写模块文档。
