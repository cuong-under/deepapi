package example

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"ds2api/internal/config"
	"ds2api/internal/plugin"
)

func init() {
	// Register plugin at compile time
	plugin.Register("example", NewExamplePlugin)
}

// ExamplePlugin implements plugin.Plugin interface
type ExamplePlugin struct {
	deps *plugin.Dependencies
}

// NewExamplePlugin creates a new example plugin instance
func NewExamplePlugin() plugin.Plugin {
	return &ExamplePlugin{}
}

func (p *ExamplePlugin) Name() string {
	return "example"
}

func (p *ExamplePlugin) Version() string {
	return "1.0.0"
}

func (p *ExamplePlugin) Description() string {
	return "Example plugin for testing"
}

func (p *ExamplePlugin) Author() string {
	return "cuongunder"
}

func (p *ExamplePlugin) Initialize(ctx context.Context, deps *plugin.Dependencies) error {
	p.deps = deps
	config.Logger.Info("[plugin:example] initialized", "version", p.Version())
	return nil
}

func (p *ExamplePlugin) RegisterRoutes(r chi.Router) error {
	r.Get("/example/hello", p.handleHello)
	r.Get("/example/info", p.handleInfo)

	config.Logger.Info("[plugin:example] routes registered")
	return nil
}

func (p *ExamplePlugin) Shutdown(ctx context.Context) error {
	config.Logger.Info("[plugin:example] shutting down")
	return nil
}

func (p *ExamplePlugin) HealthCheck(ctx context.Context) error {
	return nil
}

// handleHello returns a simple hello message
func (p *ExamplePlugin) handleHello(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Hello from example plugin!",
		"version": p.Version(),
	})
}

// handleInfo returns plugin info
func (p *ExamplePlugin) handleInfo(w http.ResponseWriter, r *http.Request) {
	cfg := p.deps.Store.Snapshot()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"plugin": map[string]string{
			"name":        p.Name(),
			"version":     p.Version(),
			"description": p.Description(),
			"author":      p.Author(),
		},
		"config_loaded": cfg.Keys != nil,
		"accounts_count": len(cfg.Accounts),
	})
}
