package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"volunteer-system/config"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
)

// assistant_tool_service.go 实现 AI 助手可控工具链服务。
//
// 该文件负责把“用户问题”转换为“可执行的内部工具调用”：
// 1. 基于场景和关键词进行规则化工具规划（PlanTools），避免模型直接决策工具链。
// 2. 执行工具调用并统一输出结构（Execute），包含超时控制、有限重试与错误分类。
// 3. 在工具层做权限与参数校验，确保组织数据访问边界清晰。
// 4. 输出稳定的 JSON 结果，供上层消息上下文与工具日志复用。
// 5. 提供输入提取与类型转换辅助函数，降低上层编排复杂度。

const (
	assistantToolActivitySearch        = "activity_search"
	assistantToolActivityStats         = "activity_stats"
	assistantToolActivityDraftGenerate = "activity_draft_generate"

	assistantToolMaxPlans       = 3
	assistantToolDefaultTimeout = 3 * time.Second
	assistantToolMinTimeout     = 300 * time.Millisecond
	assistantToolMaxTimeout     = 30 * time.Second
	assistantToolMaxAttempts    = 2 // 首次 + 1 次重试
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
		if len(plans) >= assistantToolMaxPlans {
			// 单轮最多执行有限个工具，防止模型触发过长工具链。
			return
		}
		if _, ok := added[tool]; ok {
			// 去重，避免同一工具重复计划。
			return
		}
		added[tool] = struct{}{}
		plans = append(plans, assistantToolPlan{ToolName: tool, Input: input})
	}

	// 当前为规则启发式规划：优先保证可控性，避免模型直接决定工具链。
	if scene == assistantSceneActivityDraft || util.ContainsAny(msg, "草案", "活动方案", "活动策划", "活动发布") {
		addPlan(assistantToolActivityDraftGenerate, map[string]any{
			"topic": extractTopic(msg),
		})
	}

	if scene == assistantSceneOpsAdvisor || util.ContainsAny(msg, "统计", "数据", "完结率", "参与人数", "运营建议", "分析") {
		addPlan(assistantToolActivityStats, map[string]any{})
	}

	if shouldPlanActivitySearch(msg) {
		searchInput := map[string]any{
			"keyword": extractKeyword(msg),
			"limit":   5,
		}
		// 对自然语言状态词做映射，提升检索命中率。
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
func (t *AssistantToolService) Execute(ctx context.Context, userID int64, plan assistantToolPlan) *assistantToolResult {
	result := &assistantToolResult{
		ToolName: plan.ToolName,
	}

	inputJSON, _ := json.Marshal(plan.Input)
	// 记录原始输入，便于工具日志和问题排查。
	result.InputJSON = string(inputJSON)

	start := time.Now()
	toolTimeout := resolveEinoToolTimeout(t.getAIConfig())
	baseCtx := ctx
	if baseCtx == nil {
		baseCtx = t.ctx
	}
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	var (
		output any
		err    error
	)
	// 工具执行失败时最多重试一次；权限/参数错误直接终止，避免无效重试。
	for attempt := 1; attempt <= assistantToolMaxAttempts; attempt++ {
		// 每次尝试都基于同一个父上下文派生独立超时，确保可被上层取消信号中断。
		attemptCtx, cancel := context.WithTimeout(baseCtx, toolTimeout)
		output, err = t.executeOnce(attemptCtx, userID, plan)
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
		result.ErrorMsg = util.TruncateText(err.Error(), 255)
		result.ErrorCode = classifyToolError(err)
		result.OutputJSON = "{}"
		return result
	}

	// 成功时强制输出 JSON 字符串，保障上层解析稳定性。
	outputJSON, _ := json.Marshal(output)
	result.Success = true
	result.OutputJSON = string(outputJSON)
	if strings.TrimSpace(result.OutputJSON) == "" {
		result.OutputJSON = "{}"
	}
	return result
}

func (t *AssistantToolService) getAIConfig() *config.AIConfig {
	cfg := config.GetConfig()
	if cfg == nil {
		return nil
	}
	return cfg.AI
}

func resolveEinoToolTimeout(cfg *config.AIConfig) time.Duration {
	if cfg == nil || cfg.Eino.ToolTimeoutMS <= 0 {
		return assistantToolDefaultTimeout
	}

	// 通过上下限裁剪，避免配置误填导致工具执行过快超时或超长阻塞。
	timeout := time.Duration(cfg.Eino.ToolTimeoutMS) * time.Millisecond
	if timeout < assistantToolMinTimeout {
		return assistantToolMinTimeout
	}
	if timeout > assistantToolMaxTimeout {
		return assistantToolMaxTimeout
	}
	return timeout
}

// executeOnce 仅负责路由到具体工具实现，不承担重试与超时控制。
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

// executeActivitySearch 面向普通问答检索活动列表，返回结构化列表数据。
func (t *AssistantToolService) executeActivitySearch(ctx context.Context, input map[string]any) (any, error) {
	// 做轻量参数归一化，兜底 limit 默认值。
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
		// 转换为统一输出字段，避免泄漏数据库内部列结构。
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

// executeActivityStats 面向运营分析，含组织权限校验与统计聚合。
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
	if !util.ContainsInt64(manageableOrgIDs, orgID) {
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
		// 避免除零并输出可解释的比例指标。
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

// executeActivityDraftGenerate 先生成稳定草案骨架，再交给模型在上下文中润色。
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
	if !util.ContainsInt64(accessibleOrgIDs, orgID) {
		return nil, fmt.Errorf("%w: 无权在该组织下生成活动草案", errToolPermissionDenied)
	}

	topic := strings.TrimSpace(asString(input["topic"]))
	if topic == "" {
		// 补默认主题，防止模型未传参时返回空草案。
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

// classifyToolError 将执行错误映射为可观测的标准错误码，便于上层统一处理。
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

func extractTopic(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return "环保志愿服务"
	}
	for _, token := range []string{"主题：", "主题:", "topic:"} {
		if idx := strings.Index(trimmed, token); idx >= 0 {
			value := trimByStopWords(strings.TrimSpace(trimmed[idx+len(token):]))
			if value != "" {
				return util.TruncateText(value, 30)
			}
		}
	}
	return util.TruncateText(trimByStopWords(trimmed), 30)
}

func extractKeyword(message string) string {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		return ""
	}
	for _, token := range []string{"关键词：", "关键词:", "keyword:"} {
		if idx := strings.Index(trimmed, token); idx >= 0 {
			value := trimByStopWords(strings.TrimSpace(trimmed[idx+len(token):]))
			if value != "" {
				return util.TruncateText(value, 20)
			}
		}
	}
	if len(trimmed) > 20 {
		return trimmed[:20]
	}
	return trimmed
}

func asString(v any) string {
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case nil:
		return ""
	default:
		// 兜底做字符串化，兼容 json.Number/数字等输入。
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
	if n <= 0 {
		// 工具参数中的非法或空值回退到默认值。
		return fallback
	}
	return n
}

func shouldPlanActivitySearch(msg string) bool {
	// 关键词启发式：优先捕获检索类意图。
	if util.ContainsAny(msg, "查询", "搜索", "招募", "报名", "报名中", "已结束", "已取消") {
		return true
	}
	if util.ContainsAny(msg, "有哪些活动", "什么活动", "活动列表", "最近活动", "活动推荐") {
		return true
	}
	return strings.Contains(msg, "活动") && util.ContainsAny(msg, "最近", "有哪些", "什么", "推荐")
}

func trimByStopWords(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}

	stops := []string{"。", "；", ";", "\n"}
	cut := len(value)
	for _, stop := range stops {
		if idx := strings.Index(value, stop); idx >= 0 && idx < cut {
			cut = idx
		}
	}
	return strings.TrimSpace(value[:cut])
}
