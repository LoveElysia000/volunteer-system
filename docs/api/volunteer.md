# `internal/router/volunteer.go`

## 路由

### `POST /api/volunteers/list`

- 鉴权：是
- 身份：组织管理者
- 功能：分页查询志愿者列表
- 请求体：`{ keyword:string, page:int32, pageSize:int32 }`
- 返回 `data`：`VolunteerListResponse`

### `GET /api/volunteers/detail/:id`

- 鉴权：是
- 身份：组织管理者
- 功能：查看单个志愿者详情
- 路径参数：`id:int64`
- 返回 `data`：`{ volunteer:VolunteerInfo }`

### `GET /api/volunteers/my/profile/:id`

- 鉴权：是
- 身份：志愿者
- 功能：查看自己的完整资料
- 路径参数：`id:int64`
- 返回 `data`：`MyProfileResponse`

### `GET /api/volunteers/home/summary`

- 鉴权：是
- 身份：志愿者
- 功能：获取志愿者首页摘要数据
- 请求参数：无
- 返回 `data`：`VolunteerHomeSummaryResponse`

### `PUT /api/volunteers/account`

- 鉴权：是
- 身份：志愿者
- 功能：更新自己的账号信息
- 请求体：`{ userName:string, email:string, phone:string }`
- 返回 `data`：`{}`

### `PUT /api/volunteers/:id`

- 鉴权：是
- 身份：志愿者
- 功能：更新自己的资料信息
- 路径参数：`id:int64`
- 请求体：`{ gender:int32, birthday:string, avatarUrl:string, introduction:string }`
- 返回 `data`：`{}`

### `POST /api/volunteers/real-name/submit`

- 鉴权：是
- 身份：志愿者
- 功能：提交实名认证申请
- 请求体：`{ realName:string, idCard:string }`
- 返回 `data`：`{ auditId:int64, status:int32 }`

## 数据结构

### 请求消息

### `VolunteerListRequest`

- `keyword:string`
- `page:int32`
- `pageSize:int32`

### `VolunteerDetailRequest`

- `id:int64`

### `MyProfileRequest`

- `id:int64`

### `VolunteerHomeSummaryRequest`

- 空对象 `{}`

### `VolunteerAccountUpdateRequest`

- `userName:string`
- `email:string`
- `phone:string`

### `VolunteerUpdateRequest`

- `volunteerId:int64`
- `gender:int32`
- `birthday:string`
- `avatarUrl:string`
- `introduction:string`

### `VolunteerRealNameSubmitRequest`

- `realName:string`
- `idCard:string`

### `VolunteerListResponse`

- `total:int32`
- `list:VolunteerListItem[]`

### `VolunteerListItem`

- `id:int64`
- `accountId:int64`
- `realName:string`
- `gender:int32`
- `avatarUrl:string`
- `totalHours:double`
- `serviceCount:int32`
- `creditScore:int32`
- `auditStatus:int32`
- `createdAt:string`
- `updatedAt:string`
- `status:int32`

### `MyProfileResponse`

- `volunteer:VolunteerInfo`
- `accountInfo:VolunteerAccountInfo`
- `profile:VolunteerProfileInfo`
- `verification:VolunteerVerificationInfo`

### `VolunteerHomeSummaryResponse`

- `nickname:string`
- `level:int32`
- `stats:VolunteerHomeSummaryStats`
- `monthlyGrowth:double`
- `needHoursToNextLevel:double`

### `VolunteerHomeSummaryStats`

- `points:int32`
- `hours:double`
- `activityCount:int32`

### `VolunteerInfo`

- `id:int64`
- `accountId:int64`
- `realName:string`
- `gender:int32`
- `birthday:string`
- `idCard:string`
- `avatarUrl:string`
- `introduction:string`
- `totalHours:double`
- `serviceCount:int32`
- `creditScore:int32`
- `auditStatus:int32`
- `createdAt:string`
- `updatedAt:string`
- `status:int32`

### `VolunteerAccountInfo`

- `userName:string`
- `email:string`
- `phone:string`

### `VolunteerProfileInfo`

- `gender:int32`
- `birthday:string`
- `avatarUrl:string`
- `introduction:string`

### `VolunteerVerificationInfo`

- `realName:string`
- `idCard:string`
- `auditStatus:int32`

### `VolunteerUpdateResponse`

- 空对象 `{}`

### `VolunteerAccountUpdateResponse`

- 空对象 `{}`

### `VolunteerRealNameSubmitResponse`

- `auditId:int64`
- `status:int32`
