package service

import (
	"strconv"
	"time"

	"volunteer-system/internal/model"
)

func (s *Service) publishSignupApprovedNotification(signupID, actorID int64) {
	signup, err := s.repo.GetActivitySignupByID(s.repo.DB, signupID)
	if err != nil || signup == nil {
		return
	}
	activity, err := s.repo.GetActivityByID(s.repo.DB, signup.ActivityID)
	if err != nil || activity == nil {
		return
	}
	volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, signup.VolunteerID)
	if err != nil || volunteer == nil || volunteer.AccountID <= 0 {
		return
	}

	PublishNotificationEvent(NotificationEvent{
		EventType:   model.NotificationEventSignupApproved,
		BizType:     model.NotificationBizTypeActivity,
		BizID:       signup.ID,
		SourceOrgID: activity.OrgID,
		ActorID:     actorID,
		CreatedAt:   time.Now(),
		Payload: map[string]any{
			"receiverAccountID": volunteer.AccountID,
			"activityTitle":     activity.Title,
		},
		DedupeKey: "signup.approved:" + formatNotificationID(signup.ID),
	})
}

func (s *Service) publishWorkHourNotification(eventType string, signupID, actorID int64, grantedHours float64, reason string) {
	signup, err := s.repo.GetActivitySignupByID(s.repo.DB, signupID)
	if err != nil || signup == nil {
		return
	}
	activity, err := s.repo.GetActivityByID(s.repo.DB, signup.ActivityID)
	if err != nil || activity == nil {
		return
	}
	volunteer, err := s.repo.FindVolunteerByID(s.repo.DB, signup.VolunteerID)
	if err != nil || volunteer == nil || volunteer.AccountID <= 0 {
		return
	}

	payload := map[string]any{
		"receiverAccountID": volunteer.AccountID,
		"activityTitle":     activity.Title,
		"grantedHours":      grantedHours,
	}
	if reason != "" {
		payload["reason"] = reason
	}

	PublishNotificationEvent(NotificationEvent{
		EventType:   eventType,
		BizType:     model.NotificationBizTypeActivity,
		BizID:       signup.ID,
		SourceOrgID: activity.OrgID,
		ActorID:     actorID,
		CreatedAt:   time.Now(),
		Payload:     payload,
		DedupeKey:   buildNotificationDedupeKey(NotificationEvent{EventType: eventType, BizID: signup.ID}),
	})
}

func formatNotificationID(id int64) string {
	return strconv.FormatInt(id, 10)
}
