package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"
	"volunteer-system/internal/api"
	"volunteer-system/internal/middleware"
	"volunteer-system/internal/model"
	"volunteer-system/pkg/database/mysql"

	"github.com/cloudwego/hertz/pkg/app"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupServiceTestDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if len(models) > 0 {
		if err := db.AutoMigrate(models...); err != nil {
			t.Fatalf("auto migrate failed: %v", err)
		}
	}

	prev := mysql.DB
	mysql.DB = db
	t.Cleanup(func() {
		mysql.DB = prev
	})
	return db
}

func TestAuditRecordDetailRequiresAuditIdentity(t *testing.T) {
	db := setupServiceTestDB(t, &model.AuditRecord{})
	if err := db.Create(&model.AuditRecord{
		TargetType:    model.AuditTargetSignup,
		TargetID:      1,
		CreatorID:     100,
		AuditorID:     0,
		OldContent:    "{}",
		NewContent:    "{}",
		AuditResult:   0,
		RejectReason:  "",
		AuditTime:     time.Now(),
		CreatedAt:     time.Now(),
		OperationType: model.OperationTypeCreate,
		Status:        model.AuditStatusPending,
	}).Error; err != nil {
		t.Fatalf("seed audit record failed: %v", err)
	}

	svc := NewAuditService(context.Background(), &app.RequestContext{})
	_, err := svc.AuditRecordDetail(&api.AuditRecordDetailRequest{Id: 1})
	if err == nil {
		t.Fatalf("expected audit detail to require caller identity")
	}
}

func TestApplyAuditRejectionSideEffectsMaterializesRejectedSignup(t *testing.T) {
	setupServiceTestDB(t, &model.ActivitySignup{})

	svc := NewAuditService(context.Background(), &app.RequestContext{})
	snapshotBytes, err := json.Marshal(&model.ActivitySignup{
		ActivityID:  88,
		VolunteerID: 99,
		Status:      model.ActivitySignupStatusPending,
	})
	if err != nil {
		t.Fatalf("marshal snapshot failed: %v", err)
	}

	record := &model.AuditRecord{
		TargetType:    model.AuditTargetSignup,
		TargetID:      0,
		OperationType: model.OperationTypeCreate,
		NewContent:    string(snapshotBytes),
	}

	if err := svc.repo.DB.Transaction(func(tx *gorm.DB) error {
		return svc.applyAuditRejectionSideEffects(tx, record)
	}); err != nil {
		t.Fatalf("apply rejection side effects failed: %v", err)
	}

	if record.TargetID <= 0 {
		t.Fatalf("expected target id to be materialized after rejection")
	}

	signup, err := svc.repo.GetActivitySignupByID(svc.repo.DB, record.TargetID)
	if err != nil {
		t.Fatalf("query materialized signup failed: %v", err)
	}
	if signup.Status != model.ActivitySignupStatusRejected {
		t.Fatalf("expected signup status rejected, got %d", signup.Status)
	}
}

func TestIsRequestFieldProvided(t *testing.T) {
	queryCtx := &app.RequestContext{}
	queryCtx.Request.SetRequestURI("/api/activities/1?maxPeople=0")
	if !isRequestFieldProvided(queryCtx, "maxPeople") {
		t.Fatalf("expected query field maxPeople to be detected")
	}

	bodyCtx := &app.RequestContext{}
	bodyCtx.Request.SetBodyString(`{"maxPeople":0}`)
	if !isRequestFieldProvided(bodyCtx, "maxPeople") {
		t.Fatalf("expected json field maxPeople to be detected")
	}

	absentCtx := &app.RequestContext{}
	absentCtx.Request.SetBodyString(`{"title":"x"}`)
	if isRequestFieldProvided(absentCtx, "maxPeople") {
		t.Fatalf("expected missing field maxPeople to be false")
	}
}

func TestActivityDetailShowsRegisteredWhenPendingAuditExistsAfterRejectedSignup(t *testing.T) {
	db := setupServiceTestDB(t,
		&model.SysAccount{},
		&model.Volunteer{},
		&model.Activity{},
		&model.ActivitySignup{},
		&model.AuditRecord{},
	)

	now := time.Now()
	if err := db.Create(&model.SysAccount{
		ID:           1,
		Username:     "u1",
		Mobile:       "m1",
		MobileHash:   "h1",
		Email:        "u1@test.local",
		Password:     "p1",
		IdentityType: model.RegisterTypeVolunteerCode,
		Status:       model.SysAccountNormal,
		CreatedAt:    now,
		UpdatedAt:    now,
	}).Error; err != nil {
		t.Fatalf("seed account failed: %v", err)
	}
	if err := db.Create(&model.Volunteer{
		ID:           10,
		AccountID:    1,
		RealName:     "v1",
		Gender:       1,
		IDCard:       "id1",
		AvatarURL:    "a1",
		Introduction: "intro",
		LevelID:      1,
		Status:       model.VolunteerActiveStatus,
		AuditStatus:  model.VolunteerAuditStatusApproved,
		CreatedAt:    now,
		UpdatedAt:    now,
	}).Error; err != nil {
		t.Fatalf("seed volunteer failed: %v", err)
	}
	if err := db.Create(&model.Activity{
		ID:            100,
		OrgID:         200,
		Title:         "act",
		Description:   "desc",
		CoverURL:      "cover",
		StartTime:     now.Add(2 * time.Hour),
		EndTime:       now.Add(3 * time.Hour),
		Location:      "loc",
		Address:       "addr",
		Duration:      1,
		MaxPeople:     10,
		CurrentPeople: 0,
		Status:        model.ActivityStatusRecruiting,
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error; err != nil {
		t.Fatalf("seed activity failed: %v", err)
	}
	if err := db.Create(&model.ActivitySignup{
		ID:          1000,
		ActivityID:  100,
		VolunteerID: 10,
		SignupTime:  now,
		Status:      model.ActivitySignupStatusRejected,
		CreatedAt:   now,
		UpdatedAt:   now,
	}).Error; err != nil {
		t.Fatalf("seed rejected signup failed: %v", err)
	}

	pendingSnapshot, err := json.Marshal(&model.ActivitySignup{
		ActivityID:  100,
		VolunteerID: 10,
		Status:      model.ActivitySignupStatusPending,
	})
	if err != nil {
		t.Fatalf("marshal pending snapshot failed: %v", err)
	}
	if err := db.Create(&model.AuditRecord{
		ID:            5001,
		TargetType:    model.AuditTargetSignup,
		TargetID:      0,
		CreatorID:     1,
		AuditorID:     0,
		OldContent:    "{}",
		NewContent:    string(pendingSnapshot),
		AuditResult:   0,
		RejectReason:  "",
		AuditTime:     now,
		CreatedAt:     now,
		OperationType: model.OperationTypeCreate,
		Status:        model.AuditStatusPending,
	}).Error; err != nil {
		t.Fatalf("seed pending signup audit failed: %v", err)
	}

	c := &app.RequestContext{}
	c.Set(middleware.UserIDKey, "1")
	svc := NewActivityService(context.Background(), c)
	resp, err := svc.ActivityDetail(&api.ActivityDetailRequest{Id: 100})
	if err != nil {
		t.Fatalf("activity detail failed: %v", err)
	}
	if resp == nil || resp.Activity == nil {
		t.Fatalf("expected activity detail response")
	}
	if !resp.Activity.IsRegistered {
		t.Fatalf("expected isRegistered=true when pending audit exists after rejected signup")
	}
}

func TestApplySignupAuditApprovalNonCreateBranchValidatesActivityStatus(t *testing.T) {
	db := setupServiceTestDB(t, &model.Activity{}, &model.ActivitySignup{})
	now := time.Now()
	if err := db.Create(&model.Activity{
		ID:            200,
		OrgID:         1,
		Title:         "a",
		Description:   "b",
		CoverURL:      "c",
		StartTime:     now,
		EndTime:       now.Add(time.Hour),
		Location:      "l",
		Address:       "addr",
		Duration:      1,
		MaxPeople:     10,
		CurrentPeople: 0,
		Status:        model.ActivityStatusCanceled,
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error; err != nil {
		t.Fatalf("seed activity failed: %v", err)
	}
	if err := db.Create(&model.ActivitySignup{
		ID:          2001,
		ActivityID:  200,
		VolunteerID: 2,
		SignupTime:  now,
		Status:      model.ActivitySignupStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}).Error; err != nil {
		t.Fatalf("seed signup failed: %v", err)
	}

	svc := NewAuditService(context.Background(), &app.RequestContext{})
	err := svc.repo.DB.Transaction(func(tx *gorm.DB) error {
		return svc.applySignupAuditApproval(tx, &model.AuditRecord{
			TargetType:    model.AuditTargetSignup,
			TargetID:      2001,
			OperationType: model.OperationTypeUpdate,
		})
	})
	if err == nil {
		t.Fatalf("expected approval to fail for canceled activity")
	}
	if err.Error() != "活动已结束或已取消" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoginReturnsBusinessFailureWhenUserNotFound(t *testing.T) {
	setupServiceTestDB(t, &model.SysAccount{})

	svc := NewLoginService(context.Background(), &app.RequestContext{})
	resp, err := svc.Login(&api.LoginRequest{
		LoginType:  "email",
		Identifier: "not-exists@test.local",
		Password:   "pass123",
		Identity:   model.RegisterTypeVolunteer,
	})
	if err != nil {
		t.Fatalf("expected no internal error when user not found, got: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected response")
	}
	if resp.Success {
		t.Fatalf("expected success=false for not-found user")
	}
	if resp.Message != "用户不存在" {
		t.Fatalf("unexpected message: %s", resp.Message)
	}
}

func TestVolunteerJoinOrganizationAllowsReapplyFromLeftMembership(t *testing.T) {
	db := setupServiceTestDB(t,
		&model.SysAccount{},
		&model.Volunteer{},
		&model.Organization{},
		&model.OrgMember{},
		&model.AuditRecord{},
	)

	now := time.Now()
	if err := db.Create(&model.SysAccount{
		ID:           9001,
		Username:     "u1",
		Mobile:       "m1",
		MobileHash:   "mh1",
		Email:        "u1@test.local",
		Password:     "pwd",
		IdentityType: model.RegisterTypeVolunteerCode,
		Status:       model.SysAccountNormal,
		CreatedAt:    now,
		UpdatedAt:    now,
	}).Error; err != nil {
		t.Fatalf("seed account failed: %v", err)
	}
	if err := db.Create(&model.Volunteer{
		ID:           9010,
		AccountID:    9001,
		RealName:     "v1",
		Gender:       1,
		IDCard:       "id1",
		AvatarURL:    "avatar",
		Introduction: "intro",
		LevelID:      1,
		Status:       model.VolunteerActiveStatus,
		AuditStatus:  model.VolunteerAuditStatusApproved,
		CreatedAt:    now,
		UpdatedAt:    now,
	}).Error; err != nil {
		t.Fatalf("seed volunteer failed: %v", err)
	}
	if err := db.Create(&model.Organization{
		ID:            9100,
		AccountID:     9002,
		OrgName:       "org1",
		LicenseCode:   "license1",
		ContactPerson: "owner",
		ContactPhone:  "123456",
		Address:       "addr",
		LogoURL:       "logo",
		Introduction:  "intro",
		Status:        1,
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error; err != nil {
		t.Fatalf("seed organization failed: %v", err)
	}
	joinedAt := now.Add(-48 * time.Hour)
	if err := db.Create(&model.OrgMember{
		ID:          9501,
		OrgID:       9100,
		VolunteerID: 9010,
		Role:        model.MemberRoleMember,
		Status:      model.MemberStatusLeft,
		AppliedAt:   now.Add(-72 * time.Hour),
		JoinedAt:    &joinedAt,
		CreatedAt:   now.Add(-72 * time.Hour),
		UpdatedAt:   now.Add(-24 * time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed membership failed: %v", err)
	}

	c := &app.RequestContext{}
	c.Set(middleware.UserIDKey, "9001")
	svc := NewMembershipService(context.Background(), c)

	resp, err := svc.VolunteerJoinOrganization(&api.VolunteerJoinRequest{
		OrganizationId: 9100,
		VolunteerId:    9010,
	})
	if err != nil {
		t.Fatalf("expected reapply to succeed, got error: %v", err)
	}
	if resp == nil || resp.Status != model.MemberStatusPending {
		t.Fatalf("expected pending response, got: %+v", resp)
	}

	records, _, err := svc.repo.GetAuditRecordsList(svc.repo.DB, map[string]any{
		"target_type = ?": model.AuditTargetMember,
		"target_id = ?":   int64(9501),
		"status = ?":      model.AuditStatusPending,
	}, 10, 0)
	if err != nil {
		t.Fatalf("query audit records failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected one pending member audit record, got %d", len(records))
	}
	if records[0].OperationType != model.OperationTypeUpdate {
		t.Fatalf("expected update audit operation, got %d", records[0].OperationType)
	}

	var newSnapshot model.OrgMember
	if err := json.Unmarshal([]byte(records[0].NewContent), &newSnapshot); err != nil {
		t.Fatalf("unmarshal new snapshot failed: %v", err)
	}
	if newSnapshot.Status != model.MemberStatusActive {
		t.Fatalf("expected new snapshot status active, got %d", newSnapshot.Status)
	}
	if newSnapshot.JoinedAt != nil {
		t.Fatalf("expected joined_at to be empty in pending snapshot")
	}
}

func TestVolunteerUpdateRejectsCrossAccountWithoutAccess(t *testing.T) {
	db := setupServiceTestDB(t, &model.SysAccount{}, &model.Volunteer{})
	now := time.Now()

	seedAccount := func(id int64, email string) {
		if err := db.Create(&model.SysAccount{
			ID:           id,
			Username:     email,
			Mobile:       "m" + email,
			MobileHash:   "h" + email,
			Email:        email,
			Password:     "pwd",
			IdentityType: model.RegisterTypeVolunteerCode,
			Status:       model.SysAccountNormal,
			CreatedAt:    now,
			UpdatedAt:    now,
		}).Error; err != nil {
			t.Fatalf("seed account failed: %v", err)
		}
	}
	seedVolunteer := func(id, accountID int64, name string) {
		if err := db.Create(&model.Volunteer{
			ID:           id,
			AccountID:    accountID,
			RealName:     name,
			Gender:       1,
			IDCard:       "id-" + name,
			AvatarURL:    "avatar",
			Introduction: "intro",
			LevelID:      1,
			Status:       model.VolunteerActiveStatus,
			AuditStatus:  model.VolunteerAuditStatusApproved,
			CreatedAt:    now,
			UpdatedAt:    now,
		}).Error; err != nil {
			t.Fatalf("seed volunteer failed: %v", err)
		}
	}

	seedAccount(9101, "u9101@test.local")
	seedAccount(9102, "u9102@test.local")
	seedVolunteer(9201, 9101, "v1")
	seedVolunteer(9202, 9102, "v2")

	c := &app.RequestContext{}
	c.Set(middleware.UserIDKey, "9101")
	svc := NewVolunteerService(context.Background(), c)
	_, err := svc.VolunteerUpdate(&api.VolunteerUpdateRequest{
		VolunteerId: 9202,
		RealName:    "new-name",
	})
	if err == nil {
		t.Fatalf("expected cross-account update to be denied")
	}
	if err.Error() != "无权更新该志愿者信息" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVolunteerUpdateAllowsSelfUpdate(t *testing.T) {
	db := setupServiceTestDB(t, &model.SysAccount{}, &model.Volunteer{})
	now := time.Now()

	if err := db.Create(&model.SysAccount{
		ID:           9301,
		Username:     "u9301",
		Mobile:       "m9301",
		MobileHash:   "h9301",
		Email:        "u9301@test.local",
		Password:     "pwd",
		IdentityType: model.RegisterTypeVolunteerCode,
		Status:       model.SysAccountNormal,
		CreatedAt:    now,
		UpdatedAt:    now,
	}).Error; err != nil {
		t.Fatalf("seed account failed: %v", err)
	}
	if err := db.Create(&model.Volunteer{
		ID:           9401,
		AccountID:    9301,
		RealName:     "before",
		Gender:       1,
		IDCard:       "id9401",
		AvatarURL:    "avatar",
		Introduction: "intro",
		LevelID:      1,
		Status:       model.VolunteerActiveStatus,
		AuditStatus:  model.VolunteerAuditStatusApproved,
		CreatedAt:    now,
		UpdatedAt:    now,
	}).Error; err != nil {
		t.Fatalf("seed volunteer failed: %v", err)
	}

	c := &app.RequestContext{}
	c.Set(middleware.UserIDKey, "9301")
	svc := NewVolunteerService(context.Background(), c)
	if _, err := svc.VolunteerUpdate(&api.VolunteerUpdateRequest{
		VolunteerId: 9401,
		RealName:    "after",
	}); err != nil {
		t.Fatalf("expected self update success, got: %v", err)
	}

	volunteer, err := svc.repo.FindVolunteerByID(svc.repo.DB, 9401)
	if err != nil {
		t.Fatalf("query volunteer failed: %v", err)
	}
	if volunteer.RealName != "after" {
		t.Fatalf("expected name to be updated, got %s", volunteer.RealName)
	}
}

func TestValidateVolunteerRequestAllowsOptionalAgeAndGender(t *testing.T) {
	svc := &RegisterService{}
	err := svc.validateVolunteerRequest(&api.VolunteerRegisterRequest{
		Name:     "tester",
		Phone:    "13800138000",
		Email:    "tester@example.com",
		Password: "pass123",
		UserName: "tester01",
		Age:      0,
		Gender:   "",
	})
	if err != nil {
		t.Fatalf("expected optional age/gender to be accepted, got: %v", err)
	}
}

func TestValidateVolunteerRequestRejectsTooLargeAge(t *testing.T) {
	svc := &RegisterService{}
	err := svc.validateVolunteerRequest(&api.VolunteerRegisterRequest{
		Name:     "tester",
		Phone:    "13800138000",
		Email:    "tester@example.com",
		Password: "pass123",
		UserName: "tester01",
		Age:      121,
	})
	if err == nil {
		t.Fatalf("expected age over limit to be rejected")
	}
	if err.Error() != "年龄必须在1-120岁之间" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVoidWorkHourRejectsIdempotencyKeyReuseWithDifferentReason(t *testing.T) {
	db := setupServiceTestDB(t, &model.WorkHourLog{})
	now := time.Now()
	if err := db.Create(&model.WorkHourLog{
		ID:                 8001,
		VolunteerID:        1,
		ActivityID:         2,
		SignupID:           3001,
		OperationType:      model.WorkHourOperationVoid,
		HoursDelta:         -1,
		ServiceCountDelta:  -1,
		BeforeTotalHours:   10,
		AfterTotalHours:    9,
		BeforeServiceCount: 5,
		AfterServiceCount:  4,
		WorkHourVersion:    2,
		IdempotencyKey:     "void-dup-key",
		RefLogID:           7001,
		Reason:             "old reason",
		OperatorID:         5001,
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error; err != nil {
		t.Fatalf("seed work hour log failed: %v", err)
	}

	c := &app.RequestContext{}
	c.Set(middleware.UserIDKey, "5001")
	svc := NewWorkHourService(context.Background(), c)
	_, err := svc.VoidWorkHour(&api.VoidWorkHourRequest{
		SignupId:       3001,
		Reason:         "new reason",
		IdempotencyKey: "void-dup-key",
	})
	if err == nil {
		t.Fatalf("expected idempotency conflict")
	}
	if err.Error() != "幂等键已被其他请求占用" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecalculateWorkHourRejectsIdempotencyKeyReuseWithDifferentReason(t *testing.T) {
	db := setupServiceTestDB(t, &model.WorkHourLog{})
	now := time.Now()
	if err := db.Create(&model.WorkHourLog{
		ID:                 8002,
		VolunteerID:        1,
		ActivityID:         2,
		SignupID:           3002,
		OperationType:      model.WorkHourOperationRegrant,
		HoursDelta:         1,
		ServiceCountDelta:  0,
		BeforeTotalHours:   10,
		AfterTotalHours:    11,
		BeforeServiceCount: 5,
		AfterServiceCount:  5,
		WorkHourVersion:    3,
		IdempotencyKey:     "regrant-dup-key",
		RefLogID:           7002,
		Reason:             "old reason",
		OperatorID:         5002,
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error; err != nil {
		t.Fatalf("seed work hour log failed: %v", err)
	}

	c := &app.RequestContext{}
	c.Set(middleware.UserIDKey, "5002")
	svc := NewWorkHourService(context.Background(), c)
	_, err := svc.RecalculateWorkHour(&api.RecalculateWorkHourRequest{
		SignupId:       3002,
		Hours:          2,
		Reason:         "new reason",
		IdempotencyKey: "regrant-dup-key",
	})
	if err == nil {
		t.Fatalf("expected idempotency conflict")
	}
	if err.Error() != "幂等键已被其他请求占用" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivityListRequiresUserContextBeforeKeywordEmptyResult(t *testing.T) {
	setupServiceTestDB(t, &model.Activity{})

	svc := NewActivityService(context.Background(), &app.RequestContext{})
	_, err := svc.ActivityList(&api.ActivityListRequest{
		Keyword: "no-such-activity",
	})
	if err == nil {
		t.Fatalf("expected activity list to require user context even when keyword has no matches")
	}
}

func TestActivityListReturnsErrorWhenAccountMissingEvenIfListEmpty(t *testing.T) {
	setupServiceTestDB(t, &model.Activity{}, &model.SysAccount{})

	c := &app.RequestContext{}
	c.Set(middleware.UserIDKey, "99999")

	svc := NewActivityService(context.Background(), c)
	_, err := svc.ActivityList(&api.ActivityListRequest{
		Page:     1,
		PageSize: 10,
	})
	if err == nil {
		t.Fatalf("expected error when current account does not exist")
	}
	if err.Error() != "账号不存在" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestActivityDetailReturnsAccountMissingBeforeActivityCheck(t *testing.T) {
	setupServiceTestDB(t, &model.SysAccount{}, &model.Activity{})

	c := &app.RequestContext{}
	c.Set(middleware.UserIDKey, "99999")

	svc := NewActivityService(context.Background(), c)
	_, err := svc.ActivityDetail(&api.ActivityDetailRequest{Id: 123456})
	if err == nil {
		t.Fatalf("expected error when current account does not exist")
	}
	if err.Error() != "账号不存在" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuditRecordDetailChecksOperatorBeforeRecordLookup(t *testing.T) {
	setupServiceTestDB(t, &model.AuditRecord{})

	svc := NewAuditService(context.Background(), &app.RequestContext{})
	_, err := svc.AuditRecordDetail(&api.AuditRecordDetailRequest{Id: 999999})
	if err == nil {
		t.Fatalf("expected error when caller identity is missing")
	}
	if err.Error() == "审核记录不存在" {
		t.Fatalf("expected identity error before record-not-found, got: %v", err)
	}
}

func TestGetAuditOperatorIDReturnsAccountNotFound(t *testing.T) {
	setupServiceTestDB(t, &model.SysAccount{})

	c := &app.RequestContext{}
	c.Set(middleware.UserIDKey, "12345")

	svc := NewAuditService(context.Background(), c)
	_, err := svc.getAuditOperatorID()
	if err == nil {
		t.Fatalf("expected account-not-found error")
	}
	if err.Error() != "账号不存在" {
		t.Fatalf("unexpected error: %v", err)
	}
}
