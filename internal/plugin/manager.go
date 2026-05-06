package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/go-chi/chi/v5"

	"ds2api/internal/config"
)

// Manager quản lý lifecycle của plugins
type Manager struct {
	plugins    map[string]Plugin
	manifests  map[string]Manifest
	pluginsDir string
	deps       *Dependencies
	mu         sync.RWMutex
}

// NewManager tạo plugin manager mới
func NewManager(pluginsDir string, deps *Dependencies) *Manager {
	return &Manager{
		plugins:    make(map[string]Plugin),
		manifests:  make(map[string]Manifest),
		pluginsDir: pluginsDir,
		deps:       deps,
	}
}

// DiscoverPlugins scans plugins directory và load manifests
func (m *Manager) DiscoverPlugins(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if plugins directory exists
	if _, err := os.Stat(m.pluginsDir); os.IsNotExist(err) {
		config.Logger.Info("[plugin] plugins directory does not exist, creating", "path", m.pluginsDir)
		if err := os.MkdirAll(m.pluginsDir, 0755); err != nil {
			return fmt.Errorf("create plugins directory: %w", err)
		}
		return nil
	}

	// Scan for plugin directories
	entries, err := os.ReadDir(m.pluginsDir)
	if err != nil {
		return fmt.Errorf("read plugins directory: %w", err)
	}

	config.Logger.Info("[plugin] discovering plugins", "dir", m.pluginsDir, "count", len(entries))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginName := entry.Name()
		manifestPath := filepath.Join(m.pluginsDir, pluginName, "plugin.json")

		// Check if plugin.json exists
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			config.Logger.Warn("[plugin] skipping directory without plugin.json", "name", pluginName)
			continue
		}

		// Load manifest
		manifest, err := loadManifest(manifestPath)
		if err != nil {
			config.Logger.Warn("[plugin] failed to load manifest", "name", pluginName, "error", err)
			continue
		}

		m.manifests[pluginName] = manifest
		config.Logger.Info("[plugin] discovered plugin", "name", manifest.Name, "version", manifest.Version)
	}

	return nil
}

// LoadPlugin loads a single plugin
func (m *Manager) LoadPlugin(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already loaded
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already loaded", name)
	}

	// Get manifest
	manifest, exists := m.manifests[name]
	if !exists {
		return fmt.Errorf("plugin %s not found", name)
	}

	// Check if enabled
	if !manifest.Enabled {
		config.Logger.Info("[plugin] plugin is disabled, skipping", "name", name)
		return nil
	}

	// Load plugin .so file
	pluginPath := filepath.Join(m.pluginsDir, name, manifest.EntryPoint)
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return fmt.Errorf("open plugin %s: %w", name, err)
	}

	// Lookup NewPlugin symbol
	symNewPlugin, err := p.Lookup("NewPlugin")
	if err != nil {
		return fmt.Errorf("lookup NewPlugin in %s: %w", name, err)
	}

	// Cast to function
	newPlugin, ok := symNewPlugin.(func() Plugin)
	if !ok {
		return fmt.Errorf("invalid NewPlugin signature in %s", name)
	}

	// Create plugin instance
	pluginInstance := newPlugin()

	// Initialize plugin
	if err := pluginInstance.Initialize(ctx, m.deps); err != nil {
		return fmt.Errorf("initialize plugin %s: %w", name, err)
	}

	// Store plugin
	m.plugins[name] = pluginInstance
	config.Logger.Info("[plugin] loaded plugin", "name", name, "version", manifest.Version)

	return nil
}

// LoadAllPlugins loads all discovered plugins
func (m *Manager) LoadAllPlugins(ctx context.Context) error {
	m.mu.RLock()
	names := make([]string, 0, len(m.manifests))
	for name := range m.manifests {
		names = append(names, name)
	}
	m.mu.RUnlock()

	for _, name := range names {
		if err := m.LoadPlugin(ctx, name); err != nil {
			config.Logger.Warn("[plugin] failed to load plugin", "name", name, "error", err)
			// Continue loading other plugins
		}
	}

	return nil
}

// UnloadPlugin unloads a plugin
func (m *Manager) UnloadPlugin(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pluginInstance, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not loaded", name)
	}

	// Call Shutdown
	if err := pluginInstance.Shutdown(ctx); err != nil {
		config.Logger.Warn("[plugin] error during shutdown", "name", name, "error", err)
	}

	// Remove from map
	delete(m.plugins, name)
	config.Logger.Info("[plugin] unloaded plugin", "name", name)

	// Note: Go plugins cannot be truly unloaded from memory
	return nil
}

// RegisterAllRoutes registers routes from all loaded plugins
func (m *Manager) RegisterAllRoutes(r chi.Router) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, pluginInstance := range m.plugins {
		if err := pluginInstance.RegisterRoutes(r); err != nil {
			return fmt.Errorf("register routes for plugin %s: %w", name, err)
		}
		config.Logger.Info("[plugin] registered routes", "name", name)
	}

	return nil
}

// ListPlugins returns all discovered plugins
func (m *Manager) ListPlugins() []Manifest {
	m.mu.RLock()
	defer m.mu.RUnlock()

	manifests := make([]Manifest, 0, len(m.manifests))
	for _, manifest := range m.manifests {
		manifests = append(manifests, manifest)
	}

	return manifests
}

// GetPlugin returns a loaded plugin by name
func (m *Manager) GetPlugin(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, exists := m.plugins[name]
	return p, exists
}

// GetManifest returns a plugin manifest by name
func (m *Manager) GetManifest(name string) (Manifest, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	manifest, exists := m.manifests[name]
	return manifest, exists
}

// loadManifest loads plugin manifest from file
func loadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse manifest: %w", err)
	}

	// Validate required fields
	if manifest.Name == "" {
		return Manifest{}, fmt.Errorf("manifest missing name")
	}
	if manifest.Version == "" {
		return Manifest{}, fmt.Errorf("manifest missing version")
	}
	if manifest.EntryPoint == "" {
		return Manifest{}, fmt.Errorf("manifest missing entry_point")
	}

	return manifest, nil
}
