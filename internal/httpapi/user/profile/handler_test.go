package profile

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"ds2api/internal/auth"
	"ds2api/internal/config"
	"ds2api/internal/database"
)

func TestChangePasswordRequiresEightCharacters(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	user, err := db.CreateUser("user", "user@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/user/profile/change-password", bytes.NewBufferString(`{"current_password":"password123","new_password":"short"}`))
	req = req.WithContext(auth.SetUserContext(context.Background(), user.ID, user.Username, user.Role))
	rec := httptest.NewRecorder()

	NewHandler(db).ChangePassword(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetSettingsReturnsReadOnlySafeSubset(t *testing.T) {
	t.Setenv("DS2API_CONFIG_JSON", `{
		"responses":{"store_ttl_seconds":600},
		"embeddings":{"provider":"deterministic"},
		"auto_delete":{"mode":"single"},
		"current_input_file":{"enabled":false,"min_chars":123},
		"thinking_injection":{"enabled":false,"prompt":"custom prompt"},
		"model_aliases":{"gpt-test":"deepseek-test"}
	}`)

	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	user, err := db.CreateUser("user", "user@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/user/profile/settings", nil)
	req = req.WithContext(auth.SetUserContext(context.Background(), user.ID, user.Username, user.Role))
	rec := httptest.NewRecorder()

	NewHandler(db, config.LoadStore()).GetSettings(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["admin"] != nil || body["runtime"] != nil {
		t.Fatalf("response leaked admin/runtime settings: %s", rec.Body.String())
	}
	if body["read_only"] != true {
		t.Fatalf("expected read_only=true, got %v", body["read_only"])
	}
}
