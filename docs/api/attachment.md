# `internal/router/attachment.go`

## 路由

### `POST /api/admin/import/volunteers`

- 鉴权：是
- 身份：组织管理者
- 功能：导入志愿者数据
- 请求类型：`multipart/form-data`
- 文件字段：`file`
- 返回 `data`：`ImportResultResponse`

### `POST /api/admin/import/activities`

- 鉴权：是
- 身份：组织管理者
- 功能：导入活动数据
- 请求类型：`multipart/form-data`
- 文件字段：`file`
- 返回 `data`：`ImportResultResponse`

## 数据结构

### 请求消息

### `ImportVolunteersRequest`

- `multipart/form-data`
- `file:binary`

### `ImportActivitiesRequest`

- `multipart/form-data`
- `file:binary`

### `ImportResultResponse`

- `totalCount:int32`
- `successCount:int32`
- `failedCount:int32`
- `errorFileName:string`
- `errorFileContentType:string`
- `errorFileContent:bytes`
- `failures:ImportFailureItem[]`

### `ImportFailureItem`

- `rowNumber:int32`
- `reason:string`
- `raw:string`
