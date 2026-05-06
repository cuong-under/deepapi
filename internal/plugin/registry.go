package plugin

import (
	"context"
	"fmt"
	"sync"

	"ds2api/internal/config"
)

// Registry quản lý các plugins được đăng ký compile-time
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]PluginFactory
}

// PluginFactory là function tạo plugin instance
type PluginFactory func() Plugin

var globalRegistry = &Registry{
	plugins: make(map[string]PluginFactory),
}

// Register đăng ký một plugin factory
func Register(name string, factory PluginFactory) {
	globalRegistry.mu.Lock()
	defer globalRegistry.mu.Unlock()

	if _, exists := globalRegistry.plugins[name]; exists {
		config.Logger.Warn("[plugin] plugin already registered, overwriting", "name", name)
	}

	globalRegistry.plugins[name] = factory
	config.Logger.Info("[plugin] registered plugin factory", "name", name)
}

// GetFactory returns plugin factory by name
func GetFactory(name string) (PluginFactory, bool) {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	factory, exists := globalRegistry.plugins[name]
	return factory, exists
}

// ListRegistered returns all registered plugin names
func ListRegistered() []string {
	globalRegistry.mu.RLock()
	defer globalRegistry.mu.RUnlock()

	names := make([]string, 0, len(globalRegistry.plugins))
	for name := range globalRegistry.plugins {
		names = append(names, name)
	}
	return names
}

// CreatePlugin creates a plugin instance from registered factory
func CreatePlugin(name string) (Plugin, error) {
	factory, exists := GetFactory(name)
	if !exists {
		return nil, fmt.Errorf("plugin %s not registered", name)
	}

	return factory(), nil
}

// LoadPluginFromRegistry loads a plugin from registry instead of .so file
func (m *Manager) LoadPluginFromRegistry(ctx context.Context, name string) error {
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

	// Create plugin from registry
	pluginInstance, err := CreatePlugin(name)
	if err != nil {
		return fmt.Errorf("create plugin %s: %w", name, err)
	}

	// Initialize plugin
	if err := pluginInstance.Initialize(ctx, m.deps); err != nil {
		return fmt.Errorf("initialize plugin %s: %w", name, err)
	}

	// Store plugin
	m.plugins[name] = pluginInstance
	config.Logger.Info("[plugin] loaded plugin from registry", "name", name, "version", manifest.Version)

	return nil
}

// LoadAllPluginsFromRegistry loads all plugins from registry
func (m *Manager) LoadAllPluginsFromRegistry(ctx context.Context) error {
	m.mu.RLock()
	names := make([]string, 0, len(m.manifests))
	for name := range m.manifests {
		names = append(names, name)
	}
	m.mu.RUnlock()

	for _, name := range names {
		if err := m.LoadPluginFromRegistry(ctx, name); err != nil {
			config.Logger.Warn("[plugin] failed to load plugin from registry", "name", name, "error", err)
			// Continue loading other plugins
		}
	}

	return nil
}
