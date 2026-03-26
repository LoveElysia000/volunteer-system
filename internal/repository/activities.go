package repository

import (
	"strings"
	"volunteer-system/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetActivitiesByStatus 根据状态查询活动列表
func (r *Repository) GetActivitiesByStatus(db *gorm.DB, status int32, limit, offset int) ([]*model.Activity, int64, error) {
	var activities []*model.Activity
	var total int64

	// 创建 base session，关联 organizations 表获取组织信息
	baseSession := db.WithContext(r.ctx).
		Table("activities as act")

	// 状态筛选
	if status > 0 {
		baseSession = baseSession.Where("act.status = ?", status)
	}

	// 使用 base session 获取总数
	err := baseSession.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		// 如果没有数据，直接返回空结果
		return activities, 0, nil
	}

	// 使用 base session 分页查询数据
	querySession := baseSession.Select("act.*")
	err = querySession.Offset(offset).Limit(limit).
		Order("act.created_at DESC").
		Find(&activities).Error
	if err != nil {
		return nil, 0, err
	}

	return activities, total, nil
}

// GetActivitiesByFilters queries activity list with keyword/time-range/sort filters.
func (r *Repository) GetActivitiesByFilters(db *gorm.DB, filters map[string]any, limit, offset int) ([]*model.Activity, int64, error) {
	var activities []*model.Activity
	var total int64

	baseSession := db.WithContext(r.ctx).
		Table("activities as act")

	for query, value := range filters {
		baseSession = baseSession.Where(query, value)
	}

	if err := baseSession.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return activities, 0, nil
	}

	err := baseSession.Select("act.*").
		Offset(offset).
		Limit(limit).
		Order("act.created_at DESC").
		Find(&activities).Error
	if err != nil {
		return nil, 0, err
	}

	return activities, total, nil
}

// ListActivityIDsByKeyword returns activity ids matched by keyword in title/description/location.
func (r *Repository) ListActivityIDsByKeyword(db *gorm.DB, keyword string) ([]int64, error) {
	trimmedKeyword := strings.TrimSpace(keyword)
	if trimmedKeyword == "" {
		return []int64{}, nil
	}

	like := "%" + trimmedKeyword + "%"
	ids := make([]int64, 0)
	err := db.WithContext(r.ctx).
		Table("activities as act").
		Where("(act.title LIKE ? OR act.description LIKE ? OR act.location LIKE ?)", like, like, like).
		Pluck("act.id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// GetActivityByID 根据ID查询活动
func (r *Repository) GetActivityByID(db *gorm.DB, id int64) (*model.Activity, error) {
	var activity model.Activity
	err := db.WithContext(r.ctx).Where("id = ?", id).First(&activity).Error
	if err != nil {
		return nil, err
	}
	return &activity, nil
}

// GetActivityByIDForUpdate 根据ID查询活动并加行锁
func (r *Repository) GetActivityByIDForUpdate(db *gorm.DB, id int64) (*model.Activity, error) {
	var activity model.Activity
	err := db.WithContext(r.ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&activity).Error
	if err != nil {
		return nil, err
	}
	return &activity, nil
}

// UpdateActivityAttendanceCodeByID 更新活动签到签退码相关字段。
func (r *Repository) UpdateActivityAttendanceCodeByID(db *gorm.DB, id int64, updates map[string]any) error {
	return db.WithContext(r.ctx).
		Model(&model.Activity{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// GetSignup 查询报名记录
func (r *Repository) GetSignup(db *gorm.DB, activityID, volunteerID int64) (*model.ActivitySignup, error) {
	var signup model.ActivitySignup
	err := db.WithContext(r.ctx).
		Where("activity_id = ? AND volunteer_id = ?", activityID, volunteerID).
		First(&signup).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &signup, nil
}

// GetSignupForUpdate 查询报名记录并加行锁
func (r *Repository) GetSignupForUpdate(db *gorm.DB, activityID, volunteerID int64) (*model.ActivitySignup, error) {
	var signup model.ActivitySignup
	err := db.WithContext(r.ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("activity_id = ? AND volunteer_id = ?", activityID, volunteerID).
		First(&signup).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &signup, nil
}

// GetActivitySignupByID finds a signup record by id.
func (r *Repository) GetActivitySignupByID(db *gorm.DB, id int64) (*model.ActivitySignup, error) {
	var signup model.ActivitySignup
	err := db.WithContext(r.ctx).
		Model(&model.ActivitySignup{}).
		Where("id = ?", id).
		First(&signup).Error
	if err != nil {
		return nil, err
	}
	return &signup, nil
}

// GetActivitySignupByIDForUpdate 根据ID查询报名记录并加行锁
func (r *Repository) GetActivitySignupByIDForUpdate(db *gorm.DB, id int64) (*model.ActivitySignup, error) {
	var signup model.ActivitySignup
	err := db.WithContext(r.ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&model.ActivitySignup{}).
		Where("id = ?", id).
		First(&signup).Error
	if err != nil {
		return nil, err
	}
	return &signup, nil
}

// CreateSignup 创建报名记录
func (r *Repository) CreateSignup(db *gorm.DB, signup *model.ActivitySignup) error {
	return db.WithContext(r.ctx).Create(signup).Error
}

// UpdateActivitySignupStatusByID updates signup status by id.
func (r *Repository) UpdateActivitySignupStatusByID(db *gorm.DB, id int64, status int32) error {
	return db.WithContext(r.ctx).
		Model(&model.ActivitySignup{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// UpdateSignupStatus 更新报名记录状态
func (r *Repository) UpdateSignupStatus(db *gorm.DB, signup *model.ActivitySignup) error {
	return db.WithContext(r.ctx).Save(signup).Error
}

// UpdateActivitySignupByID updates signup fields by id.
func (r *Repository) UpdateActivitySignupByID(db *gorm.DB, id int64, updates map[string]any) error {
	return db.WithContext(r.ctx).
		Model(&model.ActivitySignup{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// IncrementActivityPeople 增加活动当前报名人数（原子操作）
func (r *Repository) IncrementActivityPeople(db *gorm.DB, activityID int64) error {
	result := db.WithContext(r.ctx).Model(&model.Activity{}).
		Where("id = ? AND (current_people < max_people OR max_people = 0)", activityID).
		Update("current_people", gorm.Expr("current_people + 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// DecrementActivityPeople 减少活动当前报名人数
func (r *Repository) DecrementActivityPeople(db *gorm.DB, activityID int64) error {
	result := db.WithContext(r.ctx).Model(&model.Activity{}).
		Where("id = ? AND current_people > 0", activityID).
		Update("current_people", gorm.Expr("current_people - 1"))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ListUserSignupsByActivityIDs 查询用户在指定活动中的报名记录列表。
func (r *Repository) ListUserSignupsByActivityIDs(db *gorm.DB, volunteerID int64, activityIDs []int64) ([]*model.ActivitySignup, error) {
	if len(activityIDs) == 0 {
		return []*model.ActivitySignup{}, nil
	}

	var signups []*model.ActivitySignup
	activeSignupStatuses := []int32{
		model.ActivitySignupStatusPending,
		model.ActivitySignupStatusSuccess,
	}
	err := db.WithContext(r.ctx).
		Where("volunteer_id = ? AND activity_id IN ? AND status IN ?", volunteerID, activityIDs, activeSignupStatuses).
		Find(&signups).Error
	if err != nil {
		return nil, err
	}
	return signups, nil
}

// ListVisibleSignupsByVolunteer returns signups that should be visible in unified activity list filtering.
func (r *Repository) ListVisibleSignupsByVolunteer(db *gorm.DB, volunteerID int64) ([]*model.ActivitySignup, error) {
	if volunteerID <= 0 {
		return []*model.ActivitySignup{}, nil
	}

	var signups []*model.ActivitySignup
	visibleSignupStatuses := []int32{
		model.ActivitySignupStatusPending,
		model.ActivitySignupStatusSuccess,
	}
	err := db.WithContext(r.ctx).
		Where("volunteer_id = ? AND status IN ?", volunteerID, visibleSignupStatuses).
		Find(&signups).Error
	if err != nil {
		return nil, err
	}
	return signups, nil
}

// CountActivitySignups 统计活动报名人数
func (r *Repository) CountActivitySignups(db *gorm.DB, activityID int64) (int64, error) {
	var count int64
	err := db.WithContext(r.ctx).
		Model(&model.ActivitySignup{}).
		Where("activity_id = ? AND status IN ?", activityID, []int32{
			model.ActivitySignupStatusPending,
			model.ActivitySignupStatusSuccess,
		}).
		Count(&count).Error
	return count, err
}

// GetActivityWithOrg 查询活动及组织信息
func (r *Repository) GetActivityWithOrg(db *gorm.DB, id int64) (*model.Activity, string, error) {
	var activity model.Activity
	var orgName string

	// 查询活动信息
	err := db.WithContext(r.ctx).Where("id = ?", id).First(&activity).Error
	if err != nil {
		return nil, "", err
	}

	// 查询组织名称
	if activity.OrgID > 0 {
		err = db.WithContext(r.ctx).Model(&model.Organization{}).
			Where("id = ?", activity.OrgID).
			Pluck("org_name", &orgName).Error
		if err != nil {
			// 组织不存在或查询失败不影响主流程
			orgName = ""
		}
	}

	return &activity, orgName, nil
}

// GetMyActivities 获取用户的活动列表（从报名表获取）
func (r *Repository) GetMyActivities(db *gorm.DB, volunteerID int64, status int32, limit, offset int) ([]*model.ActivitySignup, int64, error) {
	var signups []*model.ActivitySignup
	var total int64

	visibleSignupStatuses := []int32{
		model.ActivitySignupStatusPending,
		model.ActivitySignupStatusSuccess,
		model.ActivitySignupStatusRejected,
	}
	query := db.WithContext(r.ctx).
		Where("volunteer_id = ? AND status IN ?", volunteerID, visibleSignupStatuses)

	// 状态筛选（通过关联的活动状态）
	if status > 0 {
		activityIDs := make([]int64, 0)
		if err := db.WithContext(r.ctx).
			Model(&model.Activity{}).
			Where("status = ?", status).
			Pluck("id", &activityIDs).Error; err != nil {
			return nil, 0, err
		}
		if len(activityIDs) == 0 {
			return signups, 0, nil
		}
		query = query.Where("activity_id IN ?", activityIDs)
	}

	// 查询总数
	if err := query.Model(&model.ActivitySignup{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	if err := query.Offset(offset).Limit(limit).
		Order("signup_time DESC").
		Find(&signups).Error; err != nil {
		return nil, 0, err
	}

	return signups, total, nil
}

// ListActivitiesByIDs 批量获取活动信息列表。
func (r *Repository) ListActivitiesByIDs(db *gorm.DB, activityIDs []int64) ([]*model.Activity, error) {
	if len(activityIDs) == 0 {
		return []*model.Activity{}, nil
	}

	var activities []*model.Activity
	err := db.WithContext(r.ctx).
		Where("id IN ?", activityIDs).
		Find(&activities).Error
	if err != nil {
		return nil, err
	}
	return activities, nil
}

// ListOrganizationsByIDs 批量获取组织列表（仅查询 id/org_name 字段）。
func (r *Repository) ListOrganizationsByIDs(db *gorm.DB, orgIDs []int64) ([]*model.Organization, error) {
	if len(orgIDs) == 0 {
		return []*model.Organization{}, nil
	}

	var organizations []*model.Organization
	err := db.WithContext(r.ctx).Model(&model.Organization{}).
		Select("id", "org_name").
		Where("id IN ?", orgIDs).
		Find(&organizations).Error
	if err != nil {
		return nil, err
	}
	return organizations, nil
}

// ========== 组织端活动管理 ==========

// CreateActivity 创建活动
func (r *Repository) CreateActivity(db *gorm.DB, activity *model.Activity) error {
	return db.WithContext(r.ctx).Create(activity).Error
}

// UpdateActivity 更新活动
func (r *Repository) UpdateActivity(db *gorm.DB, activity *model.Activity) error {
	return db.WithContext(r.ctx).Save(activity).Error
}

// DeleteActivity 删除活动（软删除）
func (r *Repository) DeleteActivity(db *gorm.DB, id int64) error {
	return db.WithContext(r.ctx).Delete(&model.Activity{}, id).Error
}

// CancelActivity 取消活动
func (r *Repository) CancelActivity(db *gorm.DB, id int64) error {
	return db.WithContext(r.ctx).Model(&model.Activity{}).
		Where("id = ?", id).
		Update("status", model.ActivityStatusCanceled).Error
}

// FinishActivity 完结活动
func (r *Repository) FinishActivity(db *gorm.DB, id int64) error {
	return db.WithContext(r.ctx).
		Model(&model.Activity{}).
		Where("id = ?", id).
		Update("status", model.ActivityStatusFinished).Error
}

// GetOrganizationByAccountID 根据账号ID获取组织
func (r *Repository) GetOrganizationByAccountID(db *gorm.DB, accountID int64) (*model.Organization, error) {
	var org model.Organization
	err := db.WithContext(r.ctx).Where("account_id = ?", accountID).First(&org).Error
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// GetOrganizationActivities 获取组织的活动列表
func (r *Repository) GetOrganizationActivities(db *gorm.DB, orgID int64, limit, offset int) ([]*model.Activity, int64, error) {
	var activities []*model.Activity
	var total int64

	// 创建 base session
	baseSession := db.WithContext(r.ctx).
		Table("activities as act").
		Where("act.org_id = ?", orgID)

	// 使用 base session 获取总数
	err := baseSession.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if total == 0 {
		// 如果没有数据，直接返回空结果
		return activities, 0, nil
	}

	// 使用 base session 分页查询数据
	querySession := baseSession.Select("act.*")
	err = querySession.Offset(offset).Limit(limit).
		Order("act.created_at DESC").
		Find(&activities).Error
	if err != nil {
		return nil, 0, err
	}

	return activities, total, nil
}
