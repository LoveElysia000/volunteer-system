package repository

import (
	"strings"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"

	"gorm.io/gorm"
)

// VolunteerExportRecord is the flattened row used by volunteer export.
type VolunteerExportRecord struct {
	VolunteerID  int64     `gorm:"column:volunteer_id"`
	RealName     string    `gorm:"column:real_name"`
	Gender       int32     `gorm:"column:gender"`
	Mobile       string    `gorm:"column:mobile"`
	Email        string    `gorm:"column:email"`
	OrgName      string    `gorm:"column:org_name"`
	TotalHours   float64   `gorm:"column:total_hours"`
	ServiceCount int32     `gorm:"column:service_count"`
	Status       int32     `gorm:"column:status"`
	AuditStatus  int32     `gorm:"column:audit_status"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

// ActivityExportRecord is the flattened row used by activity export.
type ActivityExportRecord struct {
	ActivityID    int64     `gorm:"column:activity_id"`
	Title         string    `gorm:"column:title"`
	Description   string    `gorm:"column:description"`
	StartTime     time.Time `gorm:"column:start_time"`
	EndTime       time.Time `gorm:"column:end_time"`
	Location      string    `gorm:"column:location"`
	Address       string    `gorm:"column:address"`
	Duration      float64   `gorm:"column:duration"`
	MaxPeople     int32     `gorm:"column:max_people"`
	CurrentPeople int32     `gorm:"column:current_people"`
	Status        int32     `gorm:"column:status"`
	OrgName       string    `gorm:"column:org_name"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

type OpsReportMetrics struct {
	ActivitiesCount int64
	SignupsCount    int64
	AttendanceCount int64
	WorkhoursCount  int64
}

func (r *Repository) getOrganizationNameForExport(db *gorm.DB, orgID int64) (string, error) {
	if orgID <= 0 {
		return "", nil
	}

	names := make([]string, 0, 1)
	if err := db.WithContext(r.ctx).
		Model(&model.Organization{}).
		Where("id = ?", orgID).
		Limit(1).
		Pluck("org_name", &names).Error; err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", nil
	}
	return names[0], nil
}

func (r *Repository) getActiveVolunteerIDsByOrgForExport(db *gorm.DB, orgID int64) ([]int64, error) {
	volunteerIDs := make([]int64, 0)
	if orgID <= 0 {
		return volunteerIDs, nil
	}

	err := db.WithContext(r.ctx).
		Model(&model.OrgMember{}).
		Where("org_id = ? AND status = ?", orgID, model.MemberStatusActive).
		Distinct("volunteer_id").
		Pluck("volunteer_id", &volunteerIDs).Error
	if err != nil {
		return nil, err
	}
	return volunteerIDs, nil
}

func (r *Repository) getActivityIDsByOrgForExport(db *gorm.DB, orgID int64) ([]int64, error) {
	activityIDs := make([]int64, 0)
	if orgID <= 0 {
		return activityIDs, nil
	}

	err := db.WithContext(r.ctx).
		Model(&model.Activity{}).
		Where("org_id = ?", orgID).
		Pluck("id", &activityIDs).Error
	if err != nil {
		return nil, err
	}
	return activityIDs, nil
}

// ListVolunteerExportRecords queries volunteer rows under current organization scope.
func (r *Repository) ListVolunteerExportRecords(
	db *gorm.DB,
	req *api.ExportVolunteersRequest,
	orgID int64,
	limit, offset int,
) ([]*VolunteerExportRecord, error) {
	rows := make([]*VolunteerExportRecord, 0)
	if limit <= 0 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	orgName, err := r.getOrganizationNameForExport(db, orgID)
	if err != nil {
		return nil, err
	}
	volunteerIDs, err := r.getActiveVolunteerIDsByOrgForExport(db, orgID)
	if err != nil {
		return nil, err
	}
	if len(volunteerIDs) == 0 {
		return rows, nil
	}

	query := db.WithContext(r.ctx).
		Table("volunteers AS v").
		Select(`
			v.id AS volunteer_id,
			v.real_name,
			v.gender,
			sa.mobile,
			sa.email,
			v.total_hours,
			v.service_count,
			v.status,
			v.audit_status,
			v.created_at
		`).
		Joins("INNER JOIN sys_accounts AS sa ON sa.id = v.account_id AND sa.deleted_at IS NULL").
		Where("v.id IN ?", volunteerIDs)

	if req != nil {
		if len(req.IdList) > 0 {
			query = query.Where("v.id IN ?", req.IdList)
		}
		keyword := strings.TrimSpace(req.Keyword)
		if keyword != "" {
			query = query.Where("v.real_name LIKE ?", "%"+keyword+"%")
		}
		if req.Status > 0 {
			query = query.Where("v.status = ?", req.Status)
		}
		if req.AuditStatus > 0 {
			query = query.Where("v.audit_status = ?", req.AuditStatus)
		}
	}

	err = query.
		Order("v.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		row.OrgName = orgName
	}
	return rows, nil
}

// ListActivityExportRecords queries activity rows under current organization scope.
func (r *Repository) ListActivityExportRecords(
	db *gorm.DB,
	req *api.ExportActivitiesRequest,
	orgID int64,
	startFrom, startTo *time.Time,
	limit, offset int,
) ([]*ActivityExportRecord, error) {
	rows := make([]*ActivityExportRecord, 0)
	if limit <= 0 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}
	orgName, err := r.getOrganizationNameForExport(db, orgID)
	if err != nil {
		return nil, err
	}

	query := db.WithContext(r.ctx).
		Table("activities AS a").
		Select(`
			a.id AS activity_id,
			a.title,
			a.description,
			a.start_time,
			a.end_time,
			a.location,
			a.address,
			a.duration,
			a.max_people,
			a.current_people,
			a.status,
			a.created_at
		`).
		Where("a.org_id = ?", orgID)

	if req != nil {
		if len(req.IdList) > 0 {
			query = query.Where("a.id IN ?", req.IdList)
		}
		keyword := strings.TrimSpace(req.Keyword)
		if keyword != "" {
			likeKeyword := "%" + keyword + "%"
			query = query.Where("(a.title LIKE ? OR a.description LIKE ?)", likeKeyword, likeKeyword)
		}
		if req.Status > 0 {
			query = query.Where("a.status = ?", req.Status)
		}
	}

	if startFrom != nil {
		query = query.Where("a.start_time >= ?", *startFrom)
	}
	if startTo != nil {
		query = query.Where("a.start_time <= ?", *startTo)
	}

	err = query.
		Order("a.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		row.OrgName = orgName
	}
	return rows, nil
}

func (r *Repository) GetOpsReportMetrics(
	db *gorm.DB,
	orgID int64,
	start, end time.Time,
) (*OpsReportMetrics, error) {
	metrics := &OpsReportMetrics{}

	activityQuery := db.WithContext(r.ctx).
		Model(&model.Activity{}).
		Where("org_id = ?", orgID).
		Where("start_time >= ? AND start_time <= ?", start, end)
	if err := activityQuery.Count(&metrics.ActivitiesCount).Error; err != nil {
		return nil, err
	}

	activityIDs, err := r.getActivityIDsByOrgForExport(db, orgID)
	if err != nil {
		return nil, err
	}
	if len(activityIDs) == 0 {
		return metrics, nil
	}

	signupQuery := db.WithContext(r.ctx).
		Model(&model.ActivitySignup{}).
		Where("activity_id IN ?", activityIDs).
		Where("signup_time >= ? AND signup_time <= ?", start, end)
	if err := signupQuery.Count(&metrics.SignupsCount).Error; err != nil {
		return nil, err
	}

	attendanceQuery := db.WithContext(r.ctx).
		Model(&model.ActivitySignup{}).
		Where("activity_id IN ?", activityIDs).
		Where("check_in_status = ?", model.ActivityCheckInDone).
		Where("signup_time >= ? AND signup_time <= ?", start, end)
	if err := attendanceQuery.Count(&metrics.AttendanceCount).Error; err != nil {
		return nil, err
	}

	workhourQuery := db.WithContext(r.ctx).
		Model(&model.ActivitySignup{}).
		Where("activity_id IN ?", activityIDs).
		Where("work_hour_status = ?", model.WorkHourStatusGranted).
		Where("signup_time >= ? AND signup_time <= ?", start, end)
	if err := workhourQuery.Count(&metrics.WorkhoursCount).Error; err != nil {
		return nil, err
	}

	return metrics, nil
}
