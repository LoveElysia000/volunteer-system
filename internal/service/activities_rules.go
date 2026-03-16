package service

import (
	"errors"
	"time"
	"volunteer-system/internal/model"
)

// canSignupActivity validates whether volunteer signup is allowed for current activity state.
func canSignupActivity(activity *model.Activity, existingSignup *model.ActivitySignup, hasPendingAudit bool) error {
	if activity == nil {
		return errors.New("活动不存在")
	}
	if activity.Status != model.ActivityStatusRecruiting {
		return errors.New("活动已结束或已取消")
	}
	if activity.MaxPeople > 0 && activity.CurrentPeople >= activity.MaxPeople {
		return errors.New("名额已满")
	}
	if existingSignup != nil &&
		(existingSignup.Status == model.ActivitySignupStatusPending || existingSignup.Status == model.ActivitySignupStatusSuccess) {
		return errors.New("请勿重复报名")
	}
	if hasPendingAudit {
		return errors.New("请勿重复报名")
	}
	return nil
}

// canUpdateActivity validates whether activity fields can still be edited.
func canUpdateActivity(activity *model.Activity) error {
	if activity == nil {
		return errors.New("活动不存在")
	}
	if activity.Status == model.ActivityStatusFinished || activity.Status == model.ActivityStatusCanceled {
		return errors.New("已结束或已取消的活动不能修改")
	}
	return nil
}

// canCancelActivity validates whether activity can be canceled by organization.
func canCancelActivity(activity *model.Activity) error {
	if activity == nil {
		return errors.New("活动不存在")
	}
	if activity.Status == model.ActivityStatusFinished || activity.Status == model.ActivityStatusCanceled {
		return errors.New("已结束或已取消的活动不能取消")
	}
	return nil
}

// canFinishActivity validates whether activity can be manually marked finished.
func canFinishActivity(activity *model.Activity) error {
	if activity == nil {
		return errors.New("活动不存在")
	}
	if activity.Status == model.ActivityStatusFinished {
		return errors.New("活动已结束")
	}
	if activity.Status == model.ActivityStatusCanceled {
		return errors.New("已取消活动不能完结")
	}
	return nil
}

// canCheckIn validates whether volunteer check-in can proceed.
func canCheckIn(activity *model.Activity, signup *model.ActivitySignup) error {
	if activity == nil {
		return errors.New("活动不存在")
	}
	if activity.Status == model.ActivityStatusCanceled {
		return errors.New("已取消活动不允许签到")
	}
	if signup == nil {
		return errors.New("报名记录不存在")
	}
	if signup.Status != model.ActivitySignupStatusSuccess {
		return errors.New("当前报名状态不允许签到")
	}
	if signup.CheckOutStatus == model.ActivityCheckOutDone {
		return errors.New("已签退，无法再次签到")
	}
	return nil
}

// canCheckOut validates whether volunteer check-out can proceed at current time.
func canCheckOut(activity *model.Activity, signup *model.ActivitySignup, now time.Time, earliestWindow time.Duration) error {
	if activity == nil {
		return errors.New("活动不存在")
	}
	if activity.Status == model.ActivityStatusCanceled {
		return errors.New("已取消活动不允许签退")
	}
	if signup == nil {
		return errors.New("报名记录不存在")
	}
	if signup.Status != model.ActivitySignupStatusSuccess {
		return errors.New("当前报名状态不允许签退")
	}
	if signup.CheckInStatus != model.ActivityCheckInDone || signup.CheckInTime == nil {
		return errors.New("未签到，无法签退")
	}
	if now.Before(activity.EndTime.Add(-earliestWindow)) {
		return errors.New("未到签退开始时间，还不能签退")
	}
	return nil
}

// canSupplementAttendance validates supplement attendance state for activity and signup.
// Pass nil signup to check activity-level eligibility only.
func canSupplementAttendance(activity *model.Activity, signup *model.ActivitySignup) error {
	if activity == nil {
		return errors.New("活动不存在")
	}
	if activity.Status == model.ActivityStatusCanceled {
		return errors.New("已取消活动不允许补录")
	}
	if signup == nil {
		return nil
	}
	if signup.Status != model.ActivitySignupStatusSuccess {
		return errors.New("当前报名状态不允许补录")
	}
	return nil
}
