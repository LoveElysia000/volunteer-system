package repository

import (
	"time"
	"volunteer-system/internal/model"

	"gorm.io/gorm"
)

type OrgFunnelMetrics struct {
	RegistrationCount int64
	MembershipCount   int64
	SignupCount       int64
	AttendanceCount   int64
	WorkhourCount     int64
}

type OpsDashboardMetrics struct {
	SignupCount         int64
	ApprovedSignupCount int64
	AttendanceCount     int64
	GrantedWorkHours    float64
}

func (r *Repository) GetOrgFunnelMetrics(db *gorm.DB, orgID int64, start, end *time.Time) (*OrgFunnelMetrics, error) {
	metrics := &OrgFunnelMetrics{}

	volunteerQuery := db.WithContext(r.ctx).Model(&model.Volunteer{})
	if start != nil {
		volunteerQuery = volunteerQuery.Where("created_at >= ?", *start)
	}
	if end != nil {
		volunteerQuery = volunteerQuery.Where("created_at <= ?", *end)
	}
	if err := volunteerQuery.Count(&metrics.RegistrationCount).Error; err != nil {
		return nil, err
	}

	memberQuery := db.WithContext(r.ctx).
		Model(&model.OrgMember{}).
		Where("org_id = ? AND status = ?", orgID, model.MemberStatusActive)
	if start != nil {
		memberQuery = memberQuery.Where("created_at >= ?", *start)
	}
	if end != nil {
		memberQuery = memberQuery.Where("created_at <= ?", *end)
	}
	if err := memberQuery.Count(&metrics.MembershipCount).Error; err != nil {
		return nil, err
	}

	activityIDs := make([]int64, 0)
	if err := db.WithContext(r.ctx).
		Model(&model.Activity{}).
		Where("org_id = ?", orgID).
		Pluck("id", &activityIDs).Error; err != nil {
		return nil, err
	}
	if len(activityIDs) == 0 {
		return metrics, nil
	}

	signupQuery := db.WithContext(r.ctx).
		Model(&model.ActivitySignup{}).
		Where("activity_id IN ?", activityIDs).
		Where("status <> ?", model.ActivitySignupStatusCanceled)
	if start != nil {
		signupQuery = signupQuery.Where("signup_time >= ?", *start)
	}
	if end != nil {
		signupQuery = signupQuery.Where("signup_time <= ?", *end)
	}
	if err := signupQuery.Count(&metrics.SignupCount).Error; err != nil {
		return nil, err
	}

	attendanceQuery := db.WithContext(r.ctx).
		Model(&model.ActivitySignup{}).
		Where("activity_id IN ?", activityIDs).
		Where("check_in_status = ?", model.ActivityCheckInDone)
	if start != nil {
		attendanceQuery = attendanceQuery.Where("signup_time >= ?", *start)
	}
	if end != nil {
		attendanceQuery = attendanceQuery.Where("signup_time <= ?", *end)
	}
	if err := attendanceQuery.Count(&metrics.AttendanceCount).Error; err != nil {
		return nil, err
	}

	workhourQuery := db.WithContext(r.ctx).
		Model(&model.ActivitySignup{}).
		Where("activity_id IN ?", activityIDs).
		Where("work_hour_status = ?", model.WorkHourStatusGranted)
	if start != nil {
		workhourQuery = workhourQuery.Where("signup_time >= ?", *start)
	}
	if end != nil {
		workhourQuery = workhourQuery.Where("signup_time <= ?", *end)
	}
	if err := workhourQuery.Count(&metrics.WorkhourCount).Error; err != nil {
		return nil, err
	}

	return metrics, nil
}

func (r *Repository) GetOpsDashboardMetrics(db *gorm.DB, orgID int64, start, end *time.Time) (*OpsDashboardMetrics, error) {
	metrics := &OpsDashboardMetrics{}

	activityIDs := make([]int64, 0)
	if err := db.WithContext(r.ctx).
		Model(&model.Activity{}).
		Where("org_id = ?", orgID).
		Pluck("id", &activityIDs).Error; err != nil {
		return nil, err
	}
	if len(activityIDs) == 0 {
		return metrics, nil
	}

	signupQuery := db.WithContext(r.ctx).
		Model(&model.ActivitySignup{}).
		Where("activity_id IN ?", activityIDs).
		Where("status <> ?", model.ActivitySignupStatusCanceled)
	if start != nil {
		signupQuery = signupQuery.Where("signup_time >= ?", *start)
	}
	if end != nil {
		signupQuery = signupQuery.Where("signup_time <= ?", *end)
	}
	if err := signupQuery.Count(&metrics.SignupCount).Error; err != nil {
		return nil, err
	}

	approvedQuery := db.WithContext(r.ctx).
		Model(&model.ActivitySignup{}).
		Where("activity_id IN ?", activityIDs).
		Where("status = ?", model.ActivitySignupStatusSuccess)
	if start != nil {
		approvedQuery = approvedQuery.Where("signup_time >= ?", *start)
	}
	if end != nil {
		approvedQuery = approvedQuery.Where("signup_time <= ?", *end)
	}
	if err := approvedQuery.Count(&metrics.ApprovedSignupCount).Error; err != nil {
		return nil, err
	}

	attendanceQuery := db.WithContext(r.ctx).
		Model(&model.ActivitySignup{}).
		Where("activity_id IN ?", activityIDs).
		Where("status = ? AND check_in_status = ?", model.ActivitySignupStatusSuccess, model.ActivityCheckInDone)
	if start != nil {
		attendanceQuery = attendanceQuery.Where("signup_time >= ?", *start)
	}
	if end != nil {
		attendanceQuery = attendanceQuery.Where("signup_time <= ?", *end)
	}
	if err := attendanceQuery.Count(&metrics.AttendanceCount).Error; err != nil {
		return nil, err
	}

	workHourQuery := db.WithContext(r.ctx).
		Model(&model.ActivitySignup{}).
		Where("activity_id IN ?", activityIDs).
		Where("work_hour_status = ?", model.WorkHourStatusGranted)
	if start != nil {
		workHourQuery = workHourQuery.Where("signup_time >= ?", *start)
	}
	if end != nil {
		workHourQuery = workHourQuery.Where("signup_time <= ?", *end)
	}
	if err := workHourQuery.Select("COALESCE(SUM(granted_hours), 0)").Scan(&metrics.GrantedWorkHours).Error; err != nil {
		return nil, err
	}

	return metrics, nil
}
