package repository

import (
	"context"
	"testing"
	"volunteer-system/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newUserRepositoryTestDB(t *testing.T) *Repository {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	schema := []string{
		`CREATE TABLE sys_accounts (
			id INTEGER PRIMARY KEY,
			username TEXT NOT NULL,
			mobile TEXT NOT NULL,
			mobile_hash TEXT NOT NULL,
			email TEXT NOT NULL,
			password TEXT NOT NULL,
			identity_type INTEGER NOT NULL,
			status INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME NULL
		)`,
	}
	for _, stmt := range schema {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("create schema failed: %v", err)
		}
	}

	seed := []string{
		`INSERT INTO sys_accounts(id, username, mobile, mobile_hash, email, password, identity_type, status) VALUES (1, 'u1', 'e1', 'h1', 'u1@example.com', 'p', 1, 1)`,
		`INSERT INTO sys_accounts(id, username, mobile, mobile_hash, email, password, identity_type, status) VALUES (2, 'u2', 'e2', 'h2', 'u2@example.com', 'p', 1, 0)`,
		`INSERT INTO sys_accounts(id, username, mobile, mobile_hash, email, password, identity_type, status) VALUES (3, 'u3', 'e3', 'h3', 'u3@example.com', 'p', 2, 1)`,
		`INSERT INTO sys_accounts(id, username, mobile, mobile_hash, email, password, identity_type, status) VALUES (4, 'u4', 'e4', 'h4', 'u4@example.com', 'p', 2, 1)`,
	}
	for _, stmt := range seed {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}

	repo := &Repository{DB: db}
	ctx := context.Background()
	repo.SetContext(&ctx)
	return repo
}

func TestListActiveAccounts(t *testing.T) {
	repo := newUserRepositoryTestDB(t)

	rows, err := repo.ListActiveAccounts(repo.DB, 2, 0)
	if err != nil {
		t.Fatalf("list active accounts failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows with limit=2, got %d", len(rows))
	}
	if rows[0].ID != 1 || rows[1].ID != 3 {
		t.Fatalf("unexpected order: first=%d second=%d", rows[0].ID, rows[1].ID)
	}
	for _, row := range rows {
		if row.Status != model.SysAccountNormal {
			t.Fatalf("expected active status only, got account=%d status=%d", row.ID, row.Status)
		}
	}

	rows, err = repo.ListActiveAccounts(repo.DB, 2, 2)
	if err != nil {
		t.Fatalf("list active accounts with offset failed: %v", err)
	}
	if len(rows) != 1 || rows[0].ID != 4 {
		t.Fatalf("unexpected pagination result: %+v", rows)
	}
}
