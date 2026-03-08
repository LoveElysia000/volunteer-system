package service

import (
	"testing"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
)

func forceSingleSQLiteConn(t *testing.T, svc *MembershipService) {
	t.Helper()
	sqlDB, err := svc.repo.DB.DB()
	if err != nil {
		t.Fatalf("open sql db failed: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
}

func TestGetOrganizationMembers_RBACManagerAllowed(t *testing.T) {
	svc := newMembershipServiceForUpdateStatusTest(t)
	forceSingleSQLiteConn(t, svc)

	resp, err := svc.GetOrganizationMembers(&api.OrganizationMembersRequest{
		OrganizationId: 1001,
		Page:           1,
		PageSize:       20,
	})
	if err != nil {
		t.Fatalf("expected RBAC manager allowed, got err: %v", err)
	}
	if resp == nil || resp.Total != 1 || len(resp.List) != 1 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if resp.List[0].Status != model.MemberStatusPending {
		t.Fatalf("unexpected member status: %d", resp.List[0].Status)
	}
}

func TestMembershipStats_RBACManagerAllowed(t *testing.T) {
	svc := newMembershipServiceForUpdateStatusTest(t)
	forceSingleSQLiteConn(t, svc)

	resp, err := svc.MembershipStats(&api.MembershipStatsRequest{OrganizationId: 1001})
	if err != nil {
		t.Fatalf("expected RBAC manager allowed, got err: %v", err)
	}
	if resp == nil {
		t.Fatalf("expected response, got nil")
	}
	if resp.PendingCount != 1 || resp.TotalCount != 1 {
		t.Fatalf("unexpected stats: %#v", resp)
	}
}
