# `internal/router/organization.go`

## 路由

### `POST /api/organizations/list`

- 鉴权：是
- 身份：组织管理者
- 功能：分页查询组织列表
- 请求体：`{ keyword:string, status:int32[], organizationType:string, region:string, page:int32, pageSize:int32 }`
- 返回 `data`：`OrganizationListResponse`

### `GET /api/organizations/:id`

- 鉴权：是
- 身份：组织管理者
- 功能：查看组织详情
- 路径参数：`id:int64`
- 返回 `data`：`OrganizationDetailResponse`

### `POST /api/organizations/create`

- 鉴权：是
- 身份：组织管理者
- 功能：创建组织
- 请求体：`{ name:string, organizationCode:string, contactPerson:string, contactPhone:string, email:string, address:string, organizationType:string, region:string, description:string, websiteUrl:string, logoUrl:string }`
- 返回 `data`：`{ id:int64, message:string }`

### `PUT /api/organizations/:id`

- 鉴权：是
- 身份：组织管理者
- 功能：更新组织信息
- 路径参数：`id:int64`
- 请求体：`{ name:string, organizationCode:string, contactPerson:string, contactPhone:string, address:string, organizationType:string, region:string, description:string, websiteUrl:string, logoUrl:string }`
- 返回 `data`：`{ message:string }`

### `PUT /api/organizations/account`

- 鉴权：是
- 身份：组织管理者
- 功能：更新组织管理者账号信息
- 请求体：`{ userName:string, email:string, phone:string }`
- 返回 `data`：`{}`

### `DELETE /api/organizations/:id`

- 鉴权：是
- 身份：组织管理者
- 功能：删除组织
- 路径参数：`id:int64`
- 返回 `data`：`{ message:string }`

### `POST /api/organizations/:id/disable`

- 鉴权：是
- 身份：组织管理者
- 功能：停用组织
- 路径参数：`id:int64`
- 请求体：`{ reason:string }`
- 返回 `data`：`{ message:string }`

### `POST /api/organizations/:id/enable`

- 鉴权：是
- 身份：组织管理者
- 功能：启用组织
- 路径参数：`id:int64`
- 请求体：`{ reason:string }`
- 返回 `data`：`{ message:string }`

### `POST /api/organizations/search`

- 鉴权：是
- 身份：组织管理者
- 功能：按条件搜索组织
- 请求体：`{ keyword:string, status:int32[], organizationType:string, region:string, startDate:string, endDate:string, page:int32, pageSize:int32 }`
- 返回 `data`：`OrganizationSearchResponse`

### `POST /api/organizations/bulk-delete`

- 鉴权：是
- 身份：组织管理者
- 功能：批量删除组织
- 请求体：`{ ids:int64[] }`
- 返回 `data`：`{ successCount:int32, failedCount:int32, message:string }`

### `POST /api/organizations/batch-disable`

- 鉴权：是
- 身份：组织管理者
- 功能：批量停用组织
- 请求体：`{ ids:int64[], reason:string }`
- 返回 `data`：`{ successCount:int32, failedIds:int64[], message:string }`

### `POST /api/organizations/batch-enable`

- 鉴权：是
- 身份：组织管理者
- 功能：批量启用组织
- 请求体：`{ ids:int64[], reason:string }`
- 返回 `data`：`{ successCount:int32, failedIds:int64[], message:string }`

## 数据结构

### 请求消息

### `OrganizationListRequest`

- `keyword:string`
- `status:int32[]`
- `organizationType:string`
- `region:string`
- `page:int32`
- `pageSize:int32`

### `OrganizationDetailRequest`

- `id:int64`

### `OrganizationCreateRequest`

- `name:string`
- `organizationCode:string`
- `contactPerson:string`
- `contactPhone:string`
- `email:string`
- `address:string`
- `organizationType:string`
- `region:string`
- `description:string`
- `websiteUrl:string`
- `logoUrl:string`

### `OrganizationUpdateRequest`

- `id:int64`
- `name:string`
- `organizationCode:string`
- `contactPerson:string`
- `contactPhone:string`
- `address:string`
- `organizationType:string`
- `region:string`
- `description:string`
- `websiteUrl:string`
- `logoUrl:string`

### `OrganizationAccountUpdateRequest`

- `userName:string`
- `email:string`
- `phone:string`

### `DeleteOrganizationRequest`

- `id:int64`

### `DisableOrganizationRequest`

- `id:int64`
- `reason:string`

### `EnableOrganizationRequest`

- `id:int64`
- `reason:string`

### `OrganizationSearchRequest`

- `keyword:string`
- `status:int32[]`
- `organizationType:string`
- `region:string`
- `startDate:string`
- `endDate:string`
- `page:int32`
- `pageSize:int32`

### `BulkDeleteOrganizationRequest`

- `ids:int64[]`

### `BatchDisableOrganizationRequest`

- `ids:int64[]`
- `reason:string`

### `BatchEnableOrganizationRequest`

- `ids:int64[]`
- `reason:string`

### `OrganizationListResponse` / `OrganizationSearchResponse`

- `total:int32`
- `list:OrganizationListItem[]`

### `OrganizationListItem`

- `id:int64`
- `name:string`
- `organizationCode:string`
- `contactPerson:string`
- `contactPhone:string`
- `email:string`
- `address:string`
- `status:int32`
- `organizationType:string`
- `region:string`
- `createdAt:string`

### `OrganizationDetailResponse`

- `organization:OrganizationInfo`
- `accountInfo:OrganizationAccountInfo`
- `organizationProfile:OrganizationProfileInfo`
- `organizationCertification:OrganizationCertificationInfo`

### `OrganizationInfo`

- `id:int64`
- `accountId:int64`
- `name:string`
- `organizationCode:string`
- `contactPerson:string`
- `contactPhone:string`
- `email:string`
- `address:string`
- `status:int32`
- `organizationType:string`
- `region:string`
- `description:string`
- `websiteUrl:string`
- `logoUrl:string`
- `createdAt:string`
- `updatedAt:string`

### `OrganizationAccountInfo`

- `userName:string`
- `email:string`
- `phone:string`

### `OrganizationProfileInfo`

- `name:string`
- `contactPerson:string`
- `contactPhone:string`
- `address:string`
- `description:string`
- `logoUrl:string`

### `OrganizationCertificationInfo`

- `organizationCode:string`

### `OrganizationCreateResponse`

- `id:int64`
- `message:string`

### `OrganizationUpdateResponse`

- `message:string`

### `OrganizationAccountUpdateResponse`

- 空对象 `{}`

### `DeleteOrganizationResponse`

- `message:string`

### `DisableOrganizationResponse`

- `message:string`

### `EnableOrganizationResponse`

- `message:string`

### `BulkDeleteOrganizationResponse`

- `successCount:int32`
- `failedCount:int32`
- `message:string`

### `BatchDisableOrganizationResponse`

- `successCount:int32`
- `failedIds:int64[]`
- `message:string`

### `BatchEnableOrganizationResponse`

- `successCount:int32`
- `failedIds:int64[]`
- `message:string`
