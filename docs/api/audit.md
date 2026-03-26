# `internal/router/audit.go`

## 路由

### `POST /api/audits/pending`

- 鉴权：是
- 身份：组织管理者
- 功能：查询待审核列表
- 请求体：`{ targetTypes:int32[], status:int32[], keyword:string, page:int32, pageSize:int32, createdFrom:string, createdTo:string, slaHours:int32 }`
- 返回 `data`：`PendingAuditListResponse`

### `POST /api/audits/approval`

- 鉴权：是
- 身份：组织管理者
- 功能：执行审核通过
- 请求体：`{ id:int64, reason:string }`
- 返回 `data`：`{}`

### `POST /api/audits/rejection`

- 鉴权：是
- 身份：组织管理者
- 功能：执行审核驳回
- 请求体：`{ id:int64, reason:string }`
- 返回 `data`：`{}`

### `POST /api/audits/batch-decision`

- 鉴权：是
- 身份：组织管理者
- 功能：批量审核
- 请求体：`{ ids:int64[], action:int32, reason:string }`
- 返回 `data`：`{ successCount:int32, failedIds:int64[] }`

### `GET /api/audits/records/:id`

- 鉴权：是
- 身份：组织管理者
- 功能：查看审核记录详情
- 路径参数：`id:int64`
- 返回 `data`：`{ record:AuditRecordDetail }`

## 数据结构

### 请求消息

### `PendingAuditListRequest`

- `targetTypes:int32[]`
- `status:int32[]`
- `keyword:string`
- `page:int32`
- `pageSize:int32`
- `createdFrom:string`
- `createdTo:string`
- `slaHours:int32`

### `AuditApprovalRequest`

- `id:int64`
- `reason:string`

### `AuditRejectionRequest`

- `id:int64`
- `reason:string`

### `AuditBatchDecisionRequest`

- `ids:int64[]`
- `action:int32`
- `reason:string`

### `AuditRecordDetailRequest`

- `id:int64`

### `PendingAuditListResponse`

- `total:int32`
- `list:PendingAuditItem[]`

### `PendingAuditItem`

- `id:int64`
- `targetType:int32`
- `targetId:int64`
- `title:string`
- `subTitle:string`
- `creatorId:int64`
- `createdAt:string`
- `isOverdue:bool`

### `AuditRecordDetail`

- `id:int64`
- `targetType:int32`
- `targetId:int64`
- `auditorId:int64`
- `status:int32`
- `oldContent:string`
- `newContent:string`
- `auditResult:int32`
- `rejectReason:string`
- `auditTime:string`
- `createdAt:string`

### `AuditApprovalResponse`

- 空对象 `{}`

### `AuditRejectionResponse`

- 空对象 `{}`

### `AuditBatchDecisionResponse`

- `successCount:int32`
- `failedIds:int64[]`

### `AuditRecordDetailResponse`

- `record:AuditRecordDetail`
