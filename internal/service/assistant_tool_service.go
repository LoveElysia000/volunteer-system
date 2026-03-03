package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"

	"github.com/cloudwego/hertz/pkg/app"
)

const (
	assistantToolActivitySearch        = "activity_search"
	assistantToolActivityStats         = "activity_stats"
	assistantToolActivityDraftGenerate = "activity_draft_generate"

	assistantToolTimeout     = 3 * time.Second
	assistantToolMaxAttempts = 2 // 首次 + 1 次重试
)

var (
	errToolPermissionDenied = errors.New("permission denied")
	errToolInvalidInput     = errors.New("invalid tool input")
)

// AssistantToolService AI 工具执行服务
type AssistantToolService struct {
	Service
}

// NewAssistantToolService 创建工具服务
func NewAssistantToolService(ctx context.Context, c *app.RequestContext) *AssistantToolService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &AssistantToolService{
		Service: Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
}

// PlanTools 按场景和用户问题推断需要执行的工具
func (t *AssistantToolService) PlanTools(scene, message string) []assistantToolPlan {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return nil
	}

	plans := make([]assistantToolPlan, 0, 3)
	added := make(map[string]struct{})

	addPlan := func(tool string, input map[string]any) {
		if len(plans) >= 3 {
			return
		}
		if _, ok := added[tool]; ok {
			return
		}
		added[tool] = struct{}{}
		plans = append(plans, assistantToolPlan{ToolName: tool, Input: input})
	}

	// 当前为规则启发式规划：优先保证可控性，避免模型直接决定工具链。
	if scene == assistantSceneActivityDraft || containsAny(msg, "草案", "活动方案", "活动策划", "活动发布") {
		addPlan(assistantToolActivityDraftGenerate, map[string]any{
			"topic": extractTopic(msg),
		})
	}

	if scene == assistantSceneOpsAdvisor || containsAny(msg, "统计", "数据", "完结率", "参与人数", "运营建议", "分析") {
		addPlan(assistantToolActivityStats, map[string]any{})
	}

	if containsAny(msg, "活动", "查询", "搜索", "报名", "招募") {
		searchInput := map[string]any{
			"keyword": extractKeyword(msg),
			"limit":   5,
		}
		if strings.Contains(msg, "报名中") {
			searchInput["status"] = model.ActivityStatusRecruiting
		} else if strings.Contains(msg, "已结束") {
			searchInput["status"] = model.ActivityStatusFinished
		} else if strings.Contains(msg, "已取消") {
			searchInput["status"] = model.ActivityStatusCanceled
		}
		addPlan(assistantToolActivitySearch, searchInput)
	}

	return plans
}

// Execute 执行单个工具，包含超时与重试
func (t *AssistantToolService) Execute(userID int64, plan assistantToolPlan) *assistantToolResult {
	result := &assistantToolResult{
		ToolName: plan.ToolName,
	}

	inputJSON, _ := json.Marshal(plan.Input)
	result.InputJSON = string(inputJSON)

	start := time.Now()
	var (
		output any
		err    error
	)
	// 工具执行失败时最多重试一次；权限/参数错误直接终止，避免无效重试。
	for attempt := 1; attempt <= assistantToolMaxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(t.ctx, assistantToolTimeout)
		output, err = t.executeOnce(ctx, userID, plan)
		cancel()
		if err == nil {
			break
		}
		if errors.Is(err, errToolPermissionDenied) || errors.Is(err, errToolInvalidInput) {
			break
		}
	}
	result.LatencyMS = int32(time.Since(start).Milliseconds())

	if err != nil {
		result.Success = false
		result.ErrorMsg = truncateText(err.Error(), 255)
		result.ErrorCode = classifyToolError(err)
		result.OutputJSON = "{}"
		return result
	}

	outputJSON, _ := json.Marshal(output)
	result.Success = true
	result.OutputJSON = string(outputJSON)
	if strings.TrimSpace(result.OutputJSON) == "" {
		result.OutputJSON = "{}"
	}
	return result
}

func (t *AssistantToolService) executeOnce(ctx context.Context, userID int64, plan assistantToolPlan) (any, error) {
	switch plan.ToolName {
	case assistantToolActivitySearch:
		return t.executeActivitySearch(ctx, plan.Input)
	case assistantToolActivityStats:
		return t.executeActivityStats(ctx, userID, plan.Input)
	case assistantToolActivityDraftGenerate:
		return t.executeActivityDraftGenerate(ctx, userID, plan.Input)
	default:
		return nil, fmt.Errorf("%w: 未知工具 %s", errToolInvalidInput, plan.ToolName)
	}
}

func (t *AssistantToolService) executeActivitySearch(ctx context.Context, input map[string]any) (any, error) {
	keyword := strings.TrimSpace(asString(input["keyword"]))
	status := asInt32(input["status"])
	limit := asInt(input["limit"], 5)
	rows, err := t.repo.SearchAssistantActivities(t.repo.DB, ctx, keyword, status, limit)
	if err != nil {
		return nil, err
	}

	list := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		list = append(list, map[string]any{
			"id":             row.ID,
			"org_id":         row.OrgID,
			"org_name":       row.OrgName,
			"title":          row.Title,
			"description":    row.Description,
			"start_time":     row.StartTime.Format("2006-01-02 15:04:05"),
			"end_time":       row.EndTime.Format("2006-01-02 15:04:05"),
			"location":       row.Location,
			"max_people":     row.MaxPeople,
			"current_people": row.CurrentPeople,
			"status":         row.Status,
		})
	}

	return map[string]any{
		"count": len(list),
		"list":  list,
	}, nil
}

func (t *AssistantToolService) executeActivityStats(ctx context.Context, userID int64, input map[string]any) (any, error) {
	manageableOrgIDs, err := t.repo.GetAssistantAccessibleOrgIDs(t.repo.DB, ctx, userID, model.MemberRoleManager)
	if err != nil {
		return nil, err
	}
	if len(manageableOrgIDs) == 0 {
		return nil, fmt.Errorf("%w: 仅组织管理员及以上可使用活动统计", errToolPermissionDenied)
	}

	// 未显式指定组织时，默认选第一个可管理组织，保证工具可直接执行。
	orgID := asInt64(input["org_id"])
	if orgID <= 0 {
		orgID = manageableOrgIDs[0]
	}
	if !containsInt64(manageableOrgIDs, orgID) {
		return nil, fmt.Errorf("%w: 无权访问该组织统计", errToolPermissionDenied)
	}

	organization, err := t.repo.GetAssistantOrganizationByIDWithContext(t.repo.DB, ctx, orgID)
	if err != nil {
		return nil, err
	}

	statusMap, err := t.repo.GetAssistantActivityStatusCountsByOrg(t.repo.DB, ctx, orgID)
	if err != nil {
		return nil, err
	}

	totalActivities := int64(0)
	recruitingCount := int64(0)
	finishedCount := int64(0)
	canceledCount := int64(0)
	for status, count := range statusMap {
		totalActivities += count
		switch status {
		case model.ActivityStatusRecruiting:
			recruitingCount = count
		case model.ActivityStatusFinished:
			finishedCount = count
		case model.ActivityStatusCanceled:
			canceledCount = count
		}
	}

	totalSignups, err := t.repo.CountAssistantSignupsByOrg(t.repo.DB, ctx, orgID)
	if err != nil {
		return nil, err
	}

	completionRate := 0.0
	if totalActivities > 0 {
		completionRate = float64(finishedCount) / float64(totalActivities)
	}

	avgSignups := 0.0
	if totalActivities > 0 {
		avgSignups = float64(totalSignups) / float64(totalActivities)
	}

	return map[string]any{
		"org_id":                  orgID,
		"org_name":                organization.OrgName,
		"total_activities":        totalActivities,
		"recruiting_activities":   recruitingCount,
		"finished_activities":     finishedCount,
		"canceled_activities":     canceledCount,
		"total_signups":           totalSignups,
		"avg_signup_per_activity": fmt.Sprintf("%.2f", avgSignups),
		"completion_rate":         fmt.Sprintf("%.2f", completionRate),
	}, nil
}

func (t *AssistantToolService) executeActivityDraftGenerate(ctx context.Context, userID int64, input map[string]any) (any, error) {
	accessibleOrgIDs, err := t.repo.GetAssistantAccessibleOrgIDs(t.repo.DB, ctx, userID, model.MemberRoleMember)
	if err != nil {
		return nil, err
	}
	if len(accessibleOrgIDs) == 0 {
		return nil, fmt.Errorf("%w: 仅组织成员可生成活动草案", errToolPermissionDenied)
	}

	orgID := asInt64(input["org_id"])
	if orgID <= 0 {
		orgID = accessibleOrgIDs[0]
	}
	if !containsInt64(accessibleOrgIDs, orgID) {
		return nil, fmt.Errorf("%w: 无权在该组织下生成活动草案", errToolPermissionDenied)
	}

	topic := strings.TrimSpace(asString(input["topic"]))
	if topic == "" {
		topic = "社区环保志愿服务"
	}
	targetPeople := strings.TrimSpace(asString(input["target_people"]))
	if targetPeople == "" {
		targetPeople = "社区居民与学生"
	}
	location := strings.TrimSpace(asString(input["location"]))
	if location == "" {
		location = "社区公共区域"
	}

	// 先返回结构化草案模板，再由大模型按上下文润色，降低空输出风险。
	title := fmt.Sprintf("%s行动", topic)
	summary := fmt.Sprintf("面向%s，在%s开展的公益活动，聚焦环境清洁、环保宣传与居民参与。", targetPeople, location)

	process := []string{
		"活动前一周发布招募通知并确认志愿者分工。",
		"活动当天开展签到分组、物资发放与安全说明。",
		"执行现场清洁、垃圾分类宣传与互动引导。",
		"活动结束后完成签退、物资回收和复盘记录。",
	}
	riskTips := []string{
		"提前准备应急药品、防暑/保暖物资。",
		"雨天或极端天气需准备改期预案。",
		"现场设置安全员，关注未成年人和老年人参与安全。",
	}

	return map[string]any{
		"org_id":        orgID,
		"title":         title,
		"summary":       summary,
		"target_people": targetPeople,
		"location":      location,
		"process":       process,
		"risk_tips":     riskTips,
	}, nil
}

func classifyToolError(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, context.DeadlineExceeded):
		return "TOOL_TIMEOUT"
	case errors.Is(err, errToolPermissionDenied):
		return "PERMISSION_DENIED"
	case errors.Is(err, errToolInvalidInput):
		return "INVALID_PARAMS"
	default:
		return "TOOL_EXEC_FAILED"
	}
}

func containsAny(text string, words ...string) bool {
	for _, word := range words {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

func extractTopic(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "环保志愿服务"
	}
	for _, token := range []string{"主题：", "主题:", "topic:"} {
		if idx := strings.Index(trimmed, token); idx >= 0 {
			value := strings.TrimSpace(trimmed[idx+len(token):])
			if value != "" {
				return truncateText(value, 30)
			}
		}
	}
	return truncateText(trimmed, 30)
}

func extractKeyword(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return ""
	}
	for _, token := range []string{"关键词：", "关键词:", "keyword:"} {
		if idx := strings.Index(trimmed, token); idx >= 0 {
			value := strings.TrimSpace(trimmed[idx+len(token):])
			if value != "" {
				return truncateText(value, 20)
			}
		}
	}
	if len(trimmed) > 20 {
		return trimmed[:20]
	}
	return trimmed
}

func containsInt64(items []int64, target int64) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func asString(v any) string {
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", val))
	}
}

func asInt64(v any) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	case float32:
		return int64(val)
	case float64:
		return int64(val)
	case json.Number:
		n, _ := val.Int64()
		return n
	default:
		return 0
	}
}

func asInt32(v any) int32 {
	return int32(asInt64(v))
}

func asInt(v any, fallback int) int {
	n := int(asInt64(v))
	if n == 0 {
		return fallback
	}
	return n
}
