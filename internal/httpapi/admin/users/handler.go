package users

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"ds2api/internal/auth"
	"ds2api/internal/database"
)

type Handler struct {
	db *database.DB
}

func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

// ListUsers handles GET /admin/users
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	// Get all users (no pagination for now)
	users, total, err := h.db.ListUsers(1000, 0)
	if err != nil {
		http.Error(w, `{"error":"failed to list users"}`, http.StatusInternalServerError)
		return
	}

	// Get counts for each user
	type UserWithCounts struct {
		ID            int64  `json:"id"`
		Username      string `json:"username"`
		Email         string `json:"email"`
		Role          string `json:"role"`
		CreatedAt     string `json:"created_at"`
		AccountsCount int    `json:"accounts_count"`
		KeysCount     int    `json:"keys_count"`
		SessionsCount int    `json:"sessions_count"`
	}

	result := make([]UserWithCounts, 0, len(users))
	for _, user := range users {
		accountsCount, err := h.db.CountUserAccounts(user.ID)
		if err != nil {
			http.Error(w, `{"error":"failed to count user accounts"}`, http.StatusInternalServerError)
			return
		}
		keysCount, err := h.db.CountUserKeys(user.ID)
		if err != nil {
			http.Error(w, `{"error":"failed to count user keys"}`, http.StatusInternalServerError)
			return
		}
		sessionsCount, err := h.db.CountUserActiveSessions(user.ID)
		if err != nil {
			http.Error(w, `{"error":"failed to count user sessions"}`, http.StatusInternalServerError)
			return
		}

		result = append(result, UserWithCounts{
			ID:            user.ID,
			Username:      user.Username,
			Email:         user.Email,
			Role:          user.Role,
			CreatedAt:     user.CreatedAt.Format("2006-01-02 15:04:05"),
			AccountsCount: accountsCount,
			KeysCount:     keysCount,
			SessionsCount: sessionsCount,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users": result,
		"total": total,
	})
}

// CreateUser handles POST /admin/users
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate
	if req.Username == "" || req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"username, email and password are required"}`, http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, `{"error":"password must be at least 8 characters"}`, http.StatusBadRequest)
		return
	}

	if req.Role == "" {
		req.Role = "user"
	}

	if req.Role != "user" && req.Role != "admin" {
		http.Error(w, `{"error":"role must be 'user' or 'admin'"}`, http.StatusBadRequest)
		return
	}

	// Check if username exists
	_, err := h.db.GetUserByUsername(req.Username)
	if err == nil {
		http.Error(w, `{"error":"username already exists"}`, http.StatusConflict)
		return
	}

	// Check if email exists
	_, err = h.db.GetUserByEmail(req.Email)
	if err == nil {
		http.Error(w, `{"error":"email already exists"}`, http.StatusConflict)
		return
	}

	// Create user
	user, err := h.db.CreateUser(req.Username, req.Email, req.Password, req.Role)
	if err != nil {
		http.Error(w, `{"error":"failed to create user"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"role":     user.Role,
	})
}

// UpdateUser handles PUT /admin/users/{id}
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Get existing user
	user, err := h.db.GetUserByID(userID)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	// Update fields
	usernameToUpdate := user.Username
	emailToUpdate := user.Email

	if req.Username != "" && req.Username != user.Username {
		// Check if new username exists
		_, err := h.db.GetUserByUsername(req.Username)
		if err == nil {
			http.Error(w, `{"error":"username already exists"}`, http.StatusConflict)
			return
		}
		usernameToUpdate = req.Username
	}

	if req.Email != "" && req.Email != user.Email {
		// Check if new email exists
		_, err := h.db.GetUserByEmail(req.Email)
		if err == nil {
			http.Error(w, `{"error":"email already exists"}`, http.StatusConflict)
			return
		}
		emailToUpdate = req.Email
	}

	// Update username and email
	if err := h.db.UpdateUser(userID, usernameToUpdate, emailToUpdate); err != nil {
		http.Error(w, `{"error":"failed to update user"}`, http.StatusInternalServerError)
		return
	}

	// Update role if provided
	if req.Role != "" && (req.Role == "user" || req.Role == "admin") && req.Role != user.Role {
		if err := h.db.UpdateUserRole(userID, req.Role); err != nil {
			http.Error(w, `{"error":"failed to update role"}`, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       userID,
		"username": usernameToUpdate,
		"email":    emailToUpdate,
		"role":     req.Role,
	})
}

// ChangePassword handles PUT /admin/users/{id}/password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if len(req.Password) < 8 {
		http.Error(w, `{"error":"password must be at least 8 characters"}`, http.StatusBadRequest)
		return
	}

	// Get user
	user, err := h.db.GetUserByID(userID)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	// Update password
	if err := h.db.UpdateUserPassword(user.ID, req.Password); err != nil {
		http.Error(w, `{"error":"failed to update password"}`, http.StatusInternalServerError)
		return
	}

	// Delete all sessions for this user (force re-login)
	h.db.DeleteUserSessions(user.ID)

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"password updated successfully"}`))
}

// DeleteUser handles DELETE /admin/users/{id}
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid user id"}`, http.StatusBadRequest)
		return
	}

	// Check if user is trying to delete themselves
	userCtx, ok := auth.GetUserContext(r.Context())
	if ok && userCtx.UserID == userID {
		http.Error(w, `{"error":"cannot delete yourself"}`, http.StatusBadRequest)
		return
	}

	// Get user
	user, err := h.db.GetUserByID(userID)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	// Delete user (cascade will delete accounts, keys, sessions)
	if err := h.db.DeleteUser(user.ID); err != nil {
		http.Error(w, `{"error":"failed to delete user"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"user deleted successfully"}`))
}
