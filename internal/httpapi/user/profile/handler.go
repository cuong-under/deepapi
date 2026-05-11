package profile

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"ds2api/internal/auth"
	"ds2api/internal/database"
)

// Handler handles user profile operations
type Handler struct {
	db *database.DB
}

// NewHandler creates a new profile handler
func NewHandler(db *database.DB) *Handler {
	return &Handler{
		db: db,
	}
}

// ChangePasswordRequest represents change password request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword changes the current user's password
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error":"user not authenticated"}`, http.StatusUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Validate password length
	if len(req.NewPassword) < 4 {
		http.Error(w, `{"error":"password must be at least 4 characters"}`, http.StatusBadRequest)
		return
	}

	// Verify current password
	user, err := h.db.GetUserByID(userID)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		http.Error(w, `{"error":"current password is incorrect"}`, http.StatusUnauthorized)
		return
	}

	// Update password (database will hash it)
	if err := h.db.UpdateUserPassword(userID, req.NewPassword); err != nil {
		http.Error(w, `{"error":"failed to update password"}`, http.StatusInternalServerError)
		return
	}

	if err := h.db.DeleteUserSessions(userID); err != nil {
		http.Error(w, `{"error":"failed to revoke sessions"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"password updated successfully, please login again"}`))
}
