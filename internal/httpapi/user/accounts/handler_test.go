package accounts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"ds2api/internal/auth"
	"ds2api/internal/database"
)

func openTestDB(t *testing.T) *database.DB {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func withUser(ctx context.Context, userID int64, username, role string) context.Context {
	return auth.SetUserContext(ctx, userID, username, role)
}

func TestListAccountsMasksPasswordAndScopesAdminToSelf(t *testing.T) {
	db := openTestDB(t)
	admin, err := db.CreateUser("admin", "admin@example.com", "password123", "admin")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	user, err := db.CreateUser("user", "user@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := db.CreateAccount(admin.ID, "admin-account", "", "admin-deepseek@example.com", "", "admin-secret", ""); err != nil {
		t.Fatalf("create admin account: %v", err)
	}
	if _, err := db.CreateAccount(user.ID, "user-account", "", "user-deepseek@example.com", "", "user-secret", ""); err != nil {
		t.Fatalf("create user account: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/user/accounts", nil)
	req = req.WithContext(withUser(req.Context(), admin.ID, admin.Username, admin.Role))
	rec := httptest.NewRecorder()

	NewHandler(db, nil).ListAccounts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() == "" || json.Valid(rec.Body.Bytes()) == false {
		t.Fatalf("invalid json: %s", rec.Body.String())
	}
	if body := rec.Body.String(); strings.Contains(body, "admin-secret") || strings.Contains(body, "user-secret") || strings.Contains(body, `"password"`) {
		t.Fatalf("response leaked password: %s", body)
	}

	var payload struct {
		Accounts []AccountResponse `json:"accounts"`
		Total    int               `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Total != 1 || len(payload.Accounts) != 1 {
		t.Fatalf("expected one admin-scoped account, got total=%d len=%d", payload.Total, len(payload.Accounts))
	}
	if payload.Accounts[0].UserID != admin.ID || payload.Accounts[0].Email != "admin-deepseek@example.com" {
		t.Fatalf("unexpected account: %+v", payload.Accounts[0])
	}
	if !payload.Accounts[0].HasPassword {
		t.Fatalf("expected has_password=true")
	}
}

func TestListAccountsSupportsSearchAndPagination(t *testing.T) {
	db := openTestDB(t)
	user, err := db.CreateUser("user", "user@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	other, err := db.CreateUser("other", "other@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create other user: %v", err)
	}
	for _, item := range []struct {
		name  string
		email string
	}{
		{name: "Alpha", email: "alpha@example.com"},
		{name: "Beta", email: "beta@example.com"},
		{name: "Alpine", email: "alpine@example.com"},
	} {
		if _, err := db.CreateAccount(user.ID, item.name, "", item.email, "", "secret", ""); err != nil {
			t.Fatalf("create account: %v", err)
		}
	}
	if _, err := db.CreateAccount(other.ID, "Alpha other", "", "other-alpha@example.com", "", "secret", ""); err != nil {
		t.Fatalf("create other account: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/user/accounts?q=alp&page=1&page_size=1", nil)
	req = req.WithContext(withUser(req.Context(), user.ID, user.Username, user.Role))
	rec := httptest.NewRecorder()

	NewHandler(db, nil).ListAccounts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Accounts   []AccountResponse `json:"accounts"`
		Total      int               `json:"total"`
		Page       int               `json:"page"`
		PageSize   int               `json:"page_size"`
		TotalPages int               `json:"total_pages"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Total != 2 || len(payload.Accounts) != 1 || payload.Page != 1 || payload.PageSize != 1 || payload.TotalPages != 2 {
		t.Fatalf("unexpected pagination payload: %+v", payload)
	}
	if payload.Accounts[0].UserID != user.ID {
		t.Fatalf("leaked account from another user: %+v", payload.Accounts[0])
	}
}
