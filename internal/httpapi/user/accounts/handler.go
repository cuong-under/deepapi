package accounts

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"ds2api/internal/auth"
	"ds2api/internal/config"
	"ds2api/internal/database"
	dsclient "ds2api/internal/deepseek/client"
)

// Handler handles user account operations
type Handler struct {
	db *database.DB
	ds *dsclient.Client
}

// NewHandler creates a new accounts handler
func NewHandler(db *database.DB, ds *dsclient.Client) *Handler {
	return &Handler{
		db: db,
		ds: ds,
	}
}

// CreateAccountRequest represents create account request
type CreateAccountRequest struct {
	Name     string `json:"name"`
	Remark   string `json:"remark"`
	Email    string `json:"email"`
	Mobile   string `json:"mobile"`
	Password string `json:"password"`
	ProxyID  string `json:"proxy_id"`
}

// UpdateAccountRequest represents update account request
type UpdateAccountRequest struct {
	Name     string `json:"name"`
	Remark   string `json:"remark"`
	Email    string `json:"email"`
	Mobile   string `json:"mobile"`
	Password string `json:"password"`
	ProxyID  string `json:"proxy_id"`
}

type AccountResponse struct {
	ID              int64  `json:"id"`
	UserID          int64  `json:"user_id"`
	Name            string `json:"name"`
	Remark          string `json:"remark"`
	Email           string `json:"email"`
	Mobile          string `json:"mobile"`
	ProxyID         string `json:"proxy_id"`
	CreatedAt       int64  `json:"created_at"`
	LastRefreshedAt *int64 `json:"last_refreshed_at,omitempty"`
	HasPassword     bool   `json:"has_password"`
}

func accountResponse(account *database.UserAccount) AccountResponse {
	return AccountResponse{
		ID:              account.ID,
		UserID:          account.UserID,
		Name:            account.Name,
		Remark:          account.Remark,
		Email:           account.Email,
		Mobile:          account.Mobile,
		ProxyID:         account.ProxyID,
		CreatedAt:       account.CreatedAt,
		LastRefreshedAt: account.LastRefreshedAt,
		HasPassword:     account.Password != "",
	}
}

func accountResponses(accounts []*database.UserAccount) []AccountResponse {
	responses := make([]AccountResponse, 0, len(accounts))
	for _, account := range accounts {
		responses = append(responses, accountResponse(account))
	}
	return responses
}

// ListAccounts lists all accounts for current user
func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error":"user not authenticated"}`, http.StatusUnauthorized)
		return
	}

	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	pageSize := parsePositiveInt(r.URL.Query().Get("page_size"), 10)
	if pageSize > 100 {
		pageSize = 100
	}

	accounts, total, err := h.db.ListAccountsByUserID(userID, database.UserAccountListOptions{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
		Search: r.URL.Query().Get("q"),
	})
	if err != nil {
		http.Error(w, `{"error":"failed to get accounts"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts":    accountResponses(accounts),
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": max(1, (total+pageSize-1)/pageSize),
	})
}

// GetAccount gets a specific account
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error":"user not authenticated"}`, http.StatusUnauthorized)
		return
	}

	accountIDStr := chi.URLParam(r, "id")
	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid account id"}`, http.StatusBadRequest)
		return
	}

	account, err := h.db.GetAccountByID(accountID)
	if err != nil {
		http.Error(w, `{"error":"account not found"}`, http.StatusNotFound)
		return
	}

	// User routes are scoped to the current principal, including admins.
	if account.UserID != userID {
		http.Error(w, `{"error":"access denied"}`, http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accountResponse(account))
}

// CreateAccount creates a new account
func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error":"user not authenticated"}`, http.StatusUnauthorized)
		return
	}

	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	req.Mobile = strings.TrimSpace(req.Mobile)
	req.Name = strings.TrimSpace(req.Name)
	req.Remark = strings.TrimSpace(req.Remark)
	req.ProxyID = strings.TrimSpace(req.ProxyID)

	// Validate: must have email or mobile
	if req.Email == "" && req.Mobile == "" {
		http.Error(w, `{"error":"email or mobile is required"}`, http.StatusBadRequest)
		return
	}

	account, err := h.db.CreateAccount(userID, req.Name, req.Remark, req.Email, req.Mobile, req.Password, req.ProxyID)
	if err != nil {
		http.Error(w, `{"error":"failed to create account"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(accountResponse(account))
}

// UpdateAccount updates an existing account
func (h *Handler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error":"user not authenticated"}`, http.StatusUnauthorized)
		return
	}

	accountIDStr := chi.URLParam(r, "id")
	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid account id"}`, http.StatusBadRequest)
		return
	}

	var req UpdateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	req.Mobile = strings.TrimSpace(req.Mobile)
	req.Name = strings.TrimSpace(req.Name)
	req.Remark = strings.TrimSpace(req.Remark)
	req.ProxyID = strings.TrimSpace(req.ProxyID)

	existing, err := h.db.GetAccountByID(accountID)
	if err != nil {
		http.Error(w, `{"error":"account not found"}`, http.StatusNotFound)
		return
	}
	if existing.UserID != userID {
		http.Error(w, `{"error":"access denied"}`, http.StatusForbidden)
		return
	}
	password := req.Password
	if password == "" {
		password = existing.Password
	}

	// Update with ownership check
	err = h.db.UpdateAccount(accountID, userID, req.Name, req.Remark, req.Email, req.Mobile, password, req.ProxyID)
	if err != nil {
		if err.Error() == "account not found or access denied" {
			http.Error(w, `{"error":"account not found or access denied"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error":"failed to update account"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"account updated successfully"}`))
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// DeleteAccount deletes an account
func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error":"user not authenticated"}`, http.StatusUnauthorized)
		return
	}

	accountIDStr := chi.URLParam(r, "id")
	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid account id"}`, http.StatusBadRequest)
		return
	}

	// Delete with ownership check
	err = h.db.DeleteAccount(accountID, userID)
	if err != nil {
		if err.Error() == "account not found or access denied" {
			http.Error(w, `{"error":"account not found or access denied"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error":"failed to delete account"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"account deleted successfully"}`))
}

// RefreshToken refreshes the DeepSeek token for an account
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error":"user not authenticated"}`, http.StatusUnauthorized)
		return
	}

	accountIDStr := chi.URLParam(r, "id")
	accountID, err := strconv.ParseInt(accountIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid account id"}`, http.StatusBadRequest)
		return
	}

	log.Printf("[RefreshToken] user_id=%d, account_id=%d", userID, accountID)

	// Get account with ownership check
	account, err := h.db.GetAccountByID(accountID)
	if err != nil {
		log.Printf("[RefreshToken] failed to get account: %v", err)
		http.Error(w, `{"error":"account not found"}`, http.StatusNotFound)
		return
	}

	// Check ownership
	if account.UserID != userID {
		log.Printf("[RefreshToken] access denied: account.UserID=%d, userID=%d", account.UserID, userID)
		http.Error(w, `{"error":"access denied"}`, http.StatusForbidden)
		return
	}

	log.Printf("[RefreshToken] attempting login for account: email=%s, mobile=%s", account.Email, account.Mobile)

	// Test login to verify credentials and get new token
	configAccount := config.Account{
		Email:    account.Email,
		Mobile:   account.Mobile,
		Password: account.Password,
	}

	// Use the DS client to login and verify account is still valid
	ctx := r.Context()
	_, err = h.ds.Login(ctx, configAccount)
	if err != nil {
		log.Printf("[RefreshToken] login failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to refresh token: " + err.Error(),
		})
		return
	}

	log.Printf("[RefreshToken] login successful, updating refresh time")

	// Update last_refreshed_at timestamp
	if err := h.db.UpdateAccountRefreshTime(accountID); err != nil {
		log.Printf("[RefreshToken] failed to update refresh time: %v", err)
		// Don't fail the request, just log the error
	}

	log.Printf("[RefreshToken] token refreshed successfully")

	// Token is refreshed successfully (stored in memory by DS client)
	// No need to save to database - token will be used from memory
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Token refreshed successfully",
	})
}
