# `internal/router/analytics.go`

## 路由

### `GET /api/analytics/org/funnel`

- 鉴权：是
- 身份：组织管理者
- 功能：查看组织转化漏斗
- 查询参数：`orgId:int64, start:string, end:string`
- 返回 `data`：`OrgFunnelSummaryResponse`

### `GET /api/analytics/org/dashboard`

- 鉴权：是
- 身份：组织管理者
- 功能：查看组织运营看板汇总
- 查询参数：`orgId:int64, start:string, end:string`
- 返回 `data`：`OpsDashboardSummaryResponse`

## 数据结构

### 请求消息

### `OrgFunnelSummaryRequest`

- `orgId:int64`
- `start:string`
- `end:string`

### `OpsDashboardSummaryRequest`

- `orgId:int64`
- `start:string`
- `end:string`

### `OrgFunnelSummaryResponse`

- `registrationCount:int64`
- `membershipCount:int64`
- `signupCount:int64`
- `attendanceCount:int64`
- `workhourCount:int64`
- `registrationToMembershipRate:double`
- `membershipToSignupRate:double`
- `signupToAttendanceRate:double`
- `attendanceToWorkhourRate:double`
- `start:string`
- `end:string`

### `OpsDashboardSummaryResponse`

- `signupCount:int64`
- `approvedSignupCount:int64`
- `attendanceCount:int64`
- `attendanceRate:double`
- `grantedWorkHours:double`
- `start:string`
- `end:string`
