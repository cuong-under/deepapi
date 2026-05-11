package profile

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"ds2api/internal/auth"
	"ds2api/internal/config"
	"ds2api/internal/database"
	"ds2api/internal/promptcompat"
)

// Handler handles user profile operations
type Handler struct {
	db    *database.DB
	store *config.Store
}

// NewHandler creates a new profile handler
func NewHandler(db *database.DB, stores ...*config.Store) *Handler {
	var store *config.Store
	if len(stores) > 0 {
		store = stores[0]
	}
	return &Handler{
		db:    db,
		store: store,
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
	if len(req.NewPassword) < 8 {
		http.Error(w, `{"error":"password must be at least 8 characters"}`, http.StatusBadRequest)
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

// GetSettings returns non-sensitive effective system settings for regular users.
func (h *Handler) GetSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := auth.GetUserID(r.Context()); !ok {
		http.Error(w, `{"error":"user not authenticated"}`, http.StatusUnauthorized)
		return
	}
	if h.store == nil {
		http.Error(w, `{"error":"settings unavailable"}`, http.StatusServiceUnavailable)
		return
	}

	snap := h.store.Snapshot()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"read_only": true,
		"responses": map[string]any{
			"store_ttl_seconds": h.store.ResponsesStoreTTLSeconds(),
		},
		"embeddings": map[string]any{
			"provider": h.store.EmbeddingsProvider(),
		},
		"auto_delete": map[string]any{
			"mode": h.store.AutoDeleteMode(),
		},
		"current_input_file": map[string]any{
			"enabled":   h.store.CurrentInputFileEnabled(),
			"min_chars": h.store.CurrentInputFileMinChars(),
		},
		"thinking_injection": map[string]any{
			"enabled":        h.store.ThinkingInjectionEnabled(),
			"prompt":         h.store.ThinkingInjectionPrompt(),
			"default_prompt": promptcompat.DefaultThinkingInjectionPrompt,
		},
		"model_aliases": snap.ModelAliases,
	})
}
