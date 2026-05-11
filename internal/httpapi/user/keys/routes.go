package keys

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers user API key routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/keys", h.ListKeys)
	r.Post("/keys", h.CreateKey)
	r.Put("/keys/{id}", h.UpdateKey)
	r.Delete("/keys/{id}", h.DeleteKey)
}
