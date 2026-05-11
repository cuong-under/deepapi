package users

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"ds2api/internal/auth"
	"ds2api/internal/database"
	"ds2api/internal/validation"
)

type Handler struct {
	db *database.DB
}

func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

// ListUsers handles GET /admin/users
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	pageSize := parsePositiveInt(r.URL.Query().Get("page_size"), 25)
	if pageSize > 100 {
		pageSize = 100
	}

	role := strings.TrimSpace(r.URL.Query().Get("role"))
	if role != "" && role != "user" && role != "admin" {
		http.Error(w, `{"error":"role must be 'user' or 'admin'"}`, http.StatusBadRequest)
		return
	}

	users, total, err := h.db.ListUsersWithCounts(database.UserListOptions{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
		Search: r.URL.Query().Get("q"),
		Role:   role,
	})
	if err != nil {
		http.Error(w, `{"error":"failed to list users"}`, http.StatusInternalServerError)
		return
	}

	result := make([]map[string]interface{}, 0, len(users))
	for _, user := range users {
		result = append(result, map[string]interface{}{
			"id":             user.ID,
			"username":       user.Username,
			"email":          user.Email,
			"role":           user.Role,
			"created_at":     user.CreatedAt.Format("2006-01-02 15:04:05"),
			"accounts_count": user.AccountsCount,
			"keys_count":     user.KeysCount,
			"sessions_count": user.SessionsCount,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"users":     result,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
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

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.Role = strings.TrimSpace(req.Role)

	validator := validation.UserValidator{}
	if err := validator.ValidateUsername(req.Username); err != nil {
		writeValidationError(w, err)
		return
	}
	if err := validator.ValidateEmail(req.Email); err != nil {
		writeValidationError(w, err)
		return
	}
	if err := validator.ValidatePasswordSimple(req.Password); err != nil {
		writeValidationError(w, err)
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

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.Role = strings.TrimSpace(req.Role)

	// Get existing user
	user, err := h.db.GetUserByID(userID)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}
	if req.Username == "" {
		req.Username = user.Username
	}
	if req.Email == "" {
		req.Email = user.Email
	}

	validator := validation.UserValidator{}
	if err := validator.ValidateUsername(req.Username); err != nil {
		writeValidationError(w, err)
		return
	}
	if err := validator.ValidateEmail(req.Email); err != nil {
		writeValidationError(w, err)
		return
	}
	if req.Role != "" && req.Role != "user" && req.Role != "admin" {
		http.Error(w, `{"error":"role must be 'user' or 'admin'"}`, http.StatusBadRequest)
		return
	}

	usernameToUpdate := user.Username
	emailToUpdate := user.Email
	roleToUpdate := user.Role

	if req.Username != user.Username {
		_, err := h.db.GetUserByUsername(req.Username)
		if err == nil {
			http.Error(w, `{"error":"username already exists"}`, http.StatusConflict)
			return
		}
		usernameToUpdate = req.Username
	}

	if req.Email != user.Email {
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

	if req.Role != "" && req.Role != user.Role {
		if user.Role == "admin" && req.Role == "user" {
			userCtx, ok := auth.GetUserContext(r.Context())
			if ok && userCtx.UserID == user.ID {
				http.Error(w, `{"error":"cannot demote yourself"}`, http.StatusBadRequest)
				return
			}

			admins, err := h.db.CountAdmins()
			if err != nil {
				http.Error(w, `{"error":"failed to count admins"}`, http.StatusInternalServerError)
				return
			}
			if admins <= 1 {
				http.Error(w, `{"error":"cannot demote the last admin"}`, http.StatusBadRequest)
				return
			}
		}
		if err := h.db.UpdateUserRole(userID, req.Role); err != nil {
			http.Error(w, `{"error":"failed to update role"}`, http.StatusInternalServerError)
			return
		}
		if err := h.db.DeleteUserSessions(userID); err != nil {
			http.Error(w, `{"error":"failed to revoke user sessions"}`, http.StatusInternalServerError)
			return
		}
		roleToUpdate = req.Role
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       userID,
		"username": usernameToUpdate,
		"email":    emailToUpdate,
		"role":     roleToUpdate,
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

	validator := validation.UserValidator{}
	if err := validator.ValidatePasswordSimple(req.Password); err != nil {
		writeValidationError(w, err)
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

	if err := h.db.DeleteUserSessions(user.ID); err != nil {
		http.Error(w, `{"error":"failed to revoke user sessions"}`, http.StatusInternalServerError)
		return
	}

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
	if user.Role == "admin" {
		admins, err := h.db.CountAdmins()
		if err != nil {
			http.Error(w, `{"error":"failed to count admins"}`, http.StatusInternalServerError)
			return
		}
		if admins <= 1 {
			http.Error(w, `{"error":"cannot delete the last admin"}`, http.StatusBadRequest)
			return
		}
	}

	// Delete user (cascade will delete accounts, keys, sessions)
	if err := h.db.DeleteUser(user.ID); err != nil {
		http.Error(w, `{"error":"failed to delete user"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"user deleted successfully"}`))
}

func parsePositiveInt(raw string, fallback int) int {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func writeValidationError(w http.ResponseWriter, err error) {
	http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
}
