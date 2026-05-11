package profile

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers profile routes
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Post("/profile/change-password", h.ChangePassword)
}
