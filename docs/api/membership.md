# `internal/router/membership.go`

## 路由

### `POST /api/memberships/join`

- 鉴权：是
- 身份：志愿者
- 功能：申请加入组织
- 请求体：`{ volunteerId:int64, organizationId:int64 }`
- 返回 `data`：`{ membershipId:int64, status:int32, message:string }`

### `POST /api/memberships/leave`

- 鉴权：是
- 身份：志愿者
- 功能：退出组织
- 请求体：`{ membershipId:int64, reason:string }`
- 返回 `data`：`{ message:string }`

### `GET /api/organizations/:organizationId/members`

- 鉴权：是
- 身份：组织管理者
- 功能：查看组织成员列表
- 路径参数：`organizationId:int64`
- 查询参数：`status:int32, role:int32, keyword:string, page:int32, pageSize:int32`
- 返回 `data`：`OrganizationMembersResponse`

### `GET /api/volunteers/:volunteerId/organizations`

- 鉴权：是
- 身份：志愿者
- 功能：查看自己加入的组织
- 路径参数：`volunteerId:int64`
- 查询参数：`status:int32, page:int32, pageSize:int32`
- 返回 `data`：`VolunteerOrganizationsResponse`

### `POST /api/memberships/status/update`

- 鉴权：是
- 身份：组织管理者
- 功能：审核或变更成员状态
- 请求体：`{ membershipId:int64, status:int32, reviewComment:string }`
- 返回 `data`：`{ message:string }`

### `GET /api/memberships/stats`

- 鉴权：是
- 身份：组织管理者
- 功能：查看成员统计
- 查询参数：`organizationId:int64`
- 返回 `data`：`MembershipStatsResponse`

## 数据结构

### 请求消息

### `VolunteerJoinRequest`

- `volunteerId:int64`
- `organizationId:int64`

### `VolunteerLeaveRequest`

- `membershipId:int64`
- `reason:string`

### `OrganizationMembersRequest`

- `organizationId:int64`
- `status:int32`
- `role:int32`
- `keyword:string`
- `page:int32`
- `pageSize:int32`

### `VolunteerOrganizationsRequest`

- `volunteerId:int64`
- `status:int32`
- `page:int32`
- `pageSize:int32`

### `MemberStatusUpdateRequest`

- `membershipId:int64`
- `status:int32`
- `reviewComment:string`

### `MembershipStatsRequest`

- `organizationId:int64`

### `VolunteerJoinResponse`

- `membershipId:int64`
- `status:int32`
- `message:string`

### `VolunteerLeaveResponse`

- `message:string`

### `MemberStatusUpdateResponse`

- `message:string`

### `OrganizationMembersResponse`

- `total:int32`
- `list:MemberInfo[]`

### `VolunteerOrganizationsResponse`

- `total:int32`
- `list:OrganizationMemberInfo[]`

### `MembershipStatsResponse`

- `pendingCount:int64`
- `activeCount:int64`
- `inactiveCount:int64`
- `suspendedCount:int64`
- `totalCount:int64`

### `MemberInfo`

- `membershipId:int64`
- `volunteerId:int64`
- `volunteerName:string`
- `volunteerCode:string`
- `organizationId:int64`
- `organizationName:string`
- `status:int32`
- `role:int32`
- `position:string`
- `motivation:string`
- `expectedHours:string`
- `joinDate:string`
- `reviewDate:string`
- `reviewComment:string`
- `leaveDate:string`
- `leaveReason:string`
- `createdAt:string`
- `updatedAt:string`

### `OrganizationMemberInfo`

- `membershipId:int64`
- `organizationId:int64`
- `organizationName:string`
- `organizationCode:string`
- `status:int32`
- `role:int32`
- `position:string`
- `joinDate:string`
- `reviewDate:string`
- `reviewComment:string`
- `createdAt:string`
- `updatedAt:string`
