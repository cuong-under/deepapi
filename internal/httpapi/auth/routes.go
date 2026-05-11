package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers authentication routes
func (h *Handler) RegisterRoutes(r chi.Router, authMiddleware interface{}) {
	// Public routes (no authentication)
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)

	// Protected routes (require authentication)
	if authMiddleware != nil {
		if mw, ok := authMiddleware.(func(http.Handler) http.Handler); ok {
			r.Group(func(ar chi.Router) {
				ar.Use(mw)
				ar.Post("/logout", h.Logout)
				ar.Get("/me", h.Me)
				ar.Post("/refresh", h.RefreshToken)
			})
		}
	}
}
