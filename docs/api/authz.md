# `internal/router/authz.go`

## 路由

### `GET /api/authz/roles`

- 鉴权：是
- 身份：组织管理者
- 功能：查询角色列表
- 查询参数：`keyword:string, includeDisabled:bool, page:int32, pageSize:int32`
- 返回 `data`：`RoleListResponse`

### `POST /api/authz/roles`

- 鉴权：是
- 身份：组织管理者
- 功能：创建角色
- 请求体：`{ roleCode:string, roleName:string, description:string }`
- 返回 `data`：`{ id:int64 }`

### `PUT /api/authz/roles/:id`

- 鉴权：是
- 身份：组织管理者
- 功能：更新角色
- 路径参数：`id:int64`
- 请求体：`{ roleName:string, description:string }`
- 返回 `data`：`{ message:string }`

### `POST /api/authz/roles/:id/status`

- 鉴权：是
- 身份：组织管理者
- 功能：启停角色
- 路径参数：`id:int64`
- 请求体：`{ status:int32 }`
- 返回 `data`：`{ message:string }`

### `GET /api/authz/permissions`

- 鉴权：是
- 身份：组织管理者
- 功能：查询权限列表
- 查询参数：`keyword:string, onlyEnabled:bool`
- 返回 `data`：`PermissionListResponse`

### `GET /api/authz/roles/:roleId/permissions`

- 鉴权：是
- 身份：组织管理者
- 功能：查看角色权限
- 路径参数：`roleId:int64`
- 返回 `data`：`RolePermissionsResponse`

### `POST /api/authz/roles/:roleId/permissions/set`

- 鉴权：是
- 身份：组织管理者
- 功能：设置角色权限
- 路径参数：`roleId:int64`
- 请求体：`{ permissionIds:int64[] }`
- 返回 `data`：`{ message:string }`

### `GET /api/authz/grants`

- 鉴权：是
- 身份：组织管理者
- 功能：查询账号授权关系
- 查询参数：`accountId:int64, scopeType:string, scopeId:int64, onlyActive:bool, page:int32, pageSize:int32`
- 返回 `data`：`AccountRoleBindingListResponse`

### `POST /api/authz/grants`

- 鉴权：是
- 身份：组织管理者
- 功能：给账号授予角色
- 请求体：`{ accountId:int64, roleId:int64, scopeType:string, scopeId:int64, expiresAt:string, remark:string }`
- 返回 `data`：`{ message:string }`

### `POST /api/authz/grants/:bindingId/revoke`

- 鉴权：是
- 身份：组织管理者
- 功能：撤销账号角色
- 路径参数：`bindingId:int64`
- 请求体：`{ remark:string }`
- 返回 `data`：`{ message:string }`

### `GET /api/authz/me`

- 鉴权：是
- 身份：志愿者/组织管理者
- 功能：查询当前账号的角色和权限
- 请求参数：无
- 返回 `data`：`MyAuthorizationResponse`

## 数据结构

### 请求消息

### `RoleListRequest`

- `keyword:string`
- `includeDisabled:bool`
- `page:int32`
- `pageSize:int32`

### `RoleCreateRequest`

- `roleCode:string`
- `roleName:string`
- `description:string`

### `RoleUpdateRequest`

- `id:int64`
- `roleName:string`
- `description:string`

### `RoleStatusUpdateRequest`

- `id:int64`
- `status:int32`

### `PermissionListRequest`

- `keyword:string`
- `onlyEnabled:bool`

### `RolePermissionsRequest`

- `roleId:int64`

### `RolePermissionsSetRequest`

- `roleId:int64`
- `permissionIds:int64[]`

### `AccountRoleBindingListRequest`

- `accountId:int64`
- `scopeType:string`
- `scopeId:int64`
- `onlyActive:bool`
- `page:int32`
- `pageSize:int32`

### `AccountRoleGrantRequest`

- `accountId:int64`
- `roleId:int64`
- `scopeType:string`
- `scopeId:int64`
- `expiresAt:string`
- `remark:string`

### `AccountRoleRevokeRequest`

- `bindingId:int64`
- `remark:string`

### `MyAuthorizationRequest`

- 空对象 `{}`

### `RoleListResponse`

- `total:int32`
- `list:RoleInfo[]`

### `RoleInfo`

- `id:int64`
- `roleCode:string`
- `roleName:string`
- `description:string`
- `status:int32`
- `createdAt:string`
- `updatedAt:string`

### `PermissionListResponse`

- `list:PermissionInfo[]`

### `PermissionInfo`

- `id:int64`
- `resource:string`
- `action:string`
- `description:string`
- `status:int32`

### `RoleCreateResponse`

- `id:int64`

### `RoleUpdateResponse`

- `message:string`

### `RoleStatusUpdateResponse`

- `message:string`

### `RolePermissionsResponse`

- `roleId:int64`
- `permissions:RolePermissionItem[]`

### `RolePermissionItem`

- `permissionId:int64`
- `resource:string`
- `action:string`
- `description:string`

### `RolePermissionsSetResponse`

- `message:string`

### `AccountRoleBindingListResponse`

- `total:int32`
- `list:AccountRoleBindingInfo[]`

### `AccountRoleBindingInfo`

- `bindingId:int64`
- `accountId:int64`
- `roleId:int64`
- `roleCode:string`
- `roleName:string`
- `scopeType:string`
- `scopeId:int64`
- `status:int32`
- `grantedBy:int64`
- `expiresAt:string`
- `createdAt:string`
- `updatedAt:string`

### `AccountRoleGrantResponse`

- `message:string`

### `AccountRoleRevokeResponse`

- `message:string`

### `MyAuthorizationResponse`

- `accountId:int64`
- `roles:AccountRoleBindingInfo[]`
- `permissions:MyPermissionScope[]`

### `MyPermissionScope`

- `resource:string`
- `action:string`
- `scopeType:string`
- `scopeId:int64`
