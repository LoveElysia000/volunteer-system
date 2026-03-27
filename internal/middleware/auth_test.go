package middleware

import (
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
)

func TestGetAccountID(t *testing.T) {
	ctx := app.NewContext(0)
	ctx.Set(AccountIDKey, "123")

	accountID, err := GetAccountID(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if accountID != "123" {
		t.Fatalf("expected account ID 123, got %q", accountID)
	}

	legacyID, err := GetUserID(ctx)
	if err != nil {
		t.Fatalf("expected no error from legacy helper, got %v", err)
	}
	if legacyID != accountID {
		t.Fatalf("expected legacy helper to match account helper, got %q", legacyID)
	}
}

func TestGetAccountIDInt(t *testing.T) {
	ctx := app.NewContext(0)
	ctx.Set(AccountIDKey, "456")

	accountID, err := GetAccountIDInt(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if accountID != 456 {
		t.Fatalf("expected account ID 456, got %d", accountID)
	}

	legacyID, err := GetUserIDInt(ctx)
	if err != nil {
		t.Fatalf("expected no error from legacy helper, got %v", err)
	}
	if legacyID != accountID {
		t.Fatalf("expected legacy helper to match account helper, got %d", legacyID)
	}
}

func TestGetAccountIDMissing(t *testing.T) {
	ctx := app.NewContext(0)

	if _, err := GetAccountID(ctx); err == nil {
		t.Fatal("expected missing account ID to return an error")
	}
	if _, err := GetAccountIDInt(ctx); err == nil {
		t.Fatal("expected missing account ID int conversion to return an error")
	}
}
