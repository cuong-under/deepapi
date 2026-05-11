package keys

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"ds2api/internal/auth"
	"ds2api/internal/database"
)

// Handler handles user API key operations
type Handler struct {
	db *database.DB
}

// NewHandler creates a new keys handler
func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

// CreateKeyRequest represents create API key request
type CreateKeyRequest struct {
	Name   string `json:"name"`
	Remark string `json:"remark"`
}

// UpdateKeyRequest represents update API key request
type UpdateKeyRequest struct {
	Name   string `json:"name"`
	Remark string `json:"remark"`
}

type KeyResponse struct {
	ID            int64  `json:"id"`
	UserID        int64  `json:"user_id"`
	APIKeyPreview string `json:"api_key_preview"`
	Name          string `json:"name"`
	Remark        string `json:"remark"`
	CreatedAt     int64  `json:"created_at"`
}

type CreatedKeyResponse struct {
	KeyResponse
	APIKey string `json:"api_key"`
}

func keyPreview(apiKey string) string {
	if len(apiKey) <= 12 {
		return apiKey
	}
	return apiKey[:6] + "..." + apiKey[len(apiKey)-4:]
}

func keyResponse(key *database.UserAPIKey) KeyResponse {
	return KeyResponse{
		ID:            key.ID,
		UserID:        key.UserID,
		APIKeyPreview: keyPreview(key.APIKey),
		Name:          key.Name,
		Remark:        key.Remark,
		CreatedAt:     key.CreatedAt,
	}
}

func keyResponses(keys []*database.UserAPIKey) []KeyResponse {
	responses := make([]KeyResponse, 0, len(keys))
	for _, key := range keys {
		responses = append(responses, keyResponse(key))
	}
	return responses
}

// generateAPIKey generates a random API key
func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "sk-" + hex.EncodeToString(b), nil
}

// ListKeys lists all API keys for current user
func (h *Handler) ListKeys(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error":"user not authenticated"}`, http.StatusUnauthorized)
		return
	}

	keys, err := h.db.GetAPIKeysByUserID(userID)
	if err != nil {
		http.Error(w, `{"error":"failed to get API keys"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"keys":  keyResponses(keys),
		"total": len(keys),
	})
}

// CreateKey creates a new API key
func (h *Handler) CreateKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error":"user not authenticated"}`, http.StatusUnauthorized)
		return
	}

	var req CreateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Generate API key
	apiKey, err := generateAPIKey()
	if err != nil {
		http.Error(w, `{"error":"failed to generate API key"}`, http.StatusInternalServerError)
		return
	}

	key, err := h.db.CreateAPIKey(userID, apiKey, req.Name, req.Remark)
	if err != nil {
		http.Error(w, `{"error":"failed to create API key"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreatedKeyResponse{
		KeyResponse: keyResponse(key),
		APIKey:      key.APIKey,
	})
}

// UpdateKey updates an existing API key
func (h *Handler) UpdateKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error":"user not authenticated"}`, http.StatusUnauthorized)
		return
	}

	keyIDStr := chi.URLParam(r, "id")
	keyID, err := strconv.ParseInt(keyIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid key id"}`, http.StatusBadRequest)
		return
	}

	var req UpdateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Update with ownership check
	err = h.db.UpdateAPIKey(keyID, userID, req.Name, req.Remark)
	if err != nil {
		if err.Error() == "api key not found or access denied" {
			http.Error(w, `{"error":"API key not found or access denied"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error":"failed to update API key"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"API key updated successfully"}`))
}

// DeleteKey deletes an API key
func (h *Handler) DeleteKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.GetUserID(r.Context())
	if !ok {
		http.Error(w, `{"error":"user not authenticated"}`, http.StatusUnauthorized)
		return
	}

	keyIDStr := chi.URLParam(r, "id")
	keyID, err := strconv.ParseInt(keyIDStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid key id"}`, http.StatusBadRequest)
		return
	}

	// Delete with ownership check
	err = h.db.DeleteAPIKey(keyID, userID)
	if err != nil {
		if err.Error() == "api key not found or access denied" {
			http.Error(w, `{"error":"API key not found or access denied"}`, http.StatusForbidden)
			return
		}
		http.Error(w, `{"error":"failed to delete API key"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"API key deleted successfully"}`))
}
