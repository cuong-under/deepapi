package update

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"ds2api/internal/config"
)

// copyFile copies a file from src to dst
func (m *Manager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	if err := os.Chmod(dst, sourceInfo.Mode()); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	return nil
}

// extractArchive extracts a zip archive to destination directory
func (m *Manager) extractArchive(archivePath, destDir string) error {
	config.Logger.Info("[update] extracting archive", "archive", archivePath, "dest", destDir)

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		path := filepath.Join(destDir, file.Name)

		// Check for ZipSlip vulnerability
		if !filepath.HasPrefix(path, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, file.Mode()); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}
			continue
		}

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("create parent directory: %w", err)
		}

		// Extract file
		if err := m.extractFile(file, path); err != nil {
			return fmt.Errorf("extract file %s: %w", file.Name, err)
		}
	}

	config.Logger.Info("[update] extraction complete")
	return nil
}

// extractFile extracts a single file from zip archive
func (m *Manager) extractFile(file *zip.File, dest string) error {
	rc, err := file.Open()
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer rc.Close()

	outFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, rc); err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	// Set file permissions
	if err := os.Chmod(dest, file.Mode()); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	return nil
}

// CleanupOldBackups removes backups older than specified days
func (m *Manager) CleanupOldBackups(maxAge int) error {
	config.Logger.Info("[update] cleaning up old backups", "max_age_days", maxAge)

	backups, err := m.ListBackups()
	if err != nil {
		return fmt.Errorf("list backups: %w", err)
	}

	cutoff := time.Now().AddDate(0, 0, -maxAge)
	removed := 0

	for _, backup := range backups {
		backupPath := filepath.Join(m.BackupDir, backup)
		info, err := os.Stat(backupPath)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.RemoveAll(backupPath); err != nil {
				config.Logger.Warn("[update] failed to remove old backup", "backup", backup, "error", err)
				continue
			}
			removed++
		}
	}

	config.Logger.Info("[update] cleanup complete", "removed", removed)
	return nil
}

// GetBackupInfo returns information about a backup
func (m *Manager) GetBackupInfo(backupName string) (map[string]interface{}, error) {
	backupPath := filepath.Join(m.BackupDir, backupName)

	info, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("stat backup: %w", err)
	}

	// Check if binary exists
	binaryPath := filepath.Join(backupPath, "ds2api.exe")
	binaryExists := false
	var binarySize int64
	if binaryInfo, err := os.Stat(binaryPath); err == nil {
		binaryExists = true
		binarySize = binaryInfo.Size()
	}

	// Check if config exists
	configPath := filepath.Join(backupPath, "config.json")
	configExists := false
	if _, err := os.Stat(configPath); err == nil {
		configExists = true
	}

	return map[string]interface{}{
		"name":          backupName,
		"created_at":    info.ModTime(),
		"binary_exists": binaryExists,
		"binary_size":   binarySize,
		"config_exists": configExists,
	}, nil
}
