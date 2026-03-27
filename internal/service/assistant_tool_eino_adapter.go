package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	etoolutils "github.com/cloudwego/eino/components/tool/utils"
)

// assistant_tool_eino_adapter.go 负责把内部工具服务适配为 Eino Tool。
// 目标是隔离框架层与业务层，避免业务代码直接耦合 Eino 接口细节。

const (
	einoToolDescActivitySearch = "根据关键词与状态筛选活动，返回活动列表和数量。"
	einoToolDescActivityStats  = "查询组织活动统计数据，仅组织管理员可访问。"
	einoToolDescDraftGenerate  = "基于主题和组织信息生成活动草案结构。"
)

type einoActivitySearchInput struct {
	Keyword string `json:"keyword,omitempty"`
	Status  int32  `json:"status,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

type einoActivityStatsInput struct {
	OrgID int64 `json:"org_id,omitempty"`
}

type einoActivityDraftInput struct {
	OrgID        int64  `json:"org_id,omitempty"`
	Topic        string `json:"topic,omitempty"`
	TargetPeople string `json:"target_people,omitempty"`
	Location     string `json:"location,omitempty"`
}

type einoToolCollector struct {
	mu    sync.Mutex
	calls []runtimeToolCall
}

// Add 在每次工具执行后记录标准化结果，供 runtime 输出回传与 fallback 使用。
func (c *einoToolCollector) Add(result *assistantToolResult) {
	if c == nil || result == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, runtimeToolCall{
		ToolName:   result.ToolName,
		InputJSON:  nonEmptyJSON(result.InputJSON),
		OutputJSON: nonEmptyJSON(result.OutputJSON),
		Success:    result.Success,
		ErrorCode:  result.ErrorCode,
		ErrorMsg:   result.ErrorMsg,
		LatencyMS:  result.LatencyMS,
	})
}

// Snapshot 返回调用快照副本，避免外部修改内部状态。
func (c *einoToolCollector) Snapshot() []runtimeToolCall {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	items := make([]runtimeToolCall, len(c.calls))
	copy(items, c.calls)
	return items
}

// buildEinoTools 将业务工具声明为 Eino 可调用工具集合。
func (s *AssistantService) buildEinoTools(accountID int64, collector *einoToolCollector) ([]tool.BaseTool, error) {
	searchTool, err := etoolutils.InferTool(assistantToolActivitySearch, einoToolDescActivitySearch,
		func(ctx context.Context, input einoActivitySearchInput) (map[string]any, error) {
			// 仅透传非零值字段，减少无意义参数噪音。
			planInput := make(map[string]any)
			if keyword := strings.TrimSpace(input.Keyword); keyword != "" {
				planInput["keyword"] = keyword
			}
			if input.Status > 0 {
				planInput["status"] = input.Status
			}
			if input.Limit > 0 {
				planInput["limit"] = input.Limit
			}
			return s.executeEinoTool(ctx, accountID, collector, assistantToolPlan{
				ToolName: assistantToolActivitySearch,
				Input:    planInput,
			})
		})
	if err != nil {
		return nil, err
	}

	statsTool, err := etoolutils.InferTool(assistantToolActivityStats, einoToolDescActivityStats,
		func(ctx context.Context, input einoActivityStatsInput) (map[string]any, error) {
			planInput := make(map[string]any)
			if input.OrgID > 0 {
				planInput["org_id"] = input.OrgID
			}
			return s.executeEinoTool(ctx, accountID, collector, assistantToolPlan{
				ToolName: assistantToolActivityStats,
				Input:    planInput,
			})
		})
	if err != nil {
		return nil, err
	}

	draftTool, err := etoolutils.InferTool(assistantToolActivityDraftGenerate, einoToolDescDraftGenerate,
		func(ctx context.Context, input einoActivityDraftInput) (map[string]any, error) {
			planInput := make(map[string]any)
			if input.OrgID > 0 {
				planInput["org_id"] = input.OrgID
			}
			if topic := strings.TrimSpace(input.Topic); topic != "" {
				planInput["topic"] = topic
			}
			if targetPeople := strings.TrimSpace(input.TargetPeople); targetPeople != "" {
				planInput["target_people"] = targetPeople
			}
			if location := strings.TrimSpace(input.Location); location != "" {
				planInput["location"] = location
			}

			return s.executeEinoTool(ctx, accountID, collector, assistantToolPlan{
				ToolName: assistantToolActivityDraftGenerate,
				Input:    planInput,
			})
		})
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{searchTool, statsTool, draftTool}, nil
}

// executeEinoTool 执行业务工具并转换为 Eino 期望的 map 输出。
// 执行失败时返回 error 让模型感知失败状态，执行结果同时写入 collector。
func (s *AssistantService) executeEinoTool(ctx context.Context, accountID int64, collector *einoToolCollector, plan assistantToolPlan) (map[string]any, error) {
	// 工具实际执行交给业务工具服务，适配层只做结果格式转换。
	result := s.toolService.Execute(ctx, accountID, plan)
	collector.Add(result)

	if result == nil {
		return nil, fmt.Errorf("tool execution returned nil result")
	}
	if !result.Success {
		// Eino 侧只识别 error，因此把结构化错误码拼进错误文本。
		code := strings.TrimSpace(result.ErrorCode)
		if code == "" {
			code = "TOOL_EXEC_FAILED"
		}
		msg := strings.TrimSpace(result.ErrorMsg)
		if msg == "" {
			msg = "tool execution failed"
		}
		return nil, fmt.Errorf("%s: %s", code, msg)
	}

	payload := make(map[string]any)
	if err := json.Unmarshal([]byte(nonEmptyJSON(result.OutputJSON)), &payload); err != nil {
		// 解析失败时保留原文，避免丢失工具真实输出。
		payload["raw"] = nonEmptyJSON(result.OutputJSON)
		payload["decode_error"] = err.Error()
	}
	return payload, nil
}
