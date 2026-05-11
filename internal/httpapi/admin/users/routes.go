package users

import (
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler) {
	r.Get("/users", h.ListUsers)
	r.Post("/users", h.CreateUser)
	r.Put("/users/{id}", h.UpdateUser)
	r.Put("/users/{id}/password", h.ChangePassword)
	r.Delete("/users/{id}", h.DeleteUser)
}
