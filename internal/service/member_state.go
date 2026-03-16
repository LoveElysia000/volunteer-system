package service

import (
	"errors"
	"volunteer-system/internal/model"
)

type memberTransitionFlow string

const (
	flowJoinReapply memberTransitionFlow = "join_reapply"
	flowLeaveApply  memberTransitionFlow = "leave_apply"
	flowAdminUpdate memberTransitionFlow = "admin_update"
)

// 成员状态机说明（仅负责“能否流转”的规则校验）：
//  1. join_reapply（志愿者重新申请加入）:
//     rejected/left -> allow（后续走审核流，由审核结果驱动最终状态）
//     active/pending -> deny
//  2. leave_apply（志愿者申请退出）:
//     pending/active/rejected -> allow（后续走审核流，目标状态为 left）
//     left -> deny
//  3. admin_update（管理员直接改状态）:
//     pending -> active/rejected/left
//     active -> rejected/left
//     rejected -> active/left
//     left -> active/rejected
//
// 说明：该文件只做状态合法性判定，不直接执行数据库更新与副作用。
type transitionDecision struct {
	allow        bool
	rejectReason string
}

var (
	joinReapplyTransitions = map[int32]transitionDecision{
		model.MemberStatusActive: {
			allow:        false,
			rejectReason: "成员关系已存在或正在审核中",
		},
		model.MemberStatusPending: {
			allow:        false,
			rejectReason: "成员关系已存在或正在审核中",
		},
		model.MemberStatusRejected: {allow: true},
		model.MemberStatusLeft:     {allow: true},
	}

	leaveApplyTransitions = map[int32]transitionDecision{
		model.MemberStatusPending:  {allow: true},
		model.MemberStatusActive:   {allow: true},
		model.MemberStatusRejected: {allow: true},
		model.MemberStatusLeft: {
			allow:        false,
			rejectReason: "该成员已退出组织",
		},
	}

	adminUpdateTransitions = map[int32]map[int32]transitionDecision{
		model.MemberStatusPending: {
			model.MemberStatusActive:   {allow: true},
			model.MemberStatusRejected: {allow: true},
			model.MemberStatusLeft:     {allow: true},
		},
		model.MemberStatusActive: {
			model.MemberStatusRejected: {allow: true},
			model.MemberStatusLeft:     {allow: true},
		},
		model.MemberStatusRejected: {
			model.MemberStatusActive: {allow: true},
			model.MemberStatusLeft:   {allow: true},
		},
		model.MemberStatusLeft: {
			model.MemberStatusActive:   {allow: true},
			model.MemberStatusRejected: {allow: true},
		},
	}
)

func validateMemberTransition(flow memberTransitionFlow, fromStatus, toStatus int32) error {
	switch flow {
	case flowJoinReapply:
		return validateMemberTransitionBySource(joinReapplyTransitions, fromStatus, "成员关系状态异常")
	case flowLeaveApply:
		return validateMemberTransitionBySource(leaveApplyTransitions, fromStatus, "成员关系状态异常")
	case flowAdminUpdate:
		targetMap, ok := adminUpdateTransitions[fromStatus]
		if !ok {
			return errors.New("成员关系状态异常")
		}
		decision, ok := targetMap[toStatus]
		if !ok || !decision.allow {
			if ok && decision.rejectReason != "" {
				return errors.New(decision.rejectReason)
			}
			return errors.New("状态流转不合法")
		}
		return nil
	default:
		return errors.New("状态流转不合法")
	}
}

func validateMemberTransitionBySource(
	decisions map[int32]transitionDecision,
	fromStatus int32,
	defaultErr string,
) error {
	decision, ok := decisions[fromStatus]
	if !ok {
		return errors.New(defaultErr)
	}
	if decision.allow {
		return nil
	}
	if decision.rejectReason != "" {
		return errors.New(decision.rejectReason)
	}
	return errors.New(defaultErr)
}
