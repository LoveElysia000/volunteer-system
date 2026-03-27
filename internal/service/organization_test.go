package service

import (
	"context"
	"testing"

	"volunteer-system/internal/api"
	"volunteer-system/internal/repository"

	"github.com/cloudwego/hertz/pkg/app"
)

func TestPublicOrganizationListRequiresAccountContext(t *testing.T) {
	svc := &OrganizationService{
		Service: Service{
			ctx:  context.Background(),
			c:    app.NewContext(0),
			repo: &repository.Repository{},
		},
	}

	_, err := svc.PublicOrganizationList(&api.OrganizationListRequest{
		Page:     1,
		PageSize: 20,
	})
	if err == nil {
		t.Fatal("expected missing account context to return an error")
	}
}
