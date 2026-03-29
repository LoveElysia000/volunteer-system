package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

const (
	exportMaxRows = 50000
)

type ExportService struct {
	Service
}

func NewExportService(ctx context.Context, c *app.RequestContext) *ExportService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &ExportService{
		Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
}

// ExportVolunteers 导出志愿者
func (s *ExportService) ExportVolunteers(req *api.ExportVolunteersRequest) (*model.ExportFile, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}

	orgID, err := s.getCurrentOrgID()
	if err != nil {
		return nil, err
	}

	rows := make([]model.VolunteerExportRow, 0, 256)
	records, err := s.repo.ListVolunteerExportRecords(s.repo.DB, req, orgID, exportMaxRows+1, 0)
	if err != nil {
		log.Error("导出志愿者失败: 查询数据异常: %v, org_id=%d", err, orgID)
		return nil, err
	}
	if len(records) > exportMaxRows {
		return nil, fmt.Errorf("单次最多导出 %d 行，请缩小筛选范围", exportMaxRows)
	}

	for _, rec := range records {
		if rec == nil {
			continue
		}

		genderText := model.DefaultUnknownText
		if text, ok := model.VolunteerGenderTextByCode[rec.Gender]; ok {
			genderText = text
		}
		statusText := model.DefaultOtherText
		if text, ok := model.VolunteerStatusTextByCode[rec.Status]; ok {
			statusText = text
		}
		auditStatusText := model.DefaultUnknownText
		if text, ok := model.VolunteerAuditStatusTextByCode[rec.AuditStatus]; ok {
			auditStatusText = text
		}

		mobile := rec.Mobile
		if rec.Mobile != "" {
			if decrypted, decryptErr := util.DecryptSensitiveField(rec.Mobile); decryptErr == nil {
				mobile = decrypted
			}
		}

		rows = append(rows, model.VolunteerExportRow{
			VolunteerID:  rec.VolunteerID,
			RealName:     rec.RealName,
			Gender:       genderText,
			Mobile:       mobile,
			Email:        rec.Email,
			Organization: rec.OrgName,
			TotalHours:   rec.TotalHours,
			ServiceCount: rec.ServiceCount,
			Status:       statusText,
			AuditStatus:  auditStatusText,
			CreatedAt:    util.FormatDateTimeOrEmpty(rec.CreatedAt),
		})
	}

	content, err := util.MarshalXLSX(rows)
	if err != nil {
		return nil, err
	}

	return &model.ExportFile{
		FileName:    fmt.Sprintf("volunteers-export-%s.xlsx", time.Now().Format("20060102150405")),
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Content:     content,
	}, nil
}

// ExportActivities 导出活动
func (s *ExportService) ExportActivities(req *api.ExportActivitiesRequest) (*model.ExportFile, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}

	var startFrom *time.Time
	if strings.TrimSpace(req.StartFrom) != "" {
		value, parseErr := util.ParseDateFilterBound(strings.TrimSpace(req.StartFrom), false)
		if parseErr != nil {
			return nil, errors.New("开始时间格式错误")
		}
		startFrom = &value
	}

	var startTo *time.Time
	if strings.TrimSpace(req.StartTo) != "" {
		value, parseErr := util.ParseDateFilterBound(strings.TrimSpace(req.StartTo), true)
		if parseErr != nil {
			return nil, errors.New("结束时间格式错误")
		}
		startTo = &value
	}

	if startFrom != nil && startTo != nil && startTo.Before(*startFrom) {
		return nil, errors.New("结束时间不能早于开始时间")
	}

	orgID, err := s.getCurrentOrgID()
	if err != nil {
		return nil, err
	}

	rows := make([]model.ActivityExportRow, 0, 256)
	records, err := s.repo.ListActivityExportRecords(s.repo.DB, req, orgID, startFrom, startTo, exportMaxRows+1, 0)
	if err != nil {
		log.Error("导出活动失败: 查询数据异常: %v, org_id=%d", err, orgID)
		return nil, err
	}
	if len(records) > exportMaxRows {
		return nil, fmt.Errorf("单次最多导出 %d 行，请缩小筛选范围", exportMaxRows)
	}

	for _, rec := range records {
		if rec == nil {
			continue
		}

		statusText := model.DefaultUnknownText
		if text, ok := model.ActivityStatusTextByCode[rec.Status]; ok {
			statusText = text
		}

		rows = append(rows, model.ActivityExportRow{
			ActivityID:    rec.ActivityID,
			Title:         rec.Title,
			Description:   rec.Description,
			StartTime:     util.FormatDateTimeOrEmpty(rec.StartTime),
			EndTime:       util.FormatDateTimeOrEmpty(rec.EndTime),
			Location:      rec.Location,
			Address:       rec.Address,
			Duration:      rec.Duration,
			MaxPeople:     rec.MaxPeople,
			CurrentPeople: rec.CurrentPeople,
			Status:        statusText,
			Organization:  rec.OrgName,
			CreatedAt:     util.FormatDateTimeOrEmpty(rec.CreatedAt),
		})
	}

	content, err := util.MarshalXLSX(rows)
	if err != nil {
		return nil, err
	}

	return &model.ExportFile{
		FileName:    fmt.Sprintf("activities-export-%s.xlsx", time.Now().Format("20060102150405")),
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Content:     content,
	}, nil
}

// ExportOpsReport exports weekly/monthly operations report template.
func (s *ExportService) ExportOpsReport(req *api.ExportOpsReportRequest) (*model.ExportFile, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}
	if req.OrgId <= 0 {
		return nil, errors.New("组织ID不能为空")
	}
	periodType := strings.ToLower(strings.TrimSpace(req.PeriodType))
	if periodType != "weekly" && periodType != "monthly" {
		return nil, errors.New("periodType 仅支持 weekly/monthly")
	}

	start, end, err := resolveOpsReportRange(periodType, req.Start, req.End)
	if err != nil {
		return nil, err
	}

	operatorAccountID, err := s.currentAccountID()
	if err != nil {
		return nil, err
	}
	if err := s.requireOrgPermission(
		operatorAccountID,
		req.OrgId,
		model.PermissionResourceExport,
		model.PermissionActionManage,
	); err != nil {
		return nil, err
	}

	metrics, err := s.repo.GetOpsReportMetrics(s.repo.DB, req.OrgId, start, end)
	if err != nil {
		return nil, err
	}

	rows := []model.OpsReportExportRow{
		{
			PeriodType:      periodType,
			OrganizationID:  req.OrgId,
			Start:           util.FormatDateTimeOrEmpty(start),
			End:             util.FormatDateTimeOrEmpty(end),
			ActivitiesCount: metrics.ActivitiesCount,
			SignupsCount:    metrics.SignupsCount,
			AttendanceCount: metrics.AttendanceCount,
			WorkhoursCount:  metrics.WorkhoursCount,
		},
	}
	content, err := util.MarshalXLSXWithFrozenHeader(rows)
	if err != nil {
		return nil, err
	}

	return &model.ExportFile{
		FileName:    fmt.Sprintf("ops-report-%s-%s.xlsx", periodType, time.Now().Format("20060102150405")),
		ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		Content:     content,
	}, nil
}

func resolveOpsReportRange(periodType, rawStart, rawEnd string) (time.Time, time.Time, error) {
	now := time.Now()
	var start time.Time
	var end time.Time

	if strings.TrimSpace(rawStart) != "" {
		value, err := util.ParseDateFilterBound(strings.TrimSpace(rawStart), false)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("开始时间格式错误")
		}
		start = value
	} else if periodType == "weekly" {
		start = now.AddDate(0, 0, -7)
	} else {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}

	if strings.TrimSpace(rawEnd) != "" {
		value, err := util.ParseDateFilterBound(strings.TrimSpace(rawEnd), true)
		if err != nil {
			return time.Time{}, time.Time{}, errors.New("结束时间格式错误")
		}
		end = value
	} else {
		end = now
	}

	if end.Before(start) {
		return time.Time{}, time.Time{}, errors.New("结束时间不能早于开始时间")
	}
	return start, end, nil
}

func (s *ExportService) getCurrentOrgID() (int64, error) {
	accountID, err := s.currentAccountID()
	if err != nil {
		return 0, err
	}

	org, err := s.repo.GetOrganizationByAccountID(s.repo.DB, accountID)
	if err == nil && org != nil {
		if err := s.requireOrgPermission(
			accountID,
			org.ID,
			model.PermissionResourceExport,
			model.PermissionActionManage,
		); err != nil {
			return 0, err
		}
		return org.ID, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	orgIDs, err := s.repo.ListOrgScopeIDsByPermission(
		s.repo.DB,
		accountID,
		model.PermissionResourceExport,
		model.PermissionActionManage,
		2,
	)
	if err != nil {
		return 0, err
	}
	if len(orgIDs) == 1 {
		return orgIDs[0], nil
	}
	if len(orgIDs) > 1 {
		return 0, errors.New("存在多个组织作用域，请使用运营报表导出接口并指定组织ID")
	}

	if err := s.requireGlobalPermission(
		accountID,
		model.PermissionResourceExport,
		model.PermissionActionManage,
	); err == nil {
		return 0, errors.New("全局导出权限需要显式指定组织ID，请使用运营报表导出接口")
	}
	return 0, errors.New("无权导出数据")
}
