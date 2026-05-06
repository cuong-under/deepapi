package analytics

import (
	"github.com/go-chi/chi/v5"
)

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/analytics/token-usage", h.GetTokenUsage)
	r.Get("/analytics/overview", h.GetOverview)
}
