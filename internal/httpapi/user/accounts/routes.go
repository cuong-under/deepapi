package accounts

import (
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes registers user account routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/accounts", h.ListAccounts)
	r.Post("/accounts", h.CreateAccount)
	r.Get("/accounts/{id}", h.GetAccount)
	r.Put("/accounts/{id}", h.UpdateAccount)
	r.Delete("/accounts/{id}", h.DeleteAccount)
	r.Post("/accounts/{id}/refresh-token", h.RefreshToken)
}
