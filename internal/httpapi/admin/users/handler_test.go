package users

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"ds2api/internal/auth"
	"ds2api/internal/database"
)

func newTestHandler(t *testing.T) (*database.DB, http.Handler) {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	router := chi.NewRouter()
	RegisterRoutes(router, NewHandler(db))
	return db, router
}

func adminContext(userID int64, username string) context.Context {
	return auth.SetUserContext(context.Background(), userID, username, "admin")
}

func performJSON(handler http.Handler, method, path string, body []byte, ctx context.Context) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if ctx != nil {
		req = req.WithContext(ctx)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestUpdateUserRejectsInvalidRole(t *testing.T) {
	db, handler := newTestHandler(t)
	admin, err := db.CreateUser("admin", "admin@example.com", "password1", "admin")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	user, err := db.CreateUser("userone", "user@example.com", "password1", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	body := []byte(`{"username":"userone","email":"user@example.com","role":"owner"}`)
	rec := performJSON(handler, http.MethodPut, "/users/"+itoa(user.ID), body, adminContext(admin.ID, admin.Username))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateUserBlocksSelfDemotion(t *testing.T) {
	db, handler := newTestHandler(t)
	admin, err := db.CreateUser("admin", "admin@example.com", "password1", "admin")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if _, err := db.CreateUser("admin2", "admin2@example.com", "password1", "admin"); err != nil {
		t.Fatalf("create second admin: %v", err)
	}

	body := []byte(`{"username":"admin","email":"admin@example.com","role":"user"}`)
	rec := performJSON(handler, http.MethodPut, "/users/"+itoa(admin.ID), body, adminContext(admin.ID, admin.Username))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateUserBlocksLastAdminDemotion(t *testing.T) {
	db, handler := newTestHandler(t)
	admin, err := db.CreateUser("admin", "admin@example.com", "password1", "admin")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}

	body := []byte(`{"username":"admin","email":"admin@example.com","role":"user"}`)
	rec := performJSON(handler, http.MethodPut, "/users/"+itoa(admin.ID), body, adminContext(999, "root"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteUserBlocksLastAdmin(t *testing.T) {
	db, handler := newTestHandler(t)
	admin, err := db.CreateUser("admin", "admin@example.com", "password1", "admin")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}

	rec := performJSON(handler, http.MethodDelete, "/users/"+itoa(admin.ID), nil, adminContext(999, "root"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRoleChangeRevokesSessionsAndReturnsFinalRole(t *testing.T) {
	db, handler := newTestHandler(t)
	admin, err := db.CreateUser("admin", "admin@example.com", "password1", "admin")
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	user, err := db.CreateUser("userone", "user@example.com", "password1", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := db.CreateSession(user.ID, "token-one", time.Hour); err != nil {
		t.Fatalf("create session: %v", err)
	}

	body := []byte(`{"username":"userone","email":"user@example.com","role":"admin"}`)
	rec := performJSON(handler, http.MethodPut, "/users/"+itoa(user.ID), body, adminContext(admin.ID, admin.Username))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response struct {
		Role string `json:"role"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Role != "admin" {
		t.Fatalf("expected role admin, got %q", response.Role)
	}

	count, err := db.CountUserActiveSessions(user.ID)
	if err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected sessions revoked, got %d", count)
	}
}

func itoa(value int64) string {
	return strconv.FormatInt(value, 10)
}
