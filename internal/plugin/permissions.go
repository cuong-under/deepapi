package plugin

import "fmt"

// Permission represents a plugin permission
type Permission string

const (
	PermReadConfig       Permission = "read:config"
	PermWriteConfig      Permission = "write:config"
	PermReadChatHistory  Permission = "read:chat_history"
	PermWriteChatHistory Permission = "write:chat_history"
	PermNetworkAccess    Permission = "network:access"
	PermFileSystem       Permission = "filesystem:access"
)

// CheckPermission validates if plugin has required permission
func (m *Manager) CheckPermission(pluginName string, perm Permission) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	manifest, exists := m.manifests[pluginName]
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	for _, p := range manifest.Permissions {
		if p == string(perm) {
			return nil
		}
	}

	return fmt.Errorf("plugin %s does not have permission %s", pluginName, perm)
}

// HasPermission checks if a plugin has a specific permission
func (m *Manager) HasPermission(pluginName string, perm Permission) bool {
	return m.CheckPermission(pluginName, perm) == nil
}

// GrantPermission adds a permission to a plugin's manifest
func (m *Manager) GrantPermission(pluginName string, perm Permission) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	manifest, exists := m.manifests[pluginName]
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	// Check if already has permission
	for _, p := range manifest.Permissions {
		if p == string(perm) {
			return nil // Already has permission
		}
	}

	// Add permission
	manifest.Permissions = append(manifest.Permissions, string(perm))
	m.manifests[pluginName] = manifest

	return nil
}

// RevokePermission removes a permission from a plugin's manifest
func (m *Manager) RevokePermission(pluginName string, perm Permission) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	manifest, exists := m.manifests[pluginName]
	if !exists {
		return fmt.Errorf("plugin %s not found", pluginName)
	}

	// Remove permission
	newPerms := make([]string, 0, len(manifest.Permissions))
	for _, p := range manifest.Permissions {
		if p != string(perm) {
			newPerms = append(newPerms, p)
		}
	}

	manifest.Permissions = newPerms
	m.manifests[pluginName] = manifest

	return nil
}
