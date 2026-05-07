package update

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"ds2api/internal/config"
	"ds2api/internal/version"
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

// SetCurrentVersion updates current version
func (m *Manager) SetCurrentVersion(version string) {
	m.currentVersion = version
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
func (m *Manager) InstallUpdate(ctx context.Context, archivePath string, newVersion string) error {
	config.Logger.Info("[update] installing update", "archive", archivePath, "new_version", newVersion)

	// Extract archive to temp directory
	tempDir := filepath.Join(m.dataDir, "temp-update")
	if err := os.RemoveAll(tempDir); err != nil {
		return fmt.Errorf("clean temp directory: %w", err)
	}

	if err := m.extractArchive(archivePath, tempDir); err != nil {
		return fmt.Errorf("extract archive: %w", err)
	}

	// Find new binary in extracted files
	// Archive may contain a subdirectory, so we need to search for the binary
	var newBinaryPath string
	err := filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (info.Name() == "ds2api.exe" || info.Name() == "ds2api") {
			newBinaryPath = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("search for binary: %w", err)
	}
	if newBinaryPath == "" {
		return fmt.Errorf("new binary not found in archive")
	}

	config.Logger.Info("[update] found binary", "path", newBinaryPath)

	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	// Resolve symlinks
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		config.Logger.Warn("[update] failed to resolve symlinks", "error", err)
	}

	config.Logger.Info("[update] executable path", "path", exePath)

	// Determine VERSION file location
	// If running with "go run", use working directory instead of temp go-build dir
	versionDir := filepath.Dir(exePath)
	if strings.Contains(exePath, "go-build") {
		// Running with "go run" - use working directory
		if wd, err := os.Getwd(); err == nil {
			versionDir = wd
			config.Logger.Info("[update] detected go run mode, using working directory", "dir", versionDir)
		}
	}

	config.Logger.Info("[update] version directory", "dir", versionDir)

	// Update VERSION file BEFORE replacing binary
	versionFile := filepath.Join(versionDir, "VERSION")
	config.Logger.Info("[update] writing VERSION file", "path", versionFile, "version", newVersion)

	// Check if directory is writable
	testFile := filepath.Join(versionDir, ".test_write")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		config.Logger.Error("[update] directory not writable", "error", err, "dir", versionDir)
		return fmt.Errorf("directory not writable: %w", err)
	}
	os.Remove(testFile)

	if err := os.WriteFile(versionFile, []byte(newVersion), 0644); err != nil {
		config.Logger.Error("[update] failed to write version file", "error", err, "path", versionFile)
		return fmt.Errorf("write version file: %w", err)
	}

	config.Logger.Info("[update] updated version file", "new_version", newVersion, "path", versionFile)

	// Verify file was written
	if content, err := os.ReadFile(versionFile); err != nil {
		config.Logger.Error("[update] failed to verify version file", "error", err)
	} else {
		config.Logger.Info("[update] verified version file content", "content", string(content))
	}

	// Update manager's current version
	m.SetCurrentVersion(newVersion)
	// Reload version package cache
	version.Reload()
	config.Logger.Info("[update] reloaded version cache")

	// Replace current binary (atomic operation)
	// On Windows, we need to rename old binary first
	// But if binary is running, this will fail
	oldBinaryPath := exePath + ".old"

	config.Logger.Info("[update] attempting to rename current binary", "from", exePath, "to", oldBinaryPath)

	if err := os.Rename(exePath, oldBinaryPath); err != nil {
		config.Logger.Error("[update] failed to rename binary - binary may be in use", "error", err)
		return fmt.Errorf("rename old binary (binary may be running): %w", err)
	}

	config.Logger.Info("[update] successfully renamed old binary")

	// Copy new binary
	config.Logger.Info("[update] copying new binary", "from", newBinaryPath, "to", exePath)

	if err := m.copyFile(newBinaryPath, exePath); err != nil {
		// Rollback
		config.Logger.Error("[update] failed to copy new binary, rolling back", "error", err)
		os.Rename(oldBinaryPath, exePath)
		return fmt.Errorf("copy new binary: %w", err)
	}

	config.Logger.Info("[update] successfully copied new binary")

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
	// Detect current platform
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	config.Logger.Info("[update] detecting platform", "os", goos, "arch", goarch)

	// Platform-specific patterns
	var pattern string
	switch goos {
	case "windows":
		pattern = "windows"
	case "darwin":
		pattern = "darwin"
	case "linux":
		pattern = "linux"
	default:
		config.Logger.Warn("[update] unknown platform", "os", goos)
		return nil
	}

	// Find matching asset
	for i := range assets {
		name := strings.ToLower(assets[i].Name)
		if strings.Contains(name, pattern) && strings.Contains(name, goarch) {
			config.Logger.Info("[update] found matching asset", "name", assets[i].Name)
			return &assets[i]
		}
	}

	// Fallback: try without arch check
	for i := range assets {
		name := strings.ToLower(assets[i].Name)
		if strings.Contains(name, pattern) {
			config.Logger.Info("[update] found platform asset (no arch match)", "name", assets[i].Name)
			return &assets[i]
		}
	}

	config.Logger.Warn("[update] no matching asset found", "platform", goos, "arch", goarch)
	return nil
}
