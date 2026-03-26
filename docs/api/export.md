# `internal/router/export.go`

## 路由

### `POST /api/admin/export/volunteers`

- 鉴权：是
- 身份：组织管理者
- 功能：导出志愿者数据
- 请求体：`{ idList:int64[], keyword:string, auditStatus:int32, status:int32 }`
- 返回：文件流

### `POST /api/admin/export/activities`

- 鉴权：是
- 身份：组织管理者
- 功能：导出活动数据
- 请求体：`{ idList:int64[], keyword:string, status:int32, startFrom:string, startTo:string }`
- 返回：文件流

### `POST /api/admin/export/ops-report`

- 鉴权：是
- 身份：组织管理者
- 功能：导出运营报表
- 请求体：`{ periodType:string, orgId:int64, start:string, end:string }`
- 返回：文件流

## 文件响应说明

- 状态码：HTTP `200`
- `Content-Disposition: attachment; filename=...`
- `Content-Type` 由服务端按导出文件类型设置
- 这 3 个接口不走统一 JSON 包装

## 数据结构

### `ExportVolunteersRequest`

- `idList:int64[]`
- `keyword:string`
- `auditStatus:int32`
- `status:int32`

### `ExportActivitiesRequest`

- `idList:int64[]`
- `keyword:string`
- `status:int32`
- `startFrom:string`
- `startTo:string`

### `ExportOpsReportRequest`

- `periodType:string`
- `orgId:int64`
- `start:string`
- `end:string`
