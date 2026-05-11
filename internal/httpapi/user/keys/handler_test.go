package keys

import (
	"bytes"
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

func TestListKeysMasksKeyAndScopesAdminToSelf(t *testing.T) {
	db := openTestDB(t)
	admin, err := db.CreateUser("admin", "admin@example.com", "password123", "admin")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	user, err := db.CreateUser("user", "user@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := db.CreateAPIKey(admin.ID, "sk-admin-secret-value", "admin-key", ""); err != nil {
		t.Fatalf("create admin key: %v", err)
	}
	if _, err := db.CreateAPIKey(user.ID, "sk-user-secret-value", "user-key", ""); err != nil {
		t.Fatalf("create user key: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/user/keys", nil)
	req = req.WithContext(withUser(req.Context(), admin.ID, admin.Username, admin.Role))
	rec := httptest.NewRecorder()

	NewHandler(db).ListKeys(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if strings.Contains(body, "sk-admin-secret-value") || strings.Contains(body, "sk-user-secret-value") || strings.Contains(body, `"api_key"`) {
		t.Fatalf("response leaked API key: %s", body)
	}

	var payload struct {
		Keys  []KeyResponse `json:"keys"`
		Total int           `json:"total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Total != 1 || len(payload.Keys) != 1 {
		t.Fatalf("expected one admin-scoped key, got total=%d len=%d", payload.Total, len(payload.Keys))
	}
	if payload.Keys[0].UserID != admin.ID || payload.Keys[0].APIKeyPreview == "" {
		t.Fatalf("unexpected key response: %+v", payload.Keys[0])
	}
}

func TestCreateKeyReturnsFullKeyOnce(t *testing.T) {
	db := openTestDB(t)
	user, err := db.CreateUser("user", "user@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/user/keys", bytes.NewBufferString(`{"name":"new","remark":"r"}`))
	req = req.WithContext(withUser(req.Context(), user.ID, user.Username, user.Role))
	rec := httptest.NewRecorder()

	NewHandler(db).CreateKey(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload CreatedKeyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !strings.HasPrefix(payload.APIKey, "sk-") {
		t.Fatalf("expected full generated key, got %+v", payload)
	}
	if payload.APIKeyPreview == "" {
		t.Fatalf("expected preview")
	}
}
