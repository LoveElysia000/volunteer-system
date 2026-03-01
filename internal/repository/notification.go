package repository

import (
	"time"
	"volunteer-system/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// NotificationInboxWithContent 通知收件箱与通知正文的聚合视图。
type NotificationInboxWithContent struct {
	InboxID        int64      `gorm:"column:inbox_id"`
	NotificationID int64      `gorm:"column:notification_id"`
	EventType      string     `gorm:"column:event_type"`
	BizType        string     `gorm:"column:biz_type"`
	BizID          int64      `gorm:"column:biz_id"`
	Title          string     `gorm:"column:title"`
	Content        string     `gorm:"column:content"`
	ReadStatus     int32      `gorm:"column:read_status"`
	ReadAt         *time.Time `gorm:"column:read_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
}

// CreateNotification 创建通知正文记录。
func (r *Repository) CreateNotification(db *gorm.DB, notification *model.Notification) error {
	return db.WithContext(r.ctx).Create(notification).Error
}

// CreateNotificationWithDedupe 基于 dedupe_key 幂等创建通知记录。
// 返回值: notificationID, created(是否新建), error。
func (r *Repository) CreateNotificationWithDedupe(db *gorm.DB, notification *model.Notification) (int64, bool, error) {
	if notification == nil {
		return 0, false, nil
	}

	// 基于唯一键 dedupe_key 做幂等插入：首次写入成功，重复写入 DoNothing。
	result := db.WithContext(r.ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "dedupe_key"}},
			DoNothing: true,
		}).
		Create(notification)
	if result.Error != nil {
		return 0, false, result.Error
	}
	if result.RowsAffected > 0 {
		return notification.ID, true, nil
	}

	// 重复事件场景下回查已有通知ID，供上层直接复用并跳过重复入箱。
	var existed model.Notification
	if err := db.WithContext(r.ctx).
		Model(&model.Notification{}).
		Select("id").
		Where("dedupe_key = ?", notification.DedupeKey).
		Take(&existed).Error; err != nil {
		return 0, false, err
	}
	return existed.ID, false, nil
}

// CreateNotificationInboxInBatches 批量创建收件箱记录。
func (r *Repository) CreateNotificationInboxInBatches(db *gorm.DB, inboxes []*model.NotificationInbox, batchSize int) error {
	if len(inboxes) == 0 {
		return nil
	}
	if batchSize <= 0 {
		batchSize = 500
	}
	return db.WithContext(r.ctx).CreateInBatches(inboxes, batchSize).Error
}

// ListNotificationInboxByReceiver 查询当前用户通知列表。
func (r *Repository) ListNotificationInboxByReceiver(db *gorm.DB, receiverID int64, unreadOnly bool, limit, offset int) ([]*NotificationInboxWithContent, int64, error) {
	rows := make([]*NotificationInboxWithContent, 0)
	if receiverID <= 0 {
		return rows, 0, nil
	}

	base := db.WithContext(r.ctx).
		Table("notification_inbox AS ni").
		Where("ni.receiver_id = ?", receiverID).
		Where("ni.inbox_status = ?", model.NotificationInboxStatusNormal)

	if unreadOnly {
		base = base.Where("ni.read_status = ?", model.NotificationReadStatusUnread)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return rows, 0, nil
	}

	query := base.
		Select("ni.id AS inbox_id, ni.notification_id, ni.read_status, ni.read_at, ni.created_at AS created_at, n.event_type, n.biz_type, n.biz_id, n.title, n.content").
		Joins("INNER JOIN notifications AS n ON n.id = ni.notification_id").
		Order("ni.created_at DESC, ni.id DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// MarkNotificationInboxReadByIDs 批量标记收件箱记录已读。
func (r *Repository) MarkNotificationInboxReadByIDs(db *gorm.DB, receiverID int64, inboxIDs []int64, readAt time.Time) (int64, error) {
	if receiverID <= 0 || len(inboxIDs) == 0 {
		return 0, nil
	}
	result := db.WithContext(r.ctx).
		Model(&model.NotificationInbox{}).
		Where("receiver_id = ? AND id IN ? AND inbox_status = ? AND read_status = ?", receiverID, inboxIDs, model.NotificationInboxStatusNormal, model.NotificationReadStatusUnread).
		Updates(map[string]any{
			"read_status": model.NotificationReadStatusRead,
			"read_at":     readAt,
		})
	return result.RowsAffected, result.Error
}

// ArchiveNotificationInboxByReceiverAndOrg 归档用户在指定组织下的通知。
func (r *Repository) ArchiveNotificationInboxByReceiverAndOrg(db *gorm.DB, receiverID, sourceOrgID int64, reason string, archivedAt time.Time) (int64, error) {
	if receiverID <= 0 || sourceOrgID <= 0 {
		return 0, nil
	}
	result := db.WithContext(r.ctx).
		Model(&model.NotificationInbox{}).
		Where("receiver_id = ? AND source_org_id = ? AND inbox_status = ?", receiverID, sourceOrgID, model.NotificationInboxStatusNormal).
		Updates(map[string]any{
			"inbox_status":    model.NotificationInboxStatusArchived,
			"archived_reason": reason,
			"archived_at":     archivedAt,
		})
	return result.RowsAffected, result.Error
}

// ListActiveReceiverAccountIDsByOrgID 查询组织下可接收通知的账号ID列表。
func (r *Repository) ListActiveReceiverAccountIDsByOrgID(db *gorm.DB, orgID int64) ([]int64, error) {
	accountIDs := make([]int64, 0)
	if orgID <= 0 {
		return accountIDs, nil
	}

	err := db.WithContext(r.ctx).
		Table("org_members AS m").
		Distinct("v.account_id").
		Joins("INNER JOIN volunteers AS v ON v.id = m.volunteer_id").
		Joins("INNER JOIN sys_accounts AS a ON a.id = v.account_id").
		Where("m.org_id = ? AND m.status = ? AND v.status = ? AND a.status = ? AND a.deleted_at IS NULL",
			orgID, model.MemberStatusActive, model.VolunteerActiveStatus, model.SysAccountNormal).
		Pluck("v.account_id", &accountIDs).Error
	if err != nil {
		return nil, err
	}
	return accountIDs, nil
}

// ListActivitySignupReceiverAccountIDs 查询活动报名用户（未取消）对应账号ID列表。
func (r *Repository) ListActivitySignupReceiverAccountIDs(db *gorm.DB, activityID int64) ([]int64, error) {
	accountIDs := make([]int64, 0)
	if activityID <= 0 {
		return accountIDs, nil
	}

	activeSignupStatuses := []int32{
		model.ActivitySignupStatusPending,
		model.ActivitySignupStatusSuccess,
	}
	err := db.WithContext(r.ctx).
		Table("activity_signups AS s").
		Distinct("v.account_id").
		Joins("INNER JOIN volunteers AS v ON v.id = s.volunteer_id").
		Joins("INNER JOIN sys_accounts AS a ON a.id = v.account_id").
		Where("s.activity_id = ? AND s.status IN ? AND v.status = ? AND a.status = ? AND a.deleted_at IS NULL",
			activityID, activeSignupStatuses, model.VolunteerActiveStatus, model.SysAccountNormal).
		Pluck("v.account_id", &accountIDs).Error
	if err != nil {
		return nil, err
	}
	return accountIDs, nil
}
