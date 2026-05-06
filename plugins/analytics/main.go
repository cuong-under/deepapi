package analytics

import (
	"context"

	"github.com/go-chi/chi/v5"

	"ds2api/internal/config"
	"ds2api/internal/httpapi/admin/analytics"
	"ds2api/internal/plugin"
)

func init() {
	// Register plugin at compile time
	plugin.Register("analytics", NewAnalyticsPlugin)
}

// AnalyticsPlugin implements plugin.Plugin interface
type AnalyticsPlugin struct {
	handler *analytics.Handler
	deps    *plugin.Dependencies
}

// NewAnalyticsPlugin creates a new analytics plugin instance
func NewAnalyticsPlugin() plugin.Plugin {
	return &AnalyticsPlugin{}
}

func (p *AnalyticsPlugin) Name() string {
	return "analytics"
}

func (p *AnalyticsPlugin) Version() string {
	return "1.0.0"
}

func (p *AnalyticsPlugin) Description() string {
	return "Token usage analytics dashboard"
}

func (p *AnalyticsPlugin) Author() string {
	return "cuongunder"
}

func (p *AnalyticsPlugin) Initialize(ctx context.Context, deps *plugin.Dependencies) error {
	p.deps = deps
	p.handler = &analytics.Handler{
		ChatHistory: deps.ChatHistory,
		Store:       deps.Store,
	}

	// Load pricing from config
	cfg := deps.Store.Snapshot()
	p.handler.SetPricing(cfg.Pricing)

	config.Logger.Info("[plugin:analytics] initialized", "version", p.Version())
	return nil
}

func (p *AnalyticsPlugin) RegisterRoutes(r chi.Router) error {
	r.Get("/analytics/token-usage", p.handler.GetTokenUsage)
	r.Get("/analytics/overview", p.handler.GetOverview)

	config.Logger.Info("[plugin:analytics] routes registered")
	return nil
}

func (p *AnalyticsPlugin) Shutdown(ctx context.Context) error {
	config.Logger.Info("[plugin:analytics] shutting down")
	return nil
}

func (p *AnalyticsPlugin) HealthCheck(ctx context.Context) error {
	// Simple health check - verify handler is initialized
	if p.handler == nil {
		return nil
	}
	return nil
}
