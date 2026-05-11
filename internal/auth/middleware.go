package auth

import (
	"log"
	"net/http"
	"strings"

	"ds2api/internal/database"
)

// Middleware handles authentication
type Middleware struct {
	jwtManager *JWTManager
	db         *database.DB
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(jwtManager *JWTManager, db *database.DB) *Middleware {
	return &Middleware{
		jwtManager: jwtManager,
		db:         db,
	}
}

// Authenticate is a middleware that validates JWT token
func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"error":"invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]

		// Validate JWT token
		claims, err := m.jwtManager.ValidateToken(token)
		if err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Verify session exists in database
		_, err = m.db.GetSessionByToken(token)
		if err != nil {
			http.Error(w, `{"error":"session not found or expired"}`, http.StatusUnauthorized)
			return
		}

		// Set user context
		ctx := SetUserContext(r.Context(), claims.UserID, claims.Username, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin is a middleware that requires admin role
func (m *Middleware) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAdmin(r.Context()) {
			http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireUser is a middleware that requires user or admin role
func (m *Middleware) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userCtx, ok := GetUserContext(r.Context())
		if !ok {
			http.Error(w, `{"error":"user context not found"}`, http.StatusUnauthorized)
			return
		}

		if userCtx.Role != "user" && userCtx.Role != "admin" {
			http.Error(w, `{"error":"user access required"}`, http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// OptionalAuth is a middleware that extracts user info if token is present, but doesn't require it
func (m *Middleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")

		// Debug log
		log.Printf("[OptionalAuth] called for %s, has auth header: %v", r.URL.Path, authHeader != "")

		if authHeader == "" {
			log.Printf("[OptionalAuth] no auth header, passing through")
			next.ServeHTTP(w, r)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			log.Printf("[OptionalAuth] invalid auth header format, passing through")
			next.ServeHTTP(w, r)
			return
		}

		token := parts[1]
		tokenPreview := token
		if len(token) > 20 {
			tokenPreview = token[:20] + "..."
		}
		log.Printf("[OptionalAuth] validating token: %s", tokenPreview)

		claims, err := m.jwtManager.ValidateToken(token)
		if err != nil {
			if looksLikeJWT(token) {
				log.Printf("[OptionalAuth] JWT validation failed: %v", err)
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
				return
			}
			log.Printf("[OptionalAuth] token is not JWT, passing through: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		// Verify session exists
		_, err = m.db.GetSessionByToken(token)
		if err != nil {
			log.Printf("[OptionalAuth] session not found: %v", err)
			http.Error(w, `{"error":"session not found or expired"}`, http.StatusUnauthorized)
			return
		}

		log.Printf("[OptionalAuth] token valid, setting user context: user_id=%d", claims.UserID)

		// Set user context if valid
		ctx := SetUserContext(r.Context(), claims.UserID, claims.Username, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func looksLikeJWT(token string) bool {
	parts := strings.Split(token, ".")
	return len(parts) == 3 && parts[0] != "" && parts[1] != "" && parts[2] != ""
}
