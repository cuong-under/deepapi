package auth

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"ds2api/internal/auth"
	"ds2api/internal/config"
	"ds2api/internal/database"
	"ds2api/internal/errors"
	"ds2api/internal/validation"
)

// Handler handles authentication endpoints
type Handler struct {
	db         *database.DB
	jwtManager *auth.JWTManager
}

// NewHandler creates a new auth handler
func NewHandler(db *database.DB, jwtManager *auth.JWTManager) *Handler {
	return &Handler{
		db:         db,
		jwtManager: jwtManager,
	}
}

// RegisterRequest represents registration request
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresIn int64     `json:"expires_in"`
	User      *UserInfo `json:"user"`
}

// UserInfo represents user information
type UserInfo struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

// Register handles user registration
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if os.Getenv("DS2API_ALLOW_REGISTRATION") != "true" {
		count, err := h.db.CountUsers()
		if err != nil {
			errors.RespondError(w, errors.ErrDatabaseError.WithDetails("failed to count users"), config.Logger)
			return
		}
		if count > 0 {
			errors.RespondError(w, errors.ErrUnauthorized.WithDetails("registration is disabled"), config.Logger)
			return
		}
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.RespondError(w, errors.ErrInvalidInput.WithDetails("invalid JSON format"), config.Logger)
		return
	}

	// Validate input
	validator := validation.UserValidator{}

	if err := validator.ValidateUsername(req.Username); err != nil {
		errors.RespondError(w, errors.ErrInvalidUsername.WithDetails(err.Error()), config.Logger)
		return
	}

	if err := validator.ValidateEmail(req.Email); err != nil {
		errors.RespondError(w, errors.ErrInvalidEmail.WithDetails(err.Error()), config.Logger)
		return
	}

	// Use simple password validation for backward compatibility
	if err := validator.ValidatePasswordSimple(req.Password); err != nil {
		errors.RespondError(w, errors.ErrWeakPassword.WithDetails(err.Error()), config.Logger)
		return
	}

	// Check if username already exists
	_, err := h.db.GetUserByUsername(req.Username)
	if err == nil {
		errors.RespondError(w, errors.ErrDuplicateUsername, config.Logger)
		return
	}

	// Check if email already exists
	_, err = h.db.GetUserByEmail(req.Email)
	if err == nil {
		errors.RespondError(w, errors.ErrDuplicateEmail, config.Logger)
		return
	}

	// Create user (default role: user)
	user, err := h.db.CreateUser(req.Username, req.Email, req.Password, "user")
	if err != nil {
		errors.RespondError(w, errors.ErrDatabaseError.WithDetails("failed to create user"), config.Logger)
		return
	}

	// Generate JWT token
	expiresIn := 7 * 24 * time.Hour // 7 days
	token, err := h.jwtManager.GenerateToken(user.ID, user.Username, user.Role, expiresIn)
	if err != nil {
		errors.RespondError(w, errors.ErrInternalServer.WithDetails("failed to generate token"), config.Logger)
		return
	}

	// Create session in database with JWT token
	_, err = h.db.CreateSession(user.ID, token, expiresIn)
	if err != nil {
		errors.RespondError(w, errors.ErrDatabaseError.WithDetails("failed to create session"), config.Logger)
		return
	}

	// Return response
	resp := AuthResponse{
		Token:     token,
		ExpiresIn: int64(expiresIn.Seconds()),
		User: &UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
		},
	}

	errors.RespondSuccess(w, resp)
}

// Login handles user login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errors.RespondError(w, errors.ErrInvalidInput.WithDetails("invalid JSON format"), config.Logger)
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		errors.RespondError(w, errors.ErrInvalidInput.WithDetails("username and password are required"), config.Logger)
		return
	}

	// Get user by username
	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		errors.RespondError(w, errors.ErrInvalidCredentials, config.Logger)
		return
	}

	// Verify password
	if !h.db.VerifyPassword(user, req.Password) {
		errors.RespondError(w, errors.ErrInvalidCredentials, config.Logger)
		return
	}

	// Generate JWT token
	expiresIn := 7 * 24 * time.Hour // 7 days
	token, err := h.jwtManager.GenerateToken(user.ID, user.Username, user.Role, expiresIn)
	if err != nil {
		errors.RespondError(w, errors.ErrInternalServer.WithDetails("failed to generate token"), config.Logger)
		return
	}

	// Create session in database with JWT token
	_, err = h.db.CreateSession(user.ID, token, expiresIn)
	if err != nil {
		errors.RespondError(w, errors.ErrDatabaseError.WithDetails("failed to create session"), config.Logger)
		return
	}

	// Return response
	resp := AuthResponse{
		Token:     token,
		ExpiresIn: int64(expiresIn.Seconds()),
		User: &UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
		},
	}

	errors.RespondSuccess(w, resp)
}

// Logout handles user logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
		return
	}

	// Extract token
	var token string
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	} else {
		http.Error(w, `{"error":"invalid authorization header"}`, http.StatusUnauthorized)
		return
	}

	// Delete session
	err := h.db.DeleteSessionByToken(token)
	if err != nil {
		http.Error(w, `{"error":"failed to logout"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message":"logged out successfully"}`))
}

// Me returns current user information
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userCtx, ok := auth.GetUserContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"user context not found"}`, http.StatusUnauthorized)
		return
	}

	// Get full user info from database
	user, err := h.db.GetUserByID(userCtx.UserID)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}

	userInfo := UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userInfo)
}

// RefreshToken refreshes the JWT token
func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
		return
	}

	// Extract token
	var token string
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	} else {
		http.Error(w, `{"error":"invalid authorization header"}`, http.StatusUnauthorized)
		return
	}

	// Validate current token
	claims, err := h.jwtManager.ValidateToken(token)
	if err != nil {
		http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
		return
	}

	// Generate new token
	expiresIn := 7 * 24 * time.Hour // 7 days
	newToken, err := h.jwtManager.GenerateToken(claims.UserID, claims.Username, claims.Role, expiresIn)
	if err != nil {
		http.Error(w, `{"error":"failed to generate token"}`, http.StatusInternalServerError)
		return
	}

	// Delete old session and create new one with new JWT token
	h.db.DeleteSessionByToken(token)
	_, err = h.db.CreateSession(claims.UserID, newToken, expiresIn)
	if err != nil {
		http.Error(w, `{"error":"failed to create session"}`, http.StatusInternalServerError)
		return
	}

	// Return new token
	resp := map[string]interface{}{
		"token":      newToken,
		"expires_in": int64(expiresIn.Seconds()),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
