# `internal/router/activities.go`

## 路由

### `POST /api/activities`

- 鉴权：是
- 身份：志愿者/组织管理者
- 功能：分页查询活动列表
- 请求体：`{ page:int32, pageSize:int32, status:int32, keyword:string, startFrom:string, startTo:string, sortBy:string, sortOrder:string }`
- 返回 `data`：`ActivityListResponse`

### `POST /api/activities/signup`

- 鉴权：是
- 身份：志愿者
- 功能：报名活动
- 请求体：`{ activityId:int64 }`
- 返回 `data`：`{ success:bool }`

### `POST /api/activities/cancel`

- 鉴权：是
- 身份：志愿者
- 功能：取消报名
- 请求体：`{ activityId:int64 }`
- 返回 `data`：`{ success:bool }`

### `GET /api/activities/:id`

- 鉴权：是
- 身份：志愿者/组织管理者
- 功能：查看活动详情
- 路径参数：`id:int64`
- 返回 `data`：`{ activity:ActivityInfo }`

### `POST /api/activities/my`

- 鉴权：是
- 身份：志愿者
- 功能：查询我的活动列表
- 请求体：`{ page:int32, pageSize:int32, status:int32 }`
- 返回 `data`：`MyActivitiesResponse`

### `POST /api/activities/checkin`

- 鉴权：是
- 身份：志愿者
- 功能：活动签到
- 请求体：`{ activityId:int64, checkInCode:string }`
- 返回 `data`：`{ success:bool, checkInTime:string }`

### `POST /api/activities/checkout`

- 鉴权：是
- 身份：志愿者
- 功能：活动签退并触发工时结算
- 请求体：`{ activityId:int64, checkOutCode:string }`
- 返回 `data`：`{ success:bool, checkOutTime:string, grantedHours:double }`

### `POST /api/activities/create`

- 鉴权：是
- 身份：组织管理者
- 功能：创建活动
- 请求体：`{ orgId:int64, title:string, description:string, coverUrl:string, startTime:string, endTime:string, location:string, address:string, duration:double, maxPeople:int32 }`
- 返回 `data`：`{ id:int64, message:string }`

### `PUT /api/activities/:id`

- 鉴权：是
- 身份：组织管理者
- 功能：更新活动
- 路径参数：`id:int64`
- 请求体：`{ title:string, description:string, coverUrl:string, startTime:string, endTime:string, location:string, address:string, duration:double, maxPeople:int32 }`
- 返回 `data`：`{ message:string }`

### `DELETE /api/activities/:id`

- 鉴权：是
- 身份：组织管理者
- 功能：删除活动
- 路径参数：`id:int64`
- 返回 `data`：`{ message:string }`

### `POST /api/activities/cancel/:id`

- 鉴权：是
- 身份：组织管理者
- 功能：取消活动
- 路径参数：`id:int64`
- 请求体：`{ reason:string }`
- 返回 `data`：`{ message:string }`

### `POST /api/activities/finish/:id`

- 鉴权：是
- 身份：组织管理者
- 功能：完结活动
- 路径参数：`id:int64`
- 返回 `data`：`{ message:string }`

### `POST /api/activities/attendance-codes/generate/:id`

- 鉴权：是
- 身份：组织管理者
- 功能：生成签到码和签退码
- 路径参数：`id:int64`
- 请求体：`{ checkInValidMinutes:int32, checkOutValidMinutes:int32 }`
- 返回 `data`：`GenerateAttendanceCodesResponse`

### `POST /api/activities/attendance-codes/reset/:id`

- 鉴权：是
- 身份：组织管理者
- 功能：重置签到码或签退码
- 路径参数：`id:int64`
- 请求体：`{ codeType:int32, validMinutes:int32 }`
- 返回 `data`：`ResetAttendanceCodeResponse`

### `GET /api/activities/attendance-codes/:id`

- 鉴权：是
- 身份：组织管理者
- 功能：查看当前签到码和签退码
- 路径参数：`id:int64`
- 返回 `data`：`GetActivityAttendanceCodesResponse`

### `POST /api/activities/supplement-attendance`

- 鉴权：是
- 身份：组织管理者
- 功能：补录签到签退并补发工时
- 请求体：`{ activityId:int64, volunteerId:int64, checkInTime:string, checkOutTime:string, reason:string }`
- 返回 `data`：`ActivitySupplementAttendanceResponse`

## 数据结构

### 请求消息

### `ActivityListRequest`

- `page:int32`
- `pageSize:int32`
- `status:int32`
- `keyword:string`
- `startFrom:string`
- `startTo:string`
- `sortBy:string`
- `sortOrder:string`

### `ActivitySignupRequest`

- `activityId:int64`

### `ActivityCancelRequest`

- `activityId:int64`

### `ActivityDetailRequest`

- `id:int64`

### `MyActivitiesRequest`

- `page:int32`
- `pageSize:int32`
- `status:int32`

### `ActivityCheckInRequest`

- `activityId:int64`
- `checkInCode:string`

### `ActivityCheckOutRequest`

- `activityId:int64`
- `checkOutCode:string`

### `CreateActivityRequest`

- `orgId:int64`
- `title:string`
- `description:string`
- `coverUrl:string`
- `startTime:string`
- `endTime:string`
- `location:string`
- `address:string`
- `duration:double`
- `maxPeople:int32`

### `UpdateActivityRequest`

- `id:int64`
- `title:string`
- `description:string`
- `coverUrl:string`
- `startTime:string`
- `endTime:string`
- `location:string`
- `address:string`
- `duration:double`
- `maxPeople:int32`

### `DeleteActivityRequest`

- `id:int64`

### `CancelActivityRequest`

- `id:int64`
- `reason:string`

### `FinishActivityRequest`

- `id:int64`

### `GenerateAttendanceCodesRequest`

- `id:int64`
- `checkInValidMinutes:int32`
- `checkOutValidMinutes:int32`

### `ResetAttendanceCodeRequest`

- `id:int64`
- `codeType:int32`
- `validMinutes:int32`

### `GetActivityAttendanceCodesRequest`

- `id:int64`

### `ActivitySupplementAttendanceRequest`

- `activityId:int64`
- `volunteerId:int64`
- `checkInTime:string`
- `checkOutTime:string`
- `reason:string`

### `ActivityListResponse`

- `total:int32`
- `list:ActivityItem[]`

### `ActivityItem`

- `id:int64`
- `title:string`
- `description:string`
- `coverUrl:string`
- `startTime:string`
- `endTime:string`
- `location:string`
- `duration:double`
- `maxPeople:int32`
- `currentPeople:int32`
- `status:int32`
- `isRegistered:bool`
- `isFull:bool`

### `ActivityInfo`

- `id:int64`
- `orgId:int64`
- `orgName:string`
- `title:string`
- `description:string`
- `coverUrl:string`
- `startTime:string`
- `endTime:string`
- `location:string`
- `address:string`
- `duration:double`
- `maxPeople:int32`
- `currentPeople:int32`
- `status:int32`
- `isRegistered:bool`
- `createdAt:string`
- `checkInStatus:int32`
- `checkInTime:string`
- `checkOutStatus:int32`
- `checkOutTime:string`
- `workHourStatus:int32`
- `grantedHours:double`

### `MyActivitiesResponse`

- `total:int32`
- `list:MyActivityItem[]`

### `MyActivityItem`

- `id:int64`
- `orgId:int64`
- `orgName:string`
- `title:string`
- `description:string`
- `coverUrl:string`
- `startTime:string`
- `endTime:string`
- `location:string`
- `duration:double`
- `maxPeople:int32`
- `currentPeople:int32`
- `status:int32`
- `signupTime:string`
- `checkInStatus:int32`
- `checkInTime:string`
- `checkOutStatus:int32`
- `checkOutTime:string`
- `workHourStatus:int32`
- `grantedHours:double`
- `signupStatus:int32`
- `auditReason:string`

### `GenerateAttendanceCodesResponse`

- `success:bool`
- `checkInCode:string`
- `checkOutCode:string`
- `attendanceCodeVersion:int64`
- `attendanceCodeUpdatedAt:string`
- `checkInExpireAt:string`
- `checkOutExpireAt:string`

### `ResetAttendanceCodeResponse`

- `success:bool`
- `codeType:int32`
- `code:string`
- `expireAt:string`
- `attendanceCodeVersion:int64`
- `attendanceCodeUpdatedAt:string`

### `GetActivityAttendanceCodesResponse`

- `success:bool`
- `checkInCode:string`
- `checkOutCode:string`
- `checkInExpireAt:string`
- `checkOutExpireAt:string`
- `attendanceCodeVersion:int64`
- `attendanceCodeUpdatedAt:string`

### `ActivitySupplementAttendanceResponse`

- `success:bool`
- `checkInTime:string`
- `checkOutTime:string`
- `grantedHours:double`

### `ActivitySignupResponse`

- `success:bool`

### `ActivityCancelResponse`

- `success:bool`

### `ActivityCheckInResponse`

- `success:bool`
- `checkInTime:string`

### `ActivityCheckOutResponse`

- `success:bool`
- `checkOutTime:string`
- `grantedHours:double`

### `CreateActivityResponse`

- `id:int64`
- `message:string`

### `UpdateActivityResponse`

- `message:string`

### `DeleteActivityResponse`

- `message:string`

### `CancelActivityResponse`

- `message:string`

### `FinishActivityResponse`

- `message:string`
