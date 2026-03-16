package service

import (
	"errors"
	"volunteer-system/internal/model"
)

type signupTransitionType string

const (
	signupTransitionApprove signupTransitionType = "approve"
	signupTransitionReject  signupTransitionType = "reject"
	signupTransitionCancel  signupTransitionType = "cancel"
)

// 活动报名状态机说明：
// 状态: pending(待审核), success(报名成功), rejected(驳回), canceled(已取消)
//
//  1. approve:
//     nil -> success（createIfMissing=true, peopleDelta=+1）
//     pending/rejected/canceled -> success（peopleDelta=+1）
//     success -> no-op
//
//  2. reject:
//     nil -> rejected（createIfMissing=true, peopleDelta=0）
//     pending -> rejected（peopleDelta=0）
//     success/rejected/canceled -> no-op
//
//  3. cancel:
//     pending -> canceled（peopleDelta=0）
//     success -> canceled（peopleDelta=-1，释放名额）
//     rejected/canceled/invalid -> error
//
// 注意：状态机只返回“迁移决策”，真正的落库与人数增减由调用方在事务内执行。
type signupTransitionPlan struct {
	toStatus        int32
	peopleDelta     int // 对活动 current_people 的变更: +1 占用名额, -1 释放名额, 0 不变
	apply           bool
	createIfMissing bool
}

func resolveSignupTransition(action signupTransitionType, signup *model.ActivitySignup) (*signupTransitionPlan, error) {
	if signup == nil {
		switch action {
		case signupTransitionApprove:
			return &signupTransitionPlan{
				toStatus:        model.ActivitySignupStatusSuccess,
				peopleDelta:     1,
				apply:           true,
				createIfMissing: true,
			}, nil
		case signupTransitionReject:
			return &signupTransitionPlan{
				toStatus:        model.ActivitySignupStatusRejected,
				peopleDelta:     0,
				apply:           true,
				createIfMissing: true,
			}, nil
		case signupTransitionCancel:
			return nil, errors.New("报名记录不存在")
		default:
			return nil, errors.New("报名状态流转动作不合法")
		}
	}

	switch action {
	case signupTransitionApprove:
		switch signup.Status {
		case model.ActivitySignupStatusSuccess:
			return &signupTransitionPlan{
				toStatus:    signup.Status,
				peopleDelta: 0,
				apply:       false,
			}, nil
		case model.ActivitySignupStatusPending, model.ActivitySignupStatusRejected, model.ActivitySignupStatusCanceled:
			return &signupTransitionPlan{
				toStatus:    model.ActivitySignupStatusSuccess,
				peopleDelta: 1,
				apply:       true,
			}, nil
		default:
			return nil, errors.New("报名状态异常")
		}
	case signupTransitionReject:
		switch signup.Status {
		case model.ActivitySignupStatusSuccess, model.ActivitySignupStatusRejected, model.ActivitySignupStatusCanceled:
			return &signupTransitionPlan{
				toStatus:    signup.Status,
				peopleDelta: 0,
				apply:       false,
			}, nil
		case model.ActivitySignupStatusPending:
			return &signupTransitionPlan{
				toStatus:    model.ActivitySignupStatusRejected,
				peopleDelta: 0,
				apply:       true,
			}, nil
		default:
			return nil, errors.New("报名状态异常")
		}
	case signupTransitionCancel:
		switch signup.Status {
		case model.ActivitySignupStatusPending:
			return &signupTransitionPlan{
				toStatus:    model.ActivitySignupStatusCanceled,
				peopleDelta: 0,
				apply:       true,
			}, nil
		case model.ActivitySignupStatusSuccess:
			return &signupTransitionPlan{
				toStatus:    model.ActivitySignupStatusCanceled,
				peopleDelta: -1,
				apply:       true,
			}, nil
		default:
			return nil, errors.New("当前状态不允许取消")
		}
	default:
		return nil, errors.New("报名状态流转动作不合法")
	}
}
