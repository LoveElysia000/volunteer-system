# 通知中心业务方案（精简版）

更新时间：2026-02-28

## 1. 业务目标与范围

目标：实现站内通知，覆盖当前 3 个核心业务场景。  
范围：仅站内通知，不引入 MQ，不影响主业务成功返回。

## 2. 业务场景与规则

### 2.1 触发场景

1. 组织创建活动：通知该组织下有效志愿者。  
2. 组织修改活动：通知报名该活动且未取消的志愿者。  
3. 志愿者成功加入组织：通知该志愿者本人。

### 2.2 通知约束

1. 通知是异步 best-effort，发送失败不回滚主业务。  
2. 需要幂等，避免重复通知。  
3. 支持按 `bizType + bizId` 追踪通知来源。  
4. 志愿者退出组织后，相关通知归档，不做物理删除。

## 3. 数据库设计

本期主结构保留两张核心表：`notifications`、`notification_inbox`。  
`notification_outbox` 作为可选增强表，按可靠投递需求启用。

状态枚举：

1. `read_status`：`0=未读` `1=已读`
2. `inbox_status`：`1=normal` `2=archived` `3=user_deleted`

### 3.1 `notifications`（通知内容主表）

用途：

1. 存通知正文与业务来源。  
2. 一条内容可分发给多个接收人。

```sql
CREATE TABLE IF NOT EXISTS `notifications` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `event_type` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '事件类型: activity.created/activity.updated/member.join.approved',
  `biz_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '业务类型: activity/membership/organization',
  `biz_id` BIGINT NOT NULL DEFAULT 0 COMMENT '业务ID',
  `source_org_id` BIGINT NOT NULL DEFAULT 0 COMMENT '来源组织ID(用于退出组织后归档)',
  `sender_id` BIGINT NOT NULL DEFAULT 0 COMMENT '发送方账号ID',
  `title` VARCHAR(200) NOT NULL DEFAULT '' COMMENT '通知标题',
  `content` VARCHAR(2000) NOT NULL DEFAULT '' COMMENT '通知正文',
  `payload` JSON NULL COMMENT '扩展字段(JSON)',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_biz` (`biz_type`, `biz_id`),
  KEY `idx_source_org_created` (`source_org_id`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通知内容主表';
```

### 3.2 `notification_inbox`（用户收件箱）

用途：

1. 记录用户维度收件、已读、归档状态。  
2. 支撑列表分页与未读筛选。  
3. 支撑退出组织后按组织归档。

```sql
CREATE TABLE IF NOT EXISTS `notification_inbox` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `notification_id` BIGINT NOT NULL COMMENT '通知内容ID',
  `receiver_id` BIGINT NOT NULL COMMENT '接收人账号ID(sys_accounts.id)',
  `source_org_id` BIGINT NOT NULL DEFAULT 0 COMMENT '来源组织ID(冗余,便于归档更新)',
  `read_status` TINYINT NOT NULL DEFAULT 0 COMMENT '读取状态: 0-未读, 1-已读',
  `read_at` DATETIME NULL COMMENT '读取时间',
  `inbox_status` TINYINT NOT NULL DEFAULT 1 COMMENT '收件箱状态: 1-normal, 2-archived, 3-user_deleted',
  `archived_reason` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '归档原因: left_org/system_clean/user_action',
  `archived_at` DATETIME NULL COMMENT '归档时间',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '入箱时间',
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_receiver_notification` (`receiver_id`, `notification_id`),
  KEY `idx_inbox_list` (`receiver_id`, `inbox_status`, `created_at`),
  KEY `idx_receiver_status_read_created` (`receiver_id`, `inbox_status`, `read_status`, `created_at`),
  KEY `idx_receiver_source_org` (`receiver_id`, `source_org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户通知收件箱';
```

### 3.3 `notification_outbox`（可选增强）

用途：

1. 主业务成功后记录通知事件，避免进程抖动造成消息丢失。  
2. 支持失败重试与死信排查。  
3. 为后续扩展任务补偿提供数据基础。

状态枚举：

1. `outbox_status`：`0=pending` `1=processing` `2=success` `3=retry_wait` `4=dead`

```sql
CREATE TABLE IF NOT EXISTS `notification_outbox` (
  `id` BIGINT NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `event_key` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '幂等键',
  `event_type` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '事件类型',
  `biz_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '业务类型',
  `biz_id` BIGINT NOT NULL DEFAULT 0 COMMENT '业务ID',
  `payload` JSON NOT NULL COMMENT '事件负载',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态: 0-pending,1-processing,2-success,3-retry_wait,4-dead',
  `retry_count` INT NOT NULL DEFAULT 0 COMMENT '重试次数',
  `next_retry_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '下次重试时间',
  `last_error` VARCHAR(500) NOT NULL DEFAULT '' COMMENT '最后错误',
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `processed_at` DATETIME NULL COMMENT '处理完成时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_event_key` (`event_key`),
  KEY `idx_status_retry` (`status`, `next_retry_at`),
  KEY `idx_biz` (`biz_type`, `biz_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='通知事件投递箱';
```

### 3.4 关键 SQL 场景

1. 通知列表：
`WHERE receiver_id=? AND inbox_status=1 ORDER BY created_at DESC LIMIT ?,?`
2. 仅未读：
`WHERE receiver_id=? AND inbox_status=1 AND read_status=0`
3. 批量已读：
`UPDATE notification_inbox SET read_status=1, read_at=NOW() WHERE receiver_id=? AND id IN (?) AND read_status=0`
4. 退出组织后归档：
`UPDATE notification_inbox SET inbox_status=2, archived_reason='left_org', archived_at=NOW() WHERE receiver_id=? AND source_org_id=? AND inbox_status=1`
5. （增强）拉取待消费 outbox：
`WHERE status IN (0,3) AND next_retry_at<=NOW() ORDER BY id ASC LIMIT ?`

## 4. 业务处理流程（channel 为主，可配 outbox 补偿）

### 4.1 结构定义

```go
type NotificationEvent struct {
    EventType   string         // activity.created / activity.updated / member.join.approved
    BizType     string         // activity / membership / organization
    BizID       int64          // 业务ID
    SourceOrgID int64          // 来源组织ID
    ActorID     int64          // 操作人账号ID
    CreatedAt   time.Time
    Payload     map[string]any // 活动名、活动时间、变更说明等
    DedupeKey   string         // 幂等键
}

type NotificationDispatcher struct {
    ch      chan NotificationEvent
    workers int
}
```

### 4.2 事件发布（主业务成功后）

```go
func (d *NotificationDispatcher) Publish(evt NotificationEvent) {
    select {
    case d.ch <- evt:
    default:
        // 队列满时丢弃并记录日志，避免拖慢主业务
    }
}
```

发布时机：活动创建/修改成功、成员入组审核通过成功之后。

事件发布入口建议：

1. 活动创建成功：`internal/service/activities.go` 创建事务提交后发布。  
2. 活动修改成功：`internal/service/activities.go` 更新事务提交后发布。  
3. 成员入组通过：`internal/service/membership.go`（或审核服务）状态更新成功后发布。

### 4.3 事件消费

```go
func (d *NotificationDispatcher) Start(ctx context.Context, svc *NotificationService) {
    for i := 0; i < d.workers; i++ {
        go func() {
            for {
                select {
                case <-ctx.Done():
                    return
                case evt := <-d.ch:
                    _ = svc.HandleEvent(ctx, evt)
                }
            }
        }()
    }
}
```

`HandleEvent` 处理步骤：

1. 根据 `EventType` 计算接收人。  
2. 渲染 `title/content`。  
3. 写入 `notifications` 一条。  
4. 批量写入 `notification_inbox` 多条（每批建议 500-1000 条）。  
5. 单次扇出量过大（如 >50000）时，按分页拆分子任务处理，避免长事务。  
6. 失败只记日志，不回传主流程。

接收人计算规则细化：

1. `activity.created`：按 `source_org_id` 查询组织在册且有效的志愿者账号。  
2. `activity.updated`：按 `activity_id` 查询报名记录中“未取消、未删除”的志愿者账号。  
3. `member.join.approved`：仅取该成员本人账号。

建议统一做一次接收人去重：

1. 使用 `util.UniquePositiveInt64` 去重并过滤非法账号 ID。  
2. 去重后为空时直接结束，记录 `skip` 日志。

### 4.4 幂等策略

1. 事件级：`DedupeKey` 按业务对象生成（如 `activity.updated:{activityID}:{version}`）。  
2. 收件箱级：依赖 `uk_receiver_notification` 唯一约束防止重复入箱。

### 4.5 本期建议落地优化

1. `notification_inbox` 必须走批量插入，避免逐条写入导致数据库抖动。  
2. 列表查询使用 `idx_inbox_list(receiver_id, inbox_status, created_at)` 保障排序分页性能。  
3. 未读筛选使用 `idx_receiver_status_read_created`，保持 `unreadOnly` 查询稳定。  
4. `Publish(channel)` 保持非阻塞写法，队列满仅记录日志与指标，不反向拖慢主业务。

### 4.6 异常处理与可观测性

1. 事件发布失败（如 channel 满）：
记录 `warn` 日志，计数 `notify_publish_dropped_total`。  
2. 事件消费失败（查询接收人/写库失败）：
记录 `error` 日志，计数 `notify_consume_failed_total`。  
3. 扇出写库部分失败：
当前批次失败直接返回错误，不重试部分成功写入，依赖幂等避免重复。  
4. 关键日志字段建议统一：
`event_type` `biz_type` `biz_id` `source_org_id` `receiver_count` `success_count` `cost_ms`。

### 4.7 端到端时序（执行版）

1. 主业务接口完成数据库写入并提交事务。  
2. 组装 `NotificationEvent`（含 `DedupeKey` 与 `Payload`）。  
3. 调用 `Publish(channel)` 非阻塞入队。  
4. Worker 拉取事件并校验事件字段。  
5. 根据 `EventType` 查询并去重接收人。  
6. 渲染通知标题与正文。  
7. 插入 `notifications` 主记录。  
8. 按批次插入 `notification_inbox`。  
9. 输出消费结果日志并更新指标。

### 4.8 主业务完成后创建通知任务流程

目标：明确“主业务结束后，如何把通知任务交给异步处理”。

本期默认（无 outbox）：

1. 主业务事务提交成功。  
2. 组装 `NotificationEvent`（包含 `eventType`、`bizType`、`bizId`、`sourceOrgId`、`payload`、`dedupeKey`）。  
3. 调用 `notificationDispatcher.Publish(evt)` 非阻塞入队。  
4. 若入队失败（队列满），仅记录日志与指标，不影响主业务返回。  
5. worker 后续消费并完成通知落库。

可选增强（启用 outbox）：

1. 在主业务本地事务中同时写入业务数据和 `notification_outbox`。  
2. 事务提交后可尝试 `Publish(channel)` 做实时投递。  
3. worker 优先消费 channel；定时扫描 outbox 做补偿。  
4. outbox 成功后置 `success`，失败进入 `retry_wait/dead`。

建议伪代码：

```go
func (s *ActivityService) CreateActivity(req *CreateReq) error {
    var evt notification.NotificationEvent

    err := s.repo.DB.Transaction(func(tx *gorm.DB) error {
        activity, err := s.repo.CreateActivity(tx, req)
        if err != nil {
            return err
        }

        evt = notification.NotificationEvent{
            EventType:   "activity.created",
            BizType:     "activity",
            BizID:       activity.ID,
            SourceOrgID: activity.OrganizationID,
            ActorID:     req.OperatorID,
            Payload: map[string]any{
                "activityName": activity.Title,
                "startTime":    activity.StartTime,
            },
            DedupeKey: fmt.Sprintf("activity.created:%d", activity.ID),
            CreatedAt: time.Now(),
        }

        // 可选增强：启用 outbox 时在同事务写入
        // if enableOutbox {
        //     if err := s.notificationRepo.CreateOutbox(tx, evt); err != nil {
        //         return err
        //     }
        // }
        return nil
    })
    if err != nil {
        return err
    }

    // 事务提交后执行异步投递，不阻塞主业务成功返回
    s.notificationDispatcher.Publish(evt)
    return nil
}
```

## 5. 接口方案

1. `GET /api/notifications`
请求：`page`、`pageSize`、`unreadOnly`  
返回：`title` `content` `bizType` `bizId` `readStatus` `createdAt`

2. `POST /api/notifications/read`
请求：`ids[]`（批量已读）

接口行为细化：

1. `GET /api/notifications` 默认仅返回 `inbox_status=normal`。  
2. `unreadOnly=true` 时增加 `read_status=0` 条件。  
3. `POST /api/notifications/read` 仅允许更新当前登录用户自己的收件记录。  
4. `ids[]` 为空时返回参数错误；超大批量建议限制上限（如 500）。

## 6. 退出组织后的处理策略

1. 用户退出组织时，将该用户在该组织下的通知批量更新为 `inbox_status=archived`。  
2. 默认通知列表只返回 `inbox_status=normal`。  
3. 如需审计，可额外提供“查看归档通知”开关。  
4. 不建议直接删库，避免丢失业务留痕。

## 7. 代码落点

1. `internal/api/notification.proto`
2. `internal/handler/notifications.go`
3. `internal/service/notification.go`
4. `internal/service/notification_dispatcher.go`
5. `internal/repository/notification.go`
6. 触发接入：
`internal/service/activities.go`、`internal/service/membership.go`（或审核服务）

## 8. 验收清单

1. 创建活动后，组织内有效志愿者收到通知。  
2. 修改活动后，报名未取消志愿者收到通知。  
3. 成员加入组织通过后，本人收到通知。  
4. 列表分页与未读筛选正确。  
5. 批量已读正确生效。  
6. 退出组织后，相关通知默认不展示且可追溯。  
7. 通知发送失败不影响主业务成功返回。

验收用例补充：

1. 同一事件重复投递时，不会出现重复收件记录。  
2. 大于 10000 接收人场景下，写库仍按批次执行且接口不超时。  
3. channel 满载时，主业务接口仍能正常成功返回。  
4. `POST /api/notifications/read` 传入他人 inbox id 时应更新 0 行。  
5. 退出组织后再次拉取列表，不包含该组织历史通知。

## 9. 增强功能补充（按需开启）

1. `notification_outbox` 可靠投递：
主业务数据写入与 outbox 写入必须在同一个本地事务中，避免“主业务成功但事件丢失”。
2. outbox 扫表并发控制：
多节点部署时，优先使用 `FOR UPDATE SKIP LOCKED` 抢占任务；不满足条件时用分布式锁保证单活扫描。
3. 数据生命周期治理（TTL）：
增加定时清理/归档任务，处理 6 个月前且已读、已归档通知，避免表持续膨胀影响查询。
