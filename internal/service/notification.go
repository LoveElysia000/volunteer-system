package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"volunteer-system/config"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/notify"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

const (
	defaultNotificationPageSize  = 20
	maxNotificationPageSize      = 100
	maxNotificationReadBatchSize = 500
	defaultInboxCreateBatchSize  = 500
)

type NotificationService struct {
	Service
}

var sendEmailFn = notify.SendEmail

// NewNotificationService 创建通知服务实例。
func NewNotificationService(ctx context.Context, c *app.RequestContext) *NotificationService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &NotificationService{
		Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
}

// ListNotifications 获取当前登录用户通知列表。
func (s *NotificationService) ListNotifications(req *api.NotificationListRequest) (*api.NotificationListResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if s.c == nil {
		return nil, errors.New("请求上下文不能为空")
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("通知列表查询失败: 获取当前用户ID异常: %v", err)
		return nil, err
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = defaultNotificationPageSize
	}
	if pageSize > maxNotificationPageSize {
		pageSize = maxNotificationPageSize
	}
	offset := (int(page) - 1) * int(pageSize)

	rows, total, err := s.repo.ListNotificationInboxByReceiver(s.repo.DB, userID, req.UnreadOnly, int(pageSize), offset)
	if err != nil {
		log.Error("通知列表查询失败: 查询通知收件箱异常: %v, user_id=%d page=%d page_size=%d unread_only=%v", err, userID, page, pageSize, req.UnreadOnly)
		return nil, err
	}

	resp := &api.NotificationListResponse{
		Total: int32(total),
		List:  make([]*api.NotificationItem, 0, len(rows)),
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		resp.List = append(resp.List, &api.NotificationItem{
			InboxId:        row.InboxID,
			NotificationId: row.NotificationID,
			EventType:      row.EventType,
			BizType:        row.BizType,
			BizId:          row.BizID,
			Title:          row.Title,
			Content:        row.Content,
			ReadStatus:     row.ReadStatus,
			ReadAt:         util.FormatDateTimePtr(row.ReadAt),
			CreatedAt:      util.FormatDateTimeOrEmpty(row.CreatedAt),
		})
	}
	return resp, nil
}

// MarkNotificationsRead 批量标记已读。
func (s *NotificationService) MarkNotificationsRead(req *api.NotificationReadRequest) (*api.NotificationReadResponse, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if s.c == nil {
		return nil, errors.New("请求上下文不能为空")
	}

	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		log.Error("通知批量已读失败: 获取当前用户ID异常: %v", err)
		return nil, err
	}

	ids := util.UniquePositiveInt64(req.Ids)
	if len(ids) == 0 {
		return nil, errors.New("通知ID不能为空")
	}
	if len(ids) > maxNotificationReadBatchSize {
		return nil, fmt.Errorf("单次最多标记%d条通知", maxNotificationReadBatchSize)
	}

	updated, err := s.repo.MarkNotificationInboxReadByIDs(s.repo.DB, userID, ids, time.Now())
	if err != nil {
		log.Error("通知批量已读失败: 更新通知异常: %v, user_id=%d id_count=%d", err, userID, len(ids))
		return nil, err
	}
	return &api.NotificationReadResponse{Updated: int32(updated)}, nil
}

// HandleEvent 异步消费通知事件并落库。
// 返回 created=true 表示本次为首次创建通知，可继续触达外部渠道（如邮件）。
func (s *NotificationService) HandleEvent(evt NotificationEvent) (bool, error) {
	if err := validateNotificationEvent(evt); err != nil {
		return false, err
	}
	// 优先使用业务侧传入的幂等键；为空时按事件类型生成稳定兜底键。
	dedupeKey := strings.TrimSpace(evt.DedupeKey)
	if dedupeKey == "" {
		dedupeKey = buildNotificationDedupeKey(evt)
	}
	if dedupeKey == "" {
		return false, errors.New("通知事件幂等键不能为空")
	}

	receiverIDs, err := s.resolveReceiverAccountIDs(evt)
	if err != nil {
		return false, err
	}
	receiverIDs = util.UniquePositiveInt64(receiverIDs)
	if len(receiverIDs) == 0 {
		log.Info("通知事件跳过: 无可用接收人 event_type=%s biz_type=%s biz_id=%d source_org_id=%d", evt.EventType, evt.BizType, evt.BizID, evt.SourceOrgID)
		return false, nil
	}

	title, content := renderNotificationMessage(evt)
	if title == "" {
		title = "系统通知"
	}
	if content == "" {
		content = "您有一条新的通知，请及时查看。"
	}

	var (
		notificationID int64
		created        bool
	)
	err = s.withTransaction(func(tx *gorm.DB) error {
		notification := &model.Notification{
			EventType:   evt.EventType,
			BizType:     evt.BizType,
			BizID:       evt.BizID,
			SourceOrgID: evt.SourceOrgID,
			SenderID:    evt.ActorID,
			DedupeKey:   dedupeKey,
			Title:       title,
			Content:     content,
		}
		notificationID, created, err = s.repo.CreateNotificationWithDedupe(tx, notification)
		if err != nil {
			return err
		}
		// 命中幂等时不再重复写入收件箱，避免重复通知。
		if !created {
			return nil
		}

		inboxes := make([]*model.NotificationInbox, 0, len(receiverIDs))
		for _, receiverID := range receiverIDs {
			inboxes = append(inboxes, &model.NotificationInbox{
				NotificationID: notificationID,
				ReceiverID:     receiverID,
				SourceOrgID:    evt.SourceOrgID,
				ReadStatus:     model.NotificationReadStatusUnread,
				InboxStatus:    model.NotificationInboxStatusNormal,
			})
		}
		return s.repo.CreateNotificationInboxInBatches(tx, inboxes, defaultInboxCreateBatchSize)
	})
	if err != nil {
		return false, err
	}
	if !created {
		log.Info("通知事件幂等跳过: event_type=%s biz_type=%s biz_id=%d dedupe_key=%s",
			evt.EventType, evt.BizType, evt.BizID, dedupeKey)
		return false, nil
	}

	log.Info("通知事件消费成功: event_type=%s biz_type=%s biz_id=%d source_org_id=%d receiver_count=%d dedupe_key=%s",
		evt.EventType, evt.BizType, evt.BizID, evt.SourceOrgID, len(receiverIDs), dedupeKey)
	return true, nil
}

// handleEventAndDispatchEmail 仅在通知首次创建成功时发送邮件渠道，避免重复事件重复发信。
func (s *NotificationService) handleEventAndDispatchEmail(evt NotificationEvent) error {
	created, err := s.HandleEvent(evt)
	if err != nil {
		return err
	}
	if !created {
		return nil
	}
	return s.sendEventEmail(evt)
}

func (s *NotificationService) sendEventEmail(evt NotificationEvent) error {
	cfg := config.GetConfig()
	if cfg == nil || cfg.Email == nil || !cfg.Email.Enabled {
		return nil
	}

	receiverIDs, err := s.resolveReceiverAccountIDs(evt)
	if err != nil {
		return err
	}
	receiverIDs = util.UniquePositiveInt64(receiverIDs)
	if len(receiverIDs) == 0 {
		return nil
	}

	subject, body := renderNotificationMessage(evt)
	for _, receiverID := range receiverIDs {
		account, findErr := s.repo.FindByID(s.repo.DB, receiverID)
		if findErr != nil || account == nil {
			continue
		}
		email := strings.TrimSpace(account.Email)
		if email == "" {
			continue
		}
		if err := sendEmailFn(cfg.Email, email, subject, body); err != nil {
			log.Error("邮件通知发送失败: receiver_id=%d email=%s err=%v", receiverID, email, err)
		}
	}
	return nil
}

func validateNotificationEvent(evt NotificationEvent) error {
	if strings.TrimSpace(evt.EventType) == "" {
		return errors.New("通知事件类型不能为空")
	}
	if strings.TrimSpace(evt.BizType) == "" {
		return errors.New("通知事件业务类型不能为空")
	}
	if evt.BizID <= 0 {
		return errors.New("通知事件业务ID不能为空")
	}
	return nil
}

func (s *NotificationService) resolveReceiverAccountIDs(evt NotificationEvent) ([]int64, error) {
	switch evt.EventType {
	case model.NotificationEventActivityCreated:
		sourceOrgID := evt.SourceOrgID
		// source_org_id 允许由事件端省略，这里基于 activity 反查做兜底。
		if sourceOrgID <= 0 && evt.BizID > 0 {
			activity, err := s.repo.GetActivityByID(s.repo.DB, evt.BizID)
			if err == nil && activity != nil {
				sourceOrgID = activity.OrgID
			}
		}
		// 兜底后仍无组织ID则拒绝继续扇出，避免错误范围通知。
		if sourceOrgID <= 0 {
			return nil, errors.New("活动创建通知缺少组织ID")
		}
		return s.repo.ListActiveReceiverAccountIDsByOrgID(s.repo.DB, sourceOrgID)

	case model.NotificationEventActivityUpdated:
		return s.repo.ListActivitySignupReceiverAccountIDs(s.repo.DB, evt.BizID)

	case model.NotificationEventActivityCanceled:
		return s.repo.ListActivitySignupReceiverAccountIDs(s.repo.DB, evt.BizID)

	case model.NotificationEventMemberJoinApproved:
		receiverID := payloadInt64(evt.Payload, "receiverAccountID")
		if receiverID > 0 {
			return []int64{receiverID}, nil
		}

		member, err := s.repo.GetMembershipByID(s.repo.DB, evt.BizID)
		if err != nil {
			return nil, err
		}
		volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, member.VolunteerID)
		if err != nil {
			return nil, err
		}
		if volunteer.AccountID <= 0 {
			return nil, errors.New("成员账号ID无效")
		}
		return []int64{volunteer.AccountID}, nil

	case model.NotificationEventSignupRejected:
		receiverID := payloadInt64(evt.Payload, "receiverAccountID")
		if receiverID > 0 {
			return []int64{receiverID}, nil
		}
		signup, err := s.repo.GetActivitySignupByID(s.repo.DB, evt.BizID)
		if err != nil {
			return nil, err
		}
		volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, signup.VolunteerID)
		if err != nil {
			return nil, err
		}
		if volunteer.AccountID <= 0 {
			return nil, errors.New("成员账号ID无效")
		}
		return []int64{volunteer.AccountID}, nil

	default:
		return nil, fmt.Errorf("不支持的通知事件类型: %s", evt.EventType)
	}
}

func renderNotificationMessage(evt NotificationEvent) (string, string) {
	switch evt.EventType {
	case model.NotificationEventActivityCreated:
		activityTitle := firstNonEmpty(
			payloadString(evt.Payload, "activityTitle"),
			payloadString(evt.Payload, "activityName"),
			payloadString(evt.Payload, "title"),
		)
		if activityTitle == "" {
			return "新活动发布", "您所在组织发布了新活动，请及时查看。"
		}
		return "新活动发布：" + activityTitle, "您所在组织发布了新活动《" + activityTitle + "》，请及时查看并报名。"

	case model.NotificationEventActivityUpdated:
		activityTitle := firstNonEmpty(
			payloadString(evt.Payload, "activityTitle"),
			payloadString(evt.Payload, "activityName"),
			payloadString(evt.Payload, "title"),
		)
		if activityTitle == "" {
			return "活动信息更新", "您报名的活动信息已更新，请及时查看最新安排。"
		}
		return "活动更新：" + activityTitle, "您报名的活动《" + activityTitle + "》信息已更新，请及时查看最新安排。"

	case model.NotificationEventActivityCanceled:
		activityTitle := firstNonEmpty(
			payloadString(evt.Payload, "activityTitle"),
			payloadString(evt.Payload, "activityName"),
			payloadString(evt.Payload, "title"),
		)
		if activityTitle == "" {
			return "活动已取消", "您报名的活动已取消，请关注后续安排。"
		}
		return "活动已取消：" + activityTitle, "您报名的活动《" + activityTitle + "》已取消，请关注后续安排。"

	case model.NotificationEventMemberJoinApproved:
		orgName := payloadString(evt.Payload, "organizationName")
		if orgName == "" {
			return "加入组织申请已通过", "您提交的加入组织申请已通过审核。"
		}
		return "加入组织申请已通过", "您提交的加入「" + orgName + "」申请已通过审核。"

	case model.NotificationEventSignupRejected:
		activityTitle := firstNonEmpty(
			payloadString(evt.Payload, "activityTitle"),
			payloadString(evt.Payload, "activityName"),
			payloadString(evt.Payload, "title"),
		)
		reason := payloadString(evt.Payload, "reason")
		if activityTitle == "" {
			if reason == "" {
				return "活动报名未通过", "您提交的活动报名申请未通过审核。"
			}
			return "活动报名未通过", "您提交的活动报名申请未通过审核，原因：" + reason
		}
		if reason == "" {
			return "活动报名未通过：" + activityTitle, "您提交的活动《" + activityTitle + "》报名申请未通过审核。"
		}
		return "活动报名未通过：" + activityTitle, "您提交的活动《" + activityTitle + "》报名申请未通过审核，原因：" + reason

	default:
		return "系统通知", "您有一条新的通知，请及时查看。"
	}
}

func payloadString(payload map[string]any, key string) string {
	if len(payload) == 0 {
		return ""
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}

func payloadInt64(payload map[string]any, key string) int64 {
	if len(payload) == 0 {
		return 0
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return 0
	}
	switch v := raw.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	default:
		return 0
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func buildNotificationDedupeKey(evt NotificationEvent) string {
	switch evt.EventType {
	case model.NotificationEventActivityCreated:
		return fmt.Sprintf("activity.created:%d", evt.BizID)
	case model.NotificationEventActivityUpdated:
		// 更新事件优先使用 updatedAt/version，确保同一版本重复投递可命中幂等。
		version := firstNonEmpty(
			payloadString(evt.Payload, "updatedAt"),
			payloadString(evt.Payload, "version"),
		)
		if version != "" {
			return fmt.Sprintf("activity.updated:%d:%s", evt.BizID, version)
		}
		return fmt.Sprintf("activity.updated:%d", evt.BizID)
	case model.NotificationEventActivityCanceled:
		return fmt.Sprintf("activity.canceled:%d", evt.BizID)
	case model.NotificationEventMemberJoinApproved:
		return fmt.Sprintf("member.join.approved:%d", evt.BizID)
	case model.NotificationEventSignupRejected:
		return fmt.Sprintf("signup.rejected:%d", evt.BizID)
	default:
		return fmt.Sprintf("%s:%s:%d:%d", evt.EventType, evt.BizType, evt.BizID, evt.SourceOrgID)
	}
}
