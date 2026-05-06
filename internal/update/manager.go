package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ds2api/internal/config"
)

// Manager quản lý update process
type Manager struct {
	currentVersion string
	BackupDir      string
	pluginsDir     string
	githubRepo     string // "CJackHwang/ds2api"
	dataDir        string
}

// Release represents a GitHub release
type Release struct {
	Version     string    `json:"version"`
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
	HTMLURL     string    `json:"html_url"`
}

// Asset represents a release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

// UpdateStatus represents current update status
type UpdateStatus struct {
	Stage       string    `json:"stage"`        // checking, downloading, backing_up, installing, verifying, complete, failed
	Progress    int       `json:"progress"`     // 0-100
	Message     string    `json:"message"`
	Error       string    `json:"error,omitempty"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// NewManager creates a new update manager
func NewManager(currentVersion, githubRepo string) *Manager {
	// Get data directory from config or use default
	dataDir := filepath.Join(filepath.Dir(config.ConfigPath()), "data")
	backupDir := filepath.Join(dataDir, "backups")
	pluginsDir := filepath.Join(filepath.Dir(config.ConfigPath()), "plugins")

	return &Manager{
		currentVersion: currentVersion,
		BackupDir:      backupDir,
		pluginsDir:     pluginsDir,
		githubRepo:     githubRepo,
		dataDir:        dataDir,
	}
}

// GetCurrentVersion returns current version
func (m *Manager) GetCurrentVersion() string {
	return m.currentVersion
}

// EnsureDirectories ensures required directories exist
func (m *Manager) EnsureDirectories() error {
	dirs := []string{m.BackupDir, m.pluginsDir, m.dataDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}
	return nil
}

// CheckUpdate checks if a new version is available
func (m *Manager) CheckUpdate(ctx context.Context) (*Release, error) {
	config.Logger.Info("[update] checking for updates", "current_version", m.currentVersion)

	// Get latest release from GitHub
	release, err := m.getLatestRelease(ctx)
	if err != nil {
		return nil, fmt.Errorf("get latest release: %w", err)
	}

	// Compare versions
	if release.TagName == m.currentVersion {
		config.Logger.Info("[update] already up to date", "version", m.currentVersion)
		return nil, nil
	}

	config.Logger.Info("[update] new version available",
		"current", m.currentVersion,
		"latest", release.TagName)

	return release, nil
}

// DownloadUpdate downloads release archive
func (m *Manager) DownloadUpdate(ctx context.Context, release *Release) (string, error) {
	config.Logger.Info("[update] downloading update", "version", release.TagName)

	// Find appropriate asset for current platform
	asset := m.findAssetForPlatform(release.Assets)
	if asset == nil {
		return "", fmt.Errorf("no suitable asset found for current platform")
	}

	// Download to temp directory
	downloadPath := filepath.Join(m.dataDir, "downloads", asset.Name)
	if err := os.MkdirAll(filepath.Dir(downloadPath), 0755); err != nil {
		return "", fmt.Errorf("create download directory: %w", err)
	}

	if err := m.downloadFile(ctx, asset.BrowserDownloadURL, downloadPath); err != nil {
		return "", fmt.Errorf("download file: %w", err)
	}

	config.Logger.Info("[update] download complete", "path", downloadPath, "size", asset.Size)
	return downloadPath, nil
}

// BackupCurrent creates backup of current installation
func (m *Manager) BackupCurrent(ctx context.Context) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(m.BackupDir, fmt.Sprintf("backup-%s", timestamp))

	config.Logger.Info("[update] creating backup", "path", backupPath)

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return "", fmt.Errorf("create backup directory: %w", err)
	}

	// Backup binary
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("get executable path: %w", err)
	}

	if err := m.copyFile(exePath, filepath.Join(backupPath, "ds2api.exe")); err != nil {
		return "", fmt.Errorf("backup binary: %w", err)
	}

	// Backup config
	configPath := config.ConfigPath()
	if configPath != "" {
		if err := m.copyFile(configPath, filepath.Join(backupPath, "config.json")); err != nil {
			config.Logger.Warn("[update] failed to backup config", "error", err)
		}
	}

	// Note: Plugins directory is NOT backed up - it's preserved during update

	config.Logger.Info("[update] backup complete", "path", backupPath)
	return backupPath, nil
}

// InstallUpdate installs downloaded update
func (m *Manager) InstallUpdate(ctx context.Context, archivePath string) error {
	config.Logger.Info("[update] installing update", "archive", archivePath)

	// Extract archive to temp directory
	tempDir := filepath.Join(m.dataDir, "temp-update")
	if err := os.RemoveAll(tempDir); err != nil {
		return fmt.Errorf("clean temp directory: %w", err)
	}

	if err := m.extractArchive(archivePath, tempDir); err != nil {
		return fmt.Errorf("extract archive: %w", err)
	}

	// Find new binary in extracted files
	newBinaryPath := filepath.Join(tempDir, "ds2api.exe")
	if _, err := os.Stat(newBinaryPath); os.IsNotExist(err) {
		return fmt.Errorf("new binary not found in archive")
	}

	// Replace current binary (atomic operation)
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	// On Windows, we need to rename old binary first
	oldBinaryPath := exePath + ".old"
	if err := os.Rename(exePath, oldBinaryPath); err != nil {
		return fmt.Errorf("rename old binary: %w", err)
	}

	// Copy new binary
	if err := m.copyFile(newBinaryPath, exePath); err != nil {
		// Rollback
		os.Rename(oldBinaryPath, exePath)
		return fmt.Errorf("copy new binary: %w", err)
	}

	// Remove old binary
	os.Remove(oldBinaryPath)

	// Cleanup temp directory
	os.RemoveAll(tempDir)

	config.Logger.Info("[update] installation complete")
	return nil
}

// Rollback restores from backup
func (m *Manager) Rollback(ctx context.Context, backupPath string) error {
	config.Logger.Info("[update] rolling back", "backup", backupPath)

	// Restore binary
	backupBinary := filepath.Join(backupPath, "ds2api.exe")
	if _, err := os.Stat(backupBinary); os.IsNotExist(err) {
		return fmt.Errorf("backup binary not found")
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	if err := m.copyFile(backupBinary, exePath); err != nil {
		return fmt.Errorf("restore binary: %w", err)
	}

	// Restore config if exists
	backupConfig := filepath.Join(backupPath, "config.json")
	if _, err := os.Stat(backupConfig); err == nil {
		configPath := config.ConfigPath()
		if configPath != "" {
			if err := m.copyFile(backupConfig, configPath); err != nil {
				config.Logger.Warn("[update] failed to restore config", "error", err)
			}
		}
	}

	config.Logger.Info("[update] rollback complete")
	return nil
}

// VerifyHealth checks if update was successful
func (m *Manager) VerifyHealth(ctx context.Context) error {
	config.Logger.Info("[update] verifying health")

	// Check if binary exists and is executable
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found")
	}

	// Check if config is valid
	if _, err := config.LoadStoreWithError(); err != nil {
		return fmt.Errorf("config invalid: %w", err)
	}

	// Check if plugins directory exists
	if _, err := os.Stat(m.pluginsDir); os.IsNotExist(err) {
		config.Logger.Warn("[update] plugins directory not found", "path", m.pluginsDir)
	}

	config.Logger.Info("[update] health check passed")
	return nil
}

// ListBackups returns list of available backups
func (m *Manager) ListBackups() ([]string, error) {
	entries, err := os.ReadDir(m.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read backup directory: %w", err)
	}

	backups := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			backups = append(backups, entry.Name())
		}
	}

	return backups, nil
}

// findAssetForPlatform finds appropriate asset for current platform
func (m *Manager) findAssetForPlatform(assets []Asset) *Asset {
	// For now, just return first asset
	// TODO: Implement platform-specific logic
	if len(assets) > 0 {
		return &assets[0]
	}
	return nil
}
