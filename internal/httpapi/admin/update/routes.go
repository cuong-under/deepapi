package update

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// UpdateHandler interface for both Handler and GitHandler
type UpdateHandler interface {
	CheckUpdate(w http.ResponseWriter, r *http.Request)
	InstallUpdate(w http.ResponseWriter, r *http.Request)
	GetStatus(w http.ResponseWriter, r *http.Request)
	ListBackups(w http.ResponseWriter, r *http.Request)
	Rollback(w http.ResponseWriter, r *http.Request)
}

// RegisterRoutes registers update routes
func RegisterRoutes(r chi.Router, h UpdateHandler) {
	if h == nil {
		return
	}
	r.Get("/update/check", h.CheckUpdate)
	r.Post("/update/install", h.InstallUpdate)
	r.Get("/update/status", h.GetStatus)
	r.Get("/update/backups", h.ListBackups)
	r.Post("/update/rollback", h.Rollback)
}
