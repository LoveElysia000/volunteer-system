# `internal/router/work_hour.go`

## 路由

### `POST /api/work-hours/list`

- 鉴权：是
- 身份：组织管理者
- 功能：查询工时流水
- 请求体：`{ page:int32, pageSize:int32, activityId:int64, signupId:int64, operationType:int32 }`
- 返回 `data`：`WorkHourLogListResponse`

### `POST /api/work-hours/void`

- 鉴权：是
- 身份：组织管理者
- 功能：作废工时
- 请求体：`{ signupId:int64, reason:string, idempotencyKey:string }`
- 返回 `data`：`{ success:bool, workHourLogId:int64 }`

### `POST /api/work-hours/recalculate`

- 鉴权：是
- 身份：组织管理者
- 功能：重算工时
- 请求体：`{ signupId:int64, hours:double, reason:string, idempotencyKey:string }`
- 返回 `data`：`{ success:bool, workHourLogId:int64, grantedHours:double }`

## 数据结构

### 请求消息

### `WorkHourLogListRequest`

- `page:int32`
- `pageSize:int32`
- `activityId:int64`
- `signupId:int64`
- `operationType:int32`

### `VoidWorkHourRequest`

- `signupId:int64`
- `reason:string`
- `idempotencyKey:string`

### `RecalculateWorkHourRequest`

- `signupId:int64`
- `hours:double`
- `reason:string`
- `idempotencyKey:string`

### `WorkHourLogListResponse`

- `total:int32`
- `list:WorkHourLogItem[]`

### `WorkHourLogItem`

- `id:int64`
- `volunteerId:int64`
- `activityId:int64`
- `signupId:int64`
- `operationType:int32`
- `hoursDelta:double`
- `serviceCountDelta:int64`
- `beforeTotalHours:double`
- `afterTotalHours:double`
- `beforeServiceCount:int64`
- `afterServiceCount:int64`
- `workHourVersion:int64`
- `refLogId:int64`
- `reason:string`
- `operatorId:int64`
- `idempotencyKey:string`
- `createdAt:string`

### `VoidWorkHourResponse`

- `success:bool`
- `workHourLogId:int64`

### `RecalculateWorkHourResponse`

- `success:bool`
- `workHourLogId:int64`
- `grantedHours:double`
