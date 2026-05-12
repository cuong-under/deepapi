package accounts

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"ds2api/internal/auth"
	"ds2api/internal/config"
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
	if !payload.Accounts[0].Enabled {
		t.Fatalf("expected enabled=true")
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

type refreshTokenDSMock struct {
	status int
	body   string
}

func (m refreshTokenDSMock) Login(context.Context, config.Account) (string, error) {
	return "new-token", nil
}

func (m refreshTokenDSMock) CreateSession(context.Context, *auth.RequestAuth, int) (string, error) {
	return "session-id", nil
}

func (m refreshTokenDSMock) GetPow(context.Context, *auth.RequestAuth, int) (string, error) {
	return "pow", nil
}

func (m refreshTokenDSMock) CallCompletion(context.Context, *auth.RequestAuth, map[string]any, string, int) (*http.Response, error) {
	status := m.status
	if status == 0 {
		status = http.StatusOK
	}
	body := m.body
	if body == "" {
		body = "data: {\"p\":\"response/content\",\"v\":\"ok\"}\n\ndata: [DONE]\n\n"
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
}

func TestRefreshTokenChecksDeepSeekAccountStatus(t *testing.T) {
	db := openTestDB(t)
	user, err := db.CreateUser("user", "user@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	account, err := db.CreateAccount(user.ID, "muted", "", "muted@example.com", "", "secret", "")
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if err := db.UpdateAccountRefreshTime(account.ID); err != nil {
		t.Fatalf("seed refresh time: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/user/accounts/1/refresh-token", nil)
	req = req.WithContext(withUser(req.Context(), user.ID, user.Username, user.Role))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", strconv.FormatInt(account.ID, 10))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	rec := httptest.NewRecorder()

	newHandlerWithClient(db, refreshTokenDSMock{
		status: http.StatusTooManyRequests,
		body:   "user is muted",
	}).RefreshToken(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "user is muted") {
		t.Fatalf("expected muted detail, got %s", rec.Body.String())
	}
	updated, err := db.GetAccountByID(account.ID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if updated.LastRefreshedAt != nil {
		t.Fatalf("muted account should not be marked refreshed")
	}
}

func TestSetAccountEnabledScopesToOwner(t *testing.T) {
	db := openTestDB(t)
	user, err := db.CreateUser("user", "user@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	other, err := db.CreateUser("other", "other@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create other: %v", err)
	}
	account, err := db.CreateAccount(user.ID, "main", "", "main@example.com", "", "secret", "")
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/user/accounts/1/enabled", strings.NewReader(`{"enabled":false}`))
	req = req.WithContext(withUser(req.Context(), other.ID, other.Username, other.Role))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", strconv.FormatInt(account.ID, 10))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	rec := httptest.NewRecorder()
	NewHandler(db, nil).SetAccountEnabled(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/api/user/accounts/1/enabled", strings.NewReader(`{"enabled":false}`))
	req = req.WithContext(withUser(req.Context(), user.ID, user.Username, user.Role))
	routeCtx = chi.NewRouteContext()
	routeCtx.URLParams.Add("id", strconv.FormatInt(account.ID, 10))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	rec = httptest.NewRecorder()
	NewHandler(db, nil).SetAccountEnabled(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected ok, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	updated, err := db.GetAccountByID(account.ID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if updated.Enabled {
		t.Fatalf("expected account disabled")
	}
}

func TestUpdateAccountCanToggleEnabled(t *testing.T) {
	db := openTestDB(t)
	user, err := db.CreateUser("user", "user@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	account, err := db.CreateAccount(user.ID, "main", "old", "main@example.com", "", "secret", "proxy-a")
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	body := `{"name":"main","remark":"old","email":"main@example.com","mobile":"","proxy_id":"proxy-a","enabled":false}`
	req := httptest.NewRequest(http.MethodPut, "/api/user/accounts/1", strings.NewReader(body))
	req = req.WithContext(withUser(req.Context(), user.ID, user.Username, user.Role))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", strconv.FormatInt(account.ID, 10))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	rec := httptest.NewRecorder()

	NewHandler(db, nil).UpdateAccount(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected ok, got status=%d body=%s", rec.Code, rec.Body.String())
	}

	updated, err := db.GetAccountByID(account.ID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if updated.Enabled {
		t.Fatalf("expected account disabled via update endpoint")
	}
}
