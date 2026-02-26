package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/gorm"
)

const (
	exportPageSize = 500
	exportMaxRows  = 50000
)

// ExportFile 导出文件内容
type ExportFile struct {
	FileName    string
	ContentType string
	Content     []byte
}

type volunteerExportRow struct {
	VolunteerID  int64   `csv:"志愿者ID"`
	RealName     string  `csv:"姓名"`
	Gender       string  `csv:"性别"`
	Mobile       string  `csv:"手机号"`
	Email        string  `csv:"邮箱"`
	Organization string  `csv:"所属组织"`
	TotalHours   float64 `csv:"累计工时"`
	ServiceCount int32   `csv:"服务次数"`
	Status       string  `csv:"志愿者状态"`
	AuditStatus  string  `csv:"实名状态"`
	CreatedAt    string  `csv:"创建时间"`
}

type activityExportRow struct {
	ActivityID    int64   `csv:"活动ID"`
	Title         string  `csv:"标题"`
	Description   string  `csv:"描述"`
	StartTime     string  `csv:"开始时间"`
	EndTime       string  `csv:"结束时间"`
	Location      string  `csv:"地点"`
	Address       string  `csv:"地址"`
	Duration      float64 `csv:"预估工时"`
	MaxPeople     int32   `csv:"最大人数"`
	CurrentPeople int32   `csv:"当前人数"`
	Status        string  `csv:"活动状态"`
	Organization  string  `csv:"发布组织"`
	CreatedAt     string  `csv:"创建时间"`
}

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
func (s *ExportService) ExportVolunteers(req *api.ExportVolunteersRequest) (*ExportFile, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}

	format, err := normalizeExportFormat(req.Format)
	if err != nil {
		return nil, err
	}

	orgID, err := s.getCurrentOrgID()
	if err != nil {
		return nil, err
	}

	rows := make([]volunteerExportRow, 0, 256)
	offset := 0

	for {
		records, err := s.repo.ListVolunteerExportRecords(s.repo.DB, req, orgID, exportPageSize, offset)
		if err != nil {
			log.Error("导出志愿者失败: 查询数据异常: %v, org_id=%d offset=%d", err, orgID, offset)
			return nil, err
		}
		if len(records) == 0 {
			break
		}

		for _, rec := range records {
			if rec == nil {
				continue
			}

			mobile := rec.Mobile
			if rec.Mobile != "" {
				if decrypted, decryptErr := util.DecryptSensitiveField(rec.Mobile); decryptErr == nil {
					mobile = decrypted
				}
			}

			// 安全优先：默认脱敏导出手机号。
			mobile = util.GetMobileMask(mobile)

			rows = append(rows, volunteerExportRow{
				VolunteerID:  rec.VolunteerID,
				RealName:     rec.RealName,
				Gender:       volunteerGenderText(rec.Gender),
				Mobile:       mobile,
				Email:        rec.Email,
				Organization: rec.OrgName,
				TotalHours:   rec.TotalHours,
				ServiceCount: rec.ServiceCount,
				Status:       volunteerStatusText(rec.Status),
				AuditStatus:  volunteerAuditStatusText(rec.AuditStatus),
				CreatedAt:    util.FormatDateTimeOrEmpty(rec.CreatedAt),
			})
		}

		if len(rows) > exportMaxRows {
			return nil, fmt.Errorf("单次最多导出 %d 行，请缩小筛选范围", exportMaxRows)
		}
		offset += exportPageSize
	}

	content, contentType, err := buildExportContent(format, rows)
	if err != nil {
		return nil, err
	}

	return &ExportFile{
		FileName:    fmt.Sprintf("volunteers-export-%s.%s", time.Now().Format("20060102150405"), format),
		ContentType: contentType,
		Content:     content,
	}, nil
}

// ExportActivities 导出活动
func (s *ExportService) ExportActivities(req *api.ExportActivitiesRequest) (*ExportFile, error) {
	if req == nil {
		return nil, errors.New("请求不能为空")
	}

	format, err := normalizeExportFormat(req.Format)
	if err != nil {
		return nil, err
	}

	var startFrom *time.Time
	if strings.TrimSpace(req.StartFrom) != "" {
		value, parseErr := util.ParseDateTime(strings.TrimSpace(req.StartFrom))
		if parseErr != nil {
			return nil, errors.New("开始时间格式错误")
		}
		startFrom = &value
	}

	var startTo *time.Time
	if strings.TrimSpace(req.StartTo) != "" {
		value, parseErr := util.ParseDateTime(strings.TrimSpace(req.StartTo))
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

	rows := make([]activityExportRow, 0, 256)
	offset := 0

	for {
		records, err := s.repo.ListActivityExportRecords(s.repo.DB, req, orgID, startFrom, startTo, exportPageSize, offset)
		if err != nil {
			log.Error("导出活动失败: 查询数据异常: %v, org_id=%d offset=%d", err, orgID, offset)
			return nil, err
		}
		if len(records) == 0 {
			break
		}

		for _, rec := range records {
			if rec == nil {
				continue
			}

			rows = append(rows, activityExportRow{
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
				Status:        activityStatusText(rec.Status),
				Organization:  rec.OrgName,
				CreatedAt:     util.FormatDateTimeOrEmpty(rec.CreatedAt),
			})
		}

		if len(rows) > exportMaxRows {
			return nil, fmt.Errorf("单次最多导出 %d 行，请缩小筛选范围", exportMaxRows)
		}
		offset += exportPageSize
	}

	content, contentType, err := buildExportContent(format, rows)
	if err != nil {
		return nil, err
	}

	return &ExportFile{
		FileName:    fmt.Sprintf("activities-export-%s.%s", time.Now().Format("20060102150405"), format),
		ContentType: contentType,
		Content:     content,
	}, nil
}

func buildExportContent(format string, rows interface{}) ([]byte, string, error) {
	switch format {
	case "csv":
		content, err := util.MarshalCSV(rows)
		if err != nil {
			return nil, "", err
		}
		return content, "text/csv; charset=utf-8", nil
	case "xlsx":
		return nil, "", errors.New("当前版本暂不支持 xlsx 导出")
	default:
		return nil, "", errors.New("不支持的导出格式")
	}
}

func normalizeExportFormat(format string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(format))
	if value == "" {
		return "csv", nil
	}
	switch value {
	case "csv", "xlsx":
		return value, nil
	default:
		return "", errors.New("导出格式仅支持 csv/xlsx")
	}
}

func (s *ExportService) getCurrentOrgID() (int64, error) {
	userID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return 0, err
	}

	org, err := s.repo.GetOrganizationByAccountID(s.repo.DB, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errors.New("组织信息不存在")
		}
		return 0, err
	}
	return org.ID, nil
}

func volunteerGenderText(gender int32) string {
	switch gender {
	case 1:
		return "男"
	case 2:
		return "女"
	default:
		return "未知"
	}
}

func volunteerStatusText(status int32) string {
	switch status {
	case model.VolunteerActiveStatus:
		return "活跃"
	case model.VolunteerInactiveStatus:
		return "非活跃"
	default:
		return "其他"
	}
}

func volunteerAuditStatusText(status int32) string {
	switch status {
	case model.VolunteerAuditStatusUnverified:
		return "未认证"
	case model.VolunteerAuditStatusPending:
		return "审核中"
	case model.VolunteerAuditStatusApproved:
		return "已通过"
	case model.VolunteerAuditStatusRejected:
		return "已驳回"
	default:
		return "未知"
	}
}

func activityStatusText(status int32) string {
	switch status {
	case model.ActivityStatusRecruiting:
		return "报名中"
	case model.ActivityStatusFinished:
		return "已结束"
	case model.ActivityStatusCanceled:
		return "已取消"
	default:
		return "未知"
	}
}
