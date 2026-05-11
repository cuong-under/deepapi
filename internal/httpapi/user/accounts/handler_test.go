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
