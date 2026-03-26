# `internal/router/notification.go`

## 路由

### `GET /api/notifications`

- 鉴权：是
- 身份：志愿者/组织管理者
- 功能：查询当前账号通知列表
- 查询参数：`page:int32, pageSize:int32, unreadOnly:bool`
- 返回 `data`：`NotificationListResponse`

### `POST /api/notifications/read`

- 鉴权：是
- 身份：志愿者/组织管理者
- 功能：批量标记通知已读
- 请求体：`{ ids:int64[] }`
- 返回 `data`：`{ updated:int32 }`

## 数据结构

### 请求消息

### `NotificationListRequest`

- `page:int32`
- `pageSize:int32`
- `unreadOnly:bool`

### `NotificationReadRequest`

- `ids:int64[]`

### `NotificationListResponse`

- `total:int32`
- `list:NotificationItem[]`

### `NotificationItem`

- `inboxId:int64`
- `notificationId:int64`
- `eventType:string`
- `bizType:string`
- `bizId:int64`
- `title:string`
- `content:string`
- `readStatus:int32`
- `readAt:string`
- `createdAt:string`

### `NotificationReadResponse`

- `updated:int32`
