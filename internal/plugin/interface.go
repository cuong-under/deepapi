package plugin

import (
	"context"

	"github.com/go-chi/chi/v5"

	"ds2api/internal/chathistory"
	"ds2api/internal/config"
)

// Plugin là interface mà tất cả plugins phải implement
type Plugin interface {
	// Metadata
	Name() string
	Version() string
	Description() string
	Author() string

	// Lifecycle
	Initialize(ctx context.Context, deps *Dependencies) error
	Shutdown(ctx context.Context) error

	// Routes registration
	RegisterRoutes(r chi.Router) error

	// Health check
	HealthCheck(ctx context.Context) error
}

// Dependencies được inject vào plugin
type Dependencies struct {
	Store       *config.Store
	ChatHistory *chathistory.Store
	// Thêm dependencies khác nếu cần
}

// Manifest chứa metadata của plugin
type Manifest struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Author       string   `json:"author"`
	EntryPoint   string   `json:"entry_point"`   // Path to .so file
	WebUIPath    string   `json:"webui_path"`    // Path to frontend bundle
	Permissions  []string `json:"permissions"`   // Required permissions
	Dependencies []string `json:"dependencies"`  // Other plugins this depends on
	Enabled      bool     `json:"enabled"`       // Whether plugin is enabled
}
