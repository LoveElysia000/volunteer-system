package service

import (
	"context"
	"sync"
	"time"
)

const (
	defaultNotificationQueueSize = 2048
	defaultNotificationWorkers   = 4
)

// NotificationEvent 通知事件。
type NotificationEvent struct {
	EventType   string
	BizType     string
	BizID       int64
	SourceOrgID int64
	ActorID     int64
	CreatedAt   time.Time
	Payload     map[string]any
	DedupeKey   string
}

// NotificationDispatcher 异步通知分发器。
type NotificationDispatcher struct {
	ch        chan NotificationEvent
	workers   int
	startOnce sync.Once
}

var defaultNotificationDispatcher = NewNotificationDispatcher(defaultNotificationQueueSize, defaultNotificationWorkers)

// NewNotificationDispatcher 创建通知分发器。
func NewNotificationDispatcher(queueSize, workers int) *NotificationDispatcher {
	if queueSize <= 0 {
		queueSize = defaultNotificationQueueSize
	}
	if workers <= 0 {
		workers = defaultNotificationWorkers
	}
	return &NotificationDispatcher{
		ch:      make(chan NotificationEvent, queueSize),
		workers: workers,
	}
}

// InitNotificationDispatcher 启动全局通知分发器。
func InitNotificationDispatcher(ctx context.Context) {
	defaultNotificationDispatcher.Start(ctx)
}

// PublishNotificationEvent 发布通知事件（非阻塞）。
func PublishNotificationEvent(evt NotificationEvent) {
	defaultNotificationDispatcher.Publish(evt)
}

// Start 启动分发器 worker。
func (d *NotificationDispatcher) Start(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	d.startOnce.Do(func() {
		for i := 0; i < d.workers; i++ {
			workerID := i + 1
			go d.consume(ctx, workerID)
		}
		log.Info("通知分发器启动完成: workers=%d queue_size=%d", d.workers, cap(d.ch))
	})
}

// Publish 将事件发布到队列，队列满时丢弃并记录日志。
func (d *NotificationDispatcher) Publish(evt NotificationEvent) {
	if evt.CreatedAt.IsZero() {
		evt.CreatedAt = time.Now()
	}
	select {
	case d.ch <- evt:
	default:
		log.Warn("通知事件入队失败: 队列已满 event_type=%s biz_type=%s biz_id=%d source_org_id=%d dedupe_key=%s",
			evt.EventType, evt.BizType, evt.BizID, evt.SourceOrgID, evt.DedupeKey)
	}
}

func (d *NotificationDispatcher) consume(ctx context.Context, workerID int) {
	svc := NewNotificationService(ctx, nil)
	for {
		select {
		case <-ctx.Done():
			log.Info("通知分发器worker退出: worker_id=%d", workerID)
			return
		case evt := <-d.ch:
			start := time.Now()
			if err := svc.HandleEvent(evt); err != nil {
				log.Error("通知事件消费失败: worker_id=%d event_type=%s biz_type=%s biz_id=%d source_org_id=%d err=%v",
					workerID, evt.EventType, evt.BizType, evt.BizID, evt.SourceOrgID, err)
				continue
			}
			log.Info("通知事件消费完成: worker_id=%d event_type=%s biz_type=%s biz_id=%d source_org_id=%d cost_ms=%d",
				workerID, evt.EventType, evt.BizType, evt.BizID, evt.SourceOrgID, time.Since(start).Milliseconds())
		}
	}
}
