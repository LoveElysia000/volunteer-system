package repository

import (
	"context"
	"errors"
	"time"
	"volunteer-system/internal/model"
	"volunteer-system/pkg/util"

	"gorm.io/gorm"
)

// AssistantActivitySearchRow 助手活动检索结果
type AssistantActivitySearchRow struct {
	ID            int64
	OrgID         int64
	OrgName       string
	Title         string
	Description   string
	StartTime     time.Time
	EndTime       time.Time
	Location      string
	MaxPeople     int32
	CurrentPeople int32
	Status        int32
}

// CreateAiSession 创建 AI 会话
func (r *Repository) CreateAiSession(db *gorm.DB, session *model.AiSession) error {
	return db.WithContext(r.ctx).Create(session).Error
}

// GetAiSessionByIDAndUser 根据会话ID与用户ID查询会话
func (r *Repository) GetAiSessionByIDAndUser(db *gorm.DB, sessionID, userID int64) (*model.AiSession, error) {
	var session model.AiSession
	err := db.WithContext(r.ctx).
		Model(&model.AiSession{}).
		Where("id = ? AND user_id = ?", sessionID, userID).
		First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// UpdateAiSessionAfterMessage 更新会话最后消息时间与可选标题
func (r *Repository) UpdateAiSessionAfterMessage(db *gorm.DB, sessionID int64, lastMessageAt time.Time, title string) error {
	updates := map[string]any{
		"last_message_at": lastMessageAt,
	}
	if title != "" {
		updates["title"] = title
	}
	return db.WithContext(r.ctx).
		Model(&model.AiSession{}).
		Where("id = ?", sessionID).
		Updates(updates).Error
}

// GetNextAiMessageSeqNo 获取会话内下一条消息序号
func (r *Repository) GetNextAiMessageSeqNo(db *gorm.DB, sessionID int64) (int32, error) {
	var maxSeq int64
	err := db.WithContext(r.ctx).
		Model(&model.AiMessage{}).
		Where("session_id = ?", sessionID).
		Select("COALESCE(MAX(seq_no), 0)").
		Scan(&maxSeq).Error
	if err != nil {
		return 0, err
	}
	return int32(maxSeq + 1), nil
}

// CreateAiMessage 创建 AI 消息
func (r *Repository) CreateAiMessage(db *gorm.DB, message *model.AiMessage) error {
	return db.WithContext(r.ctx).Create(message).Error
}

// ListRecentAiMessagesBySession 查询会话最近 N 条消息，按时间正序返回
func (r *Repository) ListRecentAiMessagesBySession(db *gorm.DB, sessionID int64, limit int) ([]*model.AiMessage, error) {
	messages := make([]*model.AiMessage, 0)
	if limit <= 0 {
		return messages, nil
	}

	err := db.WithContext(r.ctx).
		Model(&model.AiMessage{}).
		Where("session_id = ?", sessionID).
		Order("seq_no DESC").
		Limit(limit).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}

	// 反转为正序
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// ListAiMessagesBySession 查询会话全部消息（按序号正序）
func (r *Repository) ListAiMessagesBySession(db *gorm.DB, sessionID int64) ([]*model.AiMessage, error) {
	messages := make([]*model.AiMessage, 0)
	err := db.WithContext(r.ctx).
		Model(&model.AiMessage{}).
		Where("session_id = ?", sessionID).
		Order("seq_no ASC").
		Find(&messages).Error
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// CreateAiToolCall 创建 AI 工具调用日志
func (r *Repository) CreateAiToolCall(db *gorm.DB, call *model.AiToolCall) error {
	return db.WithContext(r.ctx).Create(call).Error
}

// UpdateAiToolCallMessageID 批量绑定工具调用日志对应 assistant 消息ID
func (r *Repository) UpdateAiToolCallMessageID(db *gorm.DB, ids []int64, messageID int64) error {
	if len(ids) == 0 {
		return nil
	}
	return db.WithContext(r.ctx).
		Model(&model.AiToolCall{}).
		Where("id IN ?", ids).
		Update("message_id", messageID).Error
}

// GetAiUsageDaily 查询用户当天用量
func (r *Repository) GetAiUsageDaily(db *gorm.DB, bizDate time.Time, userID int64) (*model.AiUsageDaily, error) {
	var usage model.AiUsageDaily
	err := db.WithContext(r.ctx).
		Model(&model.AiUsageDaily{}).
		Where("biz_date = ? AND user_id = ?", bizDate.Format("2006-01-02"), userID).
		First(&usage).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &usage, nil
}

// UpsertAiUsageDaily 累加每日用量
func (r *Repository) UpsertAiUsageDaily(db *gorm.DB, bizDate time.Time, userID int64, success bool, tokenIn, tokenOut int32, estimatedCost float64) error {
	successCount := int64(0)
	failedCount := int64(1)
	if success {
		successCount = 1
		failedCount = 0
	}

	// 使用原子 upsert，避免并发请求下的日聚合丢失。
	return db.WithContext(r.ctx).Exec(`
INSERT INTO ai_usage_daily (
	biz_date,
	user_id,
	request_count,
	success_count,
	failed_count,
	token_in_total,
	token_out_total,
	estimated_cost,
	created_at,
	updated_at
)
VALUES (?, ?, 1, ?, ?, ?, ?, ?, NOW(), NOW())
ON DUPLICATE KEY UPDATE
	request_count = request_count + 1,
	success_count = success_count + VALUES(success_count),
	failed_count = failed_count + VALUES(failed_count),
	token_in_total = token_in_total + VALUES(token_in_total),
	token_out_total = token_out_total + VALUES(token_out_total),
	estimated_cost = estimated_cost + VALUES(estimated_cost),
	updated_at = NOW()
`, bizDate.Format("2006-01-02"), userID, successCount, failedCount, tokenIn, tokenOut, estimatedCost).Error
}

// SearchAssistantActivities 助手活动检索
func (r *Repository) SearchAssistantActivities(db *gorm.DB, ctx context.Context, keyword string, status int32, limit int) ([]*AssistantActivitySearchRow, error) {
	rows := make([]*AssistantActivitySearchRow, 0)

	query := db.WithContext(ctx).
		Table("activities as a").
		Select("a.id, a.org_id, o.org_name, a.title, a.description, a.start_time, a.end_time, a.location, a.max_people, a.current_people, a.status").
		Joins("LEFT JOIN organizations o ON o.id = a.org_id")

	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("a.title LIKE ? OR a.description LIKE ? OR a.location LIKE ?", like, like, like)
	}
	if status > 0 {
		query = query.Where("a.status = ?", status)
	}

	if limit <= 0 {
		limit = 5
	}
	if limit > 20 {
		limit = 20
	}

	if err := query.Order("a.start_time DESC").Limit(limit).Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// GetAssistantAccessibleOrgIDs 查询账号可访问组织ID列表
func (r *Repository) GetAssistantAccessibleOrgIDs(db *gorm.DB, ctx context.Context, userID int64, minRole int32) ([]int64, error) {
	orgIDs := make([]int64, 0)

	// 组织创建者天然拥有访问权限。
	ownerOrgIDs := make([]int64, 0)
	if err := db.WithContext(ctx).
		Model(&model.Organization{}).
		Where("account_id = ?", userID).
		Pluck("id", &ownerOrgIDs).Error; err != nil {
		return nil, err
	}
	orgIDs = append(orgIDs, ownerOrgIDs...)

	// 志愿者成员身份按角色阈值过滤，最后统一去重。
	var volunteer model.Volunteer
	err := db.WithContext(ctx).
		Model(&model.Volunteer{}).
		Where("account_id = ?", userID).
		First(&volunteer).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil {
		memberOrgIDs := make([]int64, 0)
		if err := db.WithContext(ctx).
			Model(&model.OrgMember{}).
			Where("volunteer_id = ? AND status = ? AND role >= ?", volunteer.ID, model.MemberStatusActive, minRole).
			Pluck("org_id", &memberOrgIDs).Error; err != nil {
			return nil, err
		}
		orgIDs = append(orgIDs, memberOrgIDs...)
	}

	return util.UniquePositiveInt64(orgIDs), nil
}

// GetAssistantActivityStatusCountsByOrg 统计组织活动状态数量
func (r *Repository) GetAssistantActivityStatusCountsByOrg(db *gorm.DB, ctx context.Context, orgID int64) (map[int32]int64, error) {
	type statusCount struct {
		Status int32 `gorm:"column:status"`
		Count  int64 `gorm:"column:count"`
	}

	rows := make([]statusCount, 0)
	if err := db.WithContext(ctx).
		Model(&model.Activity{}).
		Select("status, COUNT(*) as count").
		Where("org_id = ?", orgID).
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[int32]int64, len(rows))
	for _, row := range rows {
		result[row.Status] = row.Count
	}
	return result, nil
}

// CountAssistantSignupsByOrg 统计组织活动报名总数
func (r *Repository) CountAssistantSignupsByOrg(db *gorm.DB, ctx context.Context, orgID int64) (int64, error) {
	var total int64
	err := db.WithContext(ctx).
		Table("activity_signups as s").
		Joins("INNER JOIN activities a ON a.id = s.activity_id").
		Where("a.org_id = ?", orgID).
		Count(&total).Error
	if err != nil {
		return 0, err
	}
	return total, nil
}

// GetAssistantOrganizationByIDWithContext 查询组织（使用外部 context，便于工具超时控制）
func (r *Repository) GetAssistantOrganizationByIDWithContext(db *gorm.DB, ctx context.Context, orgID int64) (*model.Organization, error) {
	var organization model.Organization
	err := db.WithContext(ctx).
		Model(&model.Organization{}).
		Where("id = ?", orgID).
		First(&organization).Error
	if err != nil {
		return nil, err
	}
	return &organization, nil
}
