package repository

import (
	"testing"
	"volunteer-system/internal/model"
)

func TestRBACRepositoryTypesAlignWithGeneratedModels(t *testing.T) {
	var role *RBACRole
	var permission *RBACPermission

	// If these assignments stop compiling, repository types drifted from generated schema types.
	var _ *model.RbacRole = role
	var _ *model.RbacPermission = permission
}
