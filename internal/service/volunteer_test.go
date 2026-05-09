package service

import (
	"testing"
	"volunteer-system/internal/api"
	"volunteer-system/internal/model"
)

func TestBuildVolunteerListFiltersBuildsQueryMap(t *testing.T) {
	req := &api.VolunteerListRequest{
		Keyword:       "Alice",
		AuditStatuses: []int32{model.VolunteerAuditStatusPending, model.VolunteerAuditStatusApproved},
		Statuses:      []int32{model.VolunteerActiveStatus, model.VolunteerInactiveStatus},
	}

	queryMap, err := buildVolunteerListFilters(req, []int64{11, 12})
	if err != nil {
		t.Fatalf("buildVolunteerListFilters returned error: %v", err)
	}

	gotKeywordIDs, ok := queryMap["v.id IN ?"]
	if !ok {
		t.Fatalf("expected keyword filter to be present")
	}
	ids, ok := gotKeywordIDs.([]int64)
	if !ok {
		t.Fatalf("expected keyword ids type []int64, got %T", gotKeywordIDs)
	}
	if len(ids) != 2 || ids[0] != 11 || ids[1] != 12 {
		t.Fatalf("unexpected keyword ids: %#v", ids)
	}

	gotAuditStatuses, ok := queryMap["v.audit_status IN ?"]
	if !ok {
		t.Fatalf("expected audit status filter to be present")
	}
	auditStatuses, ok := gotAuditStatuses.([]int32)
	if !ok {
		t.Fatalf("expected audit status filter type []int32, got %T", gotAuditStatuses)
	}
	if len(auditStatuses) != 2 || auditStatuses[0] != model.VolunteerAuditStatusPending || auditStatuses[1] != model.VolunteerAuditStatusApproved {
		t.Fatalf("unexpected audit statuses: %#v", auditStatuses)
	}

	gotStatuses, ok := queryMap["v.status IN ?"]
	if !ok {
		t.Fatalf("expected volunteer status filter to be present")
	}
	statuses, ok := gotStatuses.([]int32)
	if !ok {
		t.Fatalf("expected volunteer status filter type []int32, got %T", gotStatuses)
	}
	if len(statuses) != 2 || statuses[0] != model.VolunteerActiveStatus || statuses[1] != model.VolunteerInactiveStatus {
		t.Fatalf("unexpected volunteer statuses: %#v", statuses)
	}
}

func TestBuildVolunteerListFiltersRejectsInvalidAuditStatus(t *testing.T) {
	req := &api.VolunteerListRequest{
		AuditStatuses: []int32{99},
	}

	if _, err := buildVolunteerListFilters(req, nil); err == nil {
		t.Fatal("expected invalid audit status error")
	}
}

func TestBuildVolunteerListFiltersRejectsInvalidVolunteerStatus(t *testing.T) {
	req := &api.VolunteerListRequest{
		Statuses: []int32{99},
	}

	if _, err := buildVolunteerListFilters(req, nil); err == nil {
		t.Fatal("expected invalid volunteer status error")
	}
}
