package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/internal/repository"
	"volunteer-system/pkg/util"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ImportService struct {
	Service
}

type volunteerImportRow struct {
	UserName string
	RealName string
	Mobile   string
	Email    string
	Password string
	Gender   string
}

type activityImportRow struct {
	OrgID       int64
	Title       string
	Description string
	StartTime   string
	EndTime     string
	Location    string
	Address     string
	Duration    string
	MaxPeople   string
	CoverURL    string
}

type importFailure struct {
	RowNumber int32
	Reason    string
	Raw       string
}

var (
	importMobilePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)
	importEmailPattern  = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

func NewImportService(ctx context.Context, c *app.RequestContext) *ImportService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &ImportService{
		Service{
			ctx:  ctx,
			c:    c,
			repo: repository.NewRepository(ctx, c),
		},
	}
}

func (s *ImportService) ImportVolunteers(filename string, content []byte) (*api.ImportResultResponse, error) {
	if len(content) == 0 {
		return nil, errors.New("导入文件不能为空")
	}
	operatorID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}
	if err := s.requireAnyOrganizationManagePermission(operatorID); err != nil {
		return nil, err
	}

	rows, err := parseVolunteerImportFile(filename, content)
	if err != nil {
		return nil, err
	}

	failures := make([]importFailure, 0)
	successCount := 0
	for index, row := range rows {
		rowNumber := int32(index + 2)
		if err := s.importVolunteerRow(row); err != nil {
			failures = append(failures, importFailure{
				RowNumber: rowNumber,
				Reason:    err.Error(),
				Raw:       marshalVolunteerImportRow(row),
			})
			continue
		}
		successCount++
	}

	return buildImportResultResponse(failures, len(rows), successCount)
}

func (s *ImportService) ImportActivities(filename string, content []byte) (*api.ImportResultResponse, error) {
	if len(content) == 0 {
		return nil, errors.New("导入文件不能为空")
	}
	operatorID, err := middleware.GetUserIDInt(s.c)
	if err != nil {
		return nil, err
	}

	rows, err := parseActivityImportFile(filename, content)
	if err != nil {
		return nil, err
	}

	failures := make([]importFailure, 0)
	successCount := 0
	for index, row := range rows {
		rowNumber := int32(index + 2)
		if err := s.importActivityRow(operatorID, row); err != nil {
			failures = append(failures, importFailure{
				RowNumber: rowNumber,
				Reason:    err.Error(),
				Raw:       marshalActivityImportRow(row),
			})
			continue
		}
		successCount++
	}

	return buildImportResultResponse(failures, len(rows), successCount)
}

func parseVolunteerImportFile(filename string, content []byte) ([]volunteerImportRow, error) {
	rows, err := parseImportSheetRows(filename, content)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []volunteerImportRow{}, nil
	}

	result := make([]volunteerImportRow, 0, len(rows))
	for _, row := range rows {
		if len(row) == 0 || isImportRowEmpty(row) {
			continue
		}
		result = append(result, volunteerImportRow{
			UserName: importCell(row, 0),
			RealName: importCell(row, 1),
			Mobile:   importCell(row, 2),
			Email:    importCell(row, 3),
			Password: importCell(row, 4),
			Gender:   importCell(row, 5),
		})
	}
	return result, nil
}

func parseActivityImportFile(filename string, content []byte) ([]activityImportRow, error) {
	rows, err := parseImportSheetRows(filename, content)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []activityImportRow{}, nil
	}

	result := make([]activityImportRow, 0, len(rows))
	for _, row := range rows {
		if len(row) == 0 || isImportRowEmpty(row) {
			continue
		}
		orgIDText := importCell(row, 0)
		orgID, parseErr := strconv.ParseInt(orgIDText, 10, 64)
		if parseErr != nil {
			orgID = 0
		}
		result = append(result, activityImportRow{
			OrgID:       orgID,
			Title:       importCell(row, 1),
			Description: importCell(row, 2),
			StartTime:   importCell(row, 3),
			EndTime:     importCell(row, 4),
			Location:    importCell(row, 5),
			Address:     importCell(row, 6),
			Duration:    importCell(row, 7),
			MaxPeople:   importCell(row, 8),
			CoverURL:    importCell(row, 9),
		})
	}
	return result, nil
}

func buildImportFailureCSV(failures []importFailure) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write([]string{"row_number", "reason", "raw"}); err != nil {
		return nil, err
	}
	for _, failure := range failures {
		if err := writer.Write([]string{
			strconv.FormatInt(int64(failure.RowNumber), 10),
			failure.Reason,
			failure.Raw,
		}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func buildImportResultResponse(failures []importFailure, totalCount, successCount int) (*api.ImportResultResponse, error) {
	resp := &api.ImportResultResponse{
		TotalCount:   int32(totalCount),
		SuccessCount: int32(successCount),
		FailedCount:  int32(len(failures)),
		Failures:     make([]*api.ImportFailureItem, 0, len(failures)),
	}
	for _, failure := range failures {
		resp.Failures = append(resp.Failures, &api.ImportFailureItem{
			RowNumber: failure.RowNumber,
			Reason:    failure.Reason,
			Raw:       failure.Raw,
		})
	}
	if len(failures) == 0 {
		return resp, nil
	}
	content, err := buildImportFailureCSV(failures)
	if err != nil {
		return nil, err
	}
	resp.ErrorFileName = "import-errors.csv"
	resp.ErrorFileContentType = "text/csv; charset=utf-8"
	resp.ErrorFileContent = content
	return resp, nil
}

func parseImportSheetRows(filename string, content []byte) ([][]string, error) {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(filename)))
	switch ext {
	case ".csv":
		return parseCSVImportRows(content)
	case ".xlsx":
		return parseXLSXImportRows(content)
	default:
		return nil, fmt.Errorf("不支持的导入文件格式: %s", ext)
	}
}

func parseCSVImportRows(content []byte) ([][]string, error) {
	reader := csv.NewReader(bytes.NewReader(content))
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	rows := make([][]string, 0, 16)
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		rows = append(rows, trimImportRecord(record))
	}
	if len(rows) <= 1 {
		return [][]string{}, nil
	}
	return rows[1:], nil
}

func parseXLSXImportRows(content []byte) ([][]string, error) {
	file, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	sheetName := file.GetSheetName(file.GetActiveSheetIndex())
	if sheetName == "" {
		return nil, errors.New("excel 工作表不存在")
	}
	rows, err := file.GetRows(sheetName)
	if err != nil {
		return nil, err
	}
	if len(rows) <= 1 {
		return [][]string{}, nil
	}

	result := make([][]string, 0, len(rows)-1)
	for _, row := range rows[1:] {
		result = append(result, trimImportRecord(row))
	}
	return result, nil
}

func trimImportRecord(record []string) []string {
	result := make([]string, 0, len(record))
	for _, item := range record {
		result = append(result, strings.TrimSpace(item))
	}
	return result
}

func isImportRowEmpty(row []string) bool {
	for _, item := range row {
		if strings.TrimSpace(item) != "" {
			return false
		}
	}
	return true
}

func importCell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func (s *ImportService) requireAnyOrganizationManagePermission(operatorID int64) error {
	hasGlobal, err := s.hasPermissionByScope(
		operatorID,
		model.RBACScopeGlobal,
		0,
		model.PermissionResourceOrganization,
		model.PermissionActionManage,
	)
	if err != nil {
		return err
	}
	if hasGlobal {
		return nil
	}

	hasAnyOrg, err := s.repo.HasAnyOrgPermission(
		s.repo.DB,
		operatorID,
		model.PermissionResourceOrganization,
		model.PermissionActionManage,
	)
	if err != nil {
		return err
	}
	if !hasAnyOrg {
		return errForbidden("无权执行导入")
	}
	return nil
}

func (s *ImportService) importVolunteerRow(row volunteerImportRow) error {
	if err := validateVolunteerImportRow(row); err != nil {
		return err
	}

	mobilePair, err := util.ProcessSensitiveField(row.Mobile)
	if err != nil {
		return errors.New("手机号处理失败")
	}
	email := strings.TrimSpace(strings.ToLower(row.Email))

	exists, err := s.repo.CheckMobileExists(s.repo.DB, mobilePair.Hash)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("手机号已存在")
	}
	exists, err = s.repo.CheckEmailExists(s.repo.DB, email)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("邮箱已存在")
	}

	hashedPassword, err := util.HashPassword(strings.TrimSpace(row.Password))
	if err != nil {
		return errors.New("密码加密失败")
	}
	genderCode := model.GenderCodeByText[model.DefaultUnknownText]
	if strings.TrimSpace(row.Gender) != "" {
		genderCode = model.GenderCodeByText[strings.TrimSpace(row.Gender)]
	}

	account := &model.SysAccount{}
	err = s.withTransaction(func(tx *gorm.DB) error {
		account.Username = strings.TrimSpace(row.UserName)
		account.Mobile = mobilePair.Encrypted
		account.MobileHash = mobilePair.Hash
		account.Email = email
		account.Password = hashedPassword
		account.IdentityType = model.RegisterTypeVolunteerCode
		account.Status = model.SysAccountNormal
		account.CreatedAt = time.Now()

		if err := s.repo.CreateAccount(tx, account); err != nil {
			return err
		}
		return s.repo.CreateVolunteer(tx, &model.Volunteer{
			AccountID:   account.ID,
			RealName:    strings.TrimSpace(row.RealName),
			Gender:      genderCode,
			Status:      model.VolunteerActiveStatus,
			AuditStatus: model.VolunteerAuditStatusUnverified,
			CreatedAt:   time.Now(),
		})
	})
	if err != nil {
		if util.IsDuplicateEntryErr(err) {
			return errors.New("账号信息已存在")
		}
		return err
	}
	if bindErr := s.ensureDefaultRBACBinding(s.repo.DB, account); bindErr != nil {
		log.Warn("志愿者导入默认角色绑定失败(不影响导入): account_id=%d err=%v", account.ID, bindErr)
	}
	return nil
}

func (s *ImportService) importActivityRow(operatorID int64, row activityImportRow) error {
	activity, err := validateActivityImportRow(row)
	if err != nil {
		return err
	}
	if err := s.requireOrgPermission(
		operatorID,
		activity.OrgID,
		model.PermissionResourceOrganization,
		model.PermissionActionManage,
	); err != nil {
		return err
	}
	if _, err := s.repo.GetOrganizationByID(s.repo.DB, activity.OrgID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("组织不存在")
		}
		return err
	}
	return s.repo.CreateActivity(s.repo.DB, activity)
}

func validateVolunteerImportRow(row volunteerImportRow) error {
	row.UserName = strings.TrimSpace(row.UserName)
	row.RealName = strings.TrimSpace(row.RealName)
	row.Mobile = strings.TrimSpace(row.Mobile)
	row.Email = strings.TrimSpace(strings.ToLower(row.Email))
	row.Password = strings.TrimSpace(row.Password)
	row.Gender = strings.TrimSpace(row.Gender)

	if row.UserName == "" {
		return errors.New("用户名不能为空")
	}
	if row.RealName == "" {
		return errors.New("姓名不能为空")
	}
	if !importMobilePattern.MatchString(row.Mobile) {
		return errors.New("手机号格式不正确")
	}
	if !importEmailPattern.MatchString(row.Email) {
		return errors.New("邮箱格式不正确")
	}
	if err := util.ValidatePasswordStrength(row.Password); err != nil {
		return err
	}
	if row.Gender != "" {
		if _, ok := model.GenderCodeByText[row.Gender]; !ok {
			return errors.New("性别格式不正确")
		}
	}
	return nil
}

func validateActivityImportRow(row activityImportRow) (*model.Activity, error) {
	if row.OrgID <= 0 {
		return nil, errors.New("组织ID不能为空")
	}
	title := strings.TrimSpace(row.Title)
	if title == "" {
		return nil, errors.New("活动标题不能为空")
	}
	startTime, err := util.ParseDateTime(strings.TrimSpace(row.StartTime))
	if err != nil {
		return nil, errors.New("开始时间格式错误")
	}
	endTime, err := util.ParseDateTime(strings.TrimSpace(row.EndTime))
	if err != nil {
		return nil, errors.New("结束时间格式错误")
	}
	if !endTime.After(startTime) {
		return nil, errors.New("结束时间必须晚于开始时间")
	}
	duration, err := strconv.ParseFloat(strings.TrimSpace(row.Duration), 64)
	if err != nil || duration < 0 {
		return nil, errors.New("预估工时格式错误")
	}
	maxPeople, err := strconv.ParseInt(strings.TrimSpace(row.MaxPeople), 10, 32)
	if err != nil || maxPeople < 0 {
		return nil, errors.New("最大人数格式错误")
	}

	return &model.Activity{
		OrgID:         row.OrgID,
		Title:         title,
		Description:   strings.TrimSpace(row.Description),
		CoverURL:      strings.TrimSpace(row.CoverURL),
		StartTime:     startTime,
		EndTime:       endTime,
		Location:      strings.TrimSpace(row.Location),
		Address:       strings.TrimSpace(row.Address),
		Duration:      duration,
		MaxPeople:     int32(maxPeople),
		CurrentPeople: 0,
		Status:        model.ActivityStatusRecruiting,
		CreatedAt:     time.Now(),
	}, nil
}

func marshalVolunteerImportRow(row volunteerImportRow) string {
	return strings.Join([]string{
		row.UserName,
		row.RealName,
		row.Mobile,
		row.Email,
		row.Password,
		row.Gender,
	}, ",")
}

func marshalActivityImportRow(row activityImportRow) string {
	return strings.Join([]string{
		strconv.FormatInt(row.OrgID, 10),
		row.Title,
		row.Description,
		row.StartTime,
		row.EndTime,
		row.Location,
		row.Address,
		row.Duration,
		row.MaxPeople,
		row.CoverURL,
	}, ",")
}
