# 通知中心前端接入说明

## 1. 结论

志愿者端和组织管理者端可以共用同一套通知接口，不需要拆成两套。

后端是根据当前登录用户的 `token` 解析出 `account_id`，再按该账号查询通知收件箱，所以：

- 志愿者登录后调用接口，拿到的是志愿者自己的通知
- 组织管理者登录后调用接口，拿到的是组织管理者自己的通知

接口层不区分端类型，前端只需要在请求时带上当前登录人的 `Authorization` 即可。

## 2. 可用接口

### 2.1 获取通知列表

- 方法：`GET`
- 路径：`/api/notifications`

请求参数放在 `query` 中：

- `page?: number`
- `pageSize?: number`
- `unreadOnly?: boolean`
- `keyword?: string`

示例：

```text
GET /api/notifications?page=1&pageSize=20&unreadOnly=true&keyword=工时
Authorization: Bearer <token>
```

### 2.2 批量标记已读

- 方法：`POST`
- 路径：`/api/notifications/read`

请求体：

```json
{
  "ids": [101, 102, 103]
}
```

注意：

- 这里的 `ids` 传的是 `inboxId`
- 不是 `notificationId`

## 3. 返回字段

`GET /api/notifications` 返回结构：

```json
{
  "total": 2,
  "list": [
    {
      "inboxId": 101,
      "notificationId": 88,
      "eventType": "work_hour_granted",
      "bizType": "activity",
      "bizId": 9001,
      "title": "工时已发放：社区清洁",
      "content": "您参与的活动《社区清洁》工时已发放，到账 3 小时。",
      "readStatus": 0,
      "readAt": "",
      "createdAt": "2026-03-29 14:30:00"
    }
  ]
}
```

字段说明：

- `inboxId`: 收件箱记录 ID，前端做已读时使用
- `notificationId`: 通知正文 ID
- `eventType`: 通知事件类型
- `bizType`: 业务类型
- `bizId`: 业务 ID，可用于跳转
- `title`: 标题
- `content`: 正文
- `readStatus`: `0` 未读，`1` 已读
- `readAt`: 已读时间
- `createdAt`: 通知创建时间

## 4. 当前已覆盖的通知事件

后端当前已经会给志愿者发送这些通知：

- `member_join`: 加入组织申请通过
- `signup_approved`: 活动报名通过
- `signup_rejected`: 活动报名驳回
- `activity_updated`: 活动信息更新
- `activity_canceled`: 活动取消
- `work_hour_granted`: 工时发放
- `work_hour_voided`: 工时作废
- `work_hour_regranted`: 工时重算

说明：

- 这些事件已经由后端业务流程自动产出
- 前端只需要把通知页接出来，不需要自己生成这类通知

## 5. 前端接入建议

### 5.1 志愿者端和组织管理者端共用

建议两端共用同一个通知 API 模块，例如：

- `listNotifications(params)`
- `markNotificationsRead(ids)`

不建议按角色拆成：

- `listVolunteerNotifications`
- `listOrgNotifications`

因为接口本身没有角色区分，拆开只会增加维护成本。

### 5.2 两端页面建议最少支持

- 通知列表展示
- 未读筛选
- 关键词搜索
- 单条/批量已读
- 顶部未读角标或红点
- 点击通知后跳转到对应业务页

### 5.3 建议的跳转映射

前端建议根据 `eventType` 和 `bizType` 做业务跳转。

可参考下面的映射：

- `member_join` -> 我的组织 / 组织详情
- `signup_approved` -> 我的活动 / 报名详情
- `signup_rejected` -> 我的活动 / 报名详情
- `activity_updated` -> 活动详情
- `activity_canceled` -> 活动详情
- `work_hour_granted` -> 我的工时 / 活动记录
- `work_hour_voided` -> 我的工时 / 活动记录
- `work_hour_regranted` -> 我的工时 / 活动记录

说明：

- 跳转路径由前端自己决定
- 后端只返回业务标识，不负责页面路由

## 6. 联调注意事项

### 6.1 请求必须带 token

通知接口完全依赖当前登录账号，所以前端每次请求都必须带：

```text
Authorization: Bearer <token>
```

### 6.2 已读要传 inboxId

很多前端在这里容易传错。

`POST /api/notifications/read` 的 `ids` 应该传：

- `list[].inboxId`

而不是：

- `list[].notificationId`

### 6.3 搜索是服务端搜索

`keyword` 会在后端搜索以下字段：

- `title`
- `content`
- `bizType`
- `eventType`

前端不要只在当前页做本地过滤，否则分页后结果会不准确。

## 7. 直接插数据库能不能展示

可以，但必须插对两张表，不是只插一张表就行。

通知列表依赖：

- `notifications`
- `notification_inbox`

查询逻辑是收件箱表关联通知正文表。

### 7.1 只插 `notifications`

不能展示。

因为前端列表查的是当前账号的收件箱，没有收件箱记录就不会出现在列表里。

### 7.2 只插 `notification_inbox`

也不完整，不建议。

因为收件箱记录需要关联有效的 `notifications.id`，否则正文信息不完整。

### 7.3 正确做法

至少需要：

1. 在 `notifications` 插入一条通知正文
2. 在 `notification_inbox` 插入一条或多条收件箱记录

关键字段要求：

- `notification_inbox.notification_id` 必须关联到有效的 `notifications.id`
- `notification_inbox.receiver_id` 必须是目标用户的 `sys_accounts.id`
- `notification_inbox.inbox_status = 1`
- 如果希望显示为未读，`read_status = 0`
- 同一个 `receiver_id + notification_id` 不能重复

### 7.4 适用场景

直接插库适合：

- 前端联调
- 测试环境造数据
- 临时验证展示效果

不建议作为正式业务入口长期使用，因为会绕开：

- 后端收件人解析
- 幂等控制
- 业务审计
- 后续扩展的权限校验

## 8. 是否可以增加“手动创建通知”接口

可以，而且是合理的。

如果后续有这些场景，增加手动创建通知接口是有价值的：

- 运营手动发站内公告
- 管理员定向发通知
- 测试或客服补发通知
- 某些不挂在现有业务事件链上的临时通知

### 8.1 建议用途

建议把这个接口定位成：

- 后台运营/管理能力
- 补发能力
- 通知公告能力

不建议让普通志愿者调用。

### 8.2 建议最小能力

如果新增接口，建议至少支持这些字段：

- `title`
- `content`
- `receiverIds`
- `sourceOrgId`
- `bizType`
- `bizId`
- `eventType`

可以先允许前端传一个固定的人工事件类型，例如：

- `manual_notice`

### 8.3 建议权限控制

如果做这个接口，建议至少限制为：

- 超级管理员可发全局通知
- 组织管理者只能给自己组织内相关用户发通知
- 志愿者不能调用该接口

### 8.4 建议实现方式

建议新增接口时，仍然复用现有通知落库逻辑，而不是单独写一套插库代码。

这样可以保持：

- 收件箱结构一致
- 已读逻辑一致
- 后续扩展邮件/推送时更容易复用

## 9. 推荐落地顺序

建议按这个顺序推进：

1. 前端先接现有通知列表和已读能力
2. 志愿者端、组织管理者端共用同一套 API 封装
3. 两端分别补页面入口、红点和跳转
4. 联调时可先用测试数据或直接插库验证
5. 如有运营诉求，再补“手动创建通知”接口
