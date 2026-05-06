package plugin

import (
	"ds2api/internal/chathistory"
	"ds2api/internal/config"
)

// SandboxedStore wraps config.Store với permission checks
type SandboxedStore struct {
	store      *config.Store
	pluginName string
	manager    *Manager
}

// NewSandboxedStore tạo sandboxed store cho plugin
func NewSandboxedStore(store *config.Store, pluginName string, manager *Manager) *SandboxedStore {
	return &SandboxedStore{
		store:      store,
		pluginName: pluginName,
		manager:    manager,
	}
}

// Snapshot returns config snapshot nếu plugin có permission
func (s *SandboxedStore) Snapshot() config.Config {
	if err := s.manager.CheckPermission(s.pluginName, PermReadConfig); err != nil {
		config.Logger.Warn("[plugin] permission denied", "plugin", s.pluginName, "permission", PermReadConfig)
		return config.Config{}
	}
	return s.store.Snapshot()
}

// Save saves config nếu plugin có permission
func (s *SandboxedStore) Save() error {
	if err := s.manager.CheckPermission(s.pluginName, PermWriteConfig); err != nil {
		config.Logger.Warn("[plugin] permission denied", "plugin", s.pluginName, "permission", PermWriteConfig)
		return err
	}
	return s.store.Save()
}

// SandboxedChatHistory wraps chathistory.Store với permission checks
type SandboxedChatHistory struct {
	store      *chathistory.Store
	pluginName string
	manager    *Manager
}

// NewSandboxedChatHistory tạo sandboxed chat history cho plugin
func NewSandboxedChatHistory(store *chathistory.Store, pluginName string, manager *Manager) *SandboxedChatHistory {
	return &SandboxedChatHistory{
		store:      store,
		pluginName: pluginName,
		manager:    manager,
	}
}

// Snapshot returns chat history snapshot nếu plugin có permission
func (s *SandboxedChatHistory) Snapshot() (chathistory.File, error) {
	if err := s.manager.CheckPermission(s.pluginName, PermReadChatHistory); err != nil {
		config.Logger.Warn("[plugin] permission denied", "plugin", s.pluginName, "permission", PermReadChatHistory)
		return chathistory.File{}, err
	}
	return s.store.Snapshot()
}

// Get returns chat history entry nếu plugin có permission
func (s *SandboxedChatHistory) Get(id string) (chathistory.Entry, error) {
	if err := s.manager.CheckPermission(s.pluginName, PermReadChatHistory); err != nil {
		config.Logger.Warn("[plugin] permission denied", "plugin", s.pluginName, "permission", PermReadChatHistory)
		return chathistory.Entry{}, err
	}
	return s.store.Get(id)
}

// Delete deletes chat history entry nếu plugin có permission
func (s *SandboxedChatHistory) Delete(id string) error {
	if err := s.manager.CheckPermission(s.pluginName, PermWriteChatHistory); err != nil {
		config.Logger.Warn("[plugin] permission denied", "plugin", s.pluginName, "permission", PermWriteChatHistory)
		return err
	}
	return s.store.Delete(id)
}

// Path returns store path
func (s *SandboxedChatHistory) Path() string {
	return s.store.Path()
}

// Err returns store error
func (s *SandboxedChatHistory) Err() error {
	return s.store.Err()
}
