package update

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers update routes
func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/update/check", h.CheckUpdate)
	r.Post("/update/install", h.InstallUpdate)
	r.Get("/update/status", h.GetStatus)
	r.Get("/update/backups", h.ListBackups)
	r.Post("/update/rollback", h.Rollback)
}
