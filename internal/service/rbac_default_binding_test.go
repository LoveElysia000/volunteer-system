package service

import (
	"testing"
	"volunteer-system/internal/model"
)

func TestServiceBindDefaultRole(t *testing.T) {
	svc := newLoginServiceWithRBACTestDB(t)

	if err := svc.Service.bindDefaultRole(svc.repo.DB, 11, model.RBACRoleVolunteer, model.RBACScopeGlobal, 0); err != nil {
		t.Fatalf("bind default role failed: %v", err)
	}

	var count int64
	err := svc.repo.DB.Table("rbac_account_roles ar").
		Joins("JOIN rbac_roles r ON r.id = ar.role_id").
		Where("ar.account_id = ? AND r.role_code = ? AND ar.scope_type = ? AND ar.scope_id = ?", 11, "volunteer", "global", 0).
		Count(&count).Error
	if err != nil {
		t.Fatalf("count binding failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 binding, got %d", count)
	}
}

func TestServiceEnsureDefaultRBACBinding(t *testing.T) {
	svc := newLoginServiceWithRBACTestDB(t)
	account, err := svc.repo.FindByID(svc.repo.DB, 12)
	if err != nil {
		t.Fatalf("query account failed: %v", err)
	}

	if err := svc.Service.ensureDefaultRBACBinding(svc.repo.DB, account); err != nil {
		t.Fatalf("ensure default rbac binding failed: %v", err)
	}

	var count int64
	err = svc.repo.DB.Table("rbac_account_roles ar").
		Joins("JOIN rbac_roles r ON r.id = ar.role_id").
		Where("ar.account_id = ? AND r.role_code = ? AND ar.scope_type = ? AND ar.scope_id = ?", 12, "org_owner", "org", 1001).
		Count(&count).Error
	if err != nil {
		t.Fatalf("count binding failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 binding, got %d", count)
	}
}
