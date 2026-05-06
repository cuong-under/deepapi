package update

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"ds2api/internal/config"
	"ds2api/internal/update"
)

// Handler handles update-related HTTP requests
type Handler struct {
	UpdateManager *update.Manager
	mu            sync.RWMutex
	currentStatus *update.UpdateStatus
}

// NewHandler creates a new update handler
func NewHandler(updateManager *update.Manager) *Handler {
	return &Handler{
		UpdateManager: updateManager,
		currentStatus: &update.UpdateStatus{
			Stage:    "idle",
			Progress: 0,
			Message:  "No update in progress",
		},
	}
}

// CheckUpdate handles GET /admin/update/check
func (h *Handler) CheckUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	config.Logger.Info("[update] checking for updates")

	release, err := h.UpdateManager.CheckUpdate(ctx)
	if err != nil {
		config.Logger.Warn("[update] check failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if release == nil {
		// Already up to date
		json.NewEncoder(w).Encode(map[string]interface{}{
			"update_available": false,
			"current_version":  h.UpdateManager.GetCurrentVersion(),
			"message":          "Already up to date",
		})
		return
	}

	// New version available
	json.NewEncoder(w).Encode(map[string]interface{}{
		"update_available": true,
		"current_version":  h.UpdateManager.GetCurrentVersion(),
		"latest_version":   release.TagName,
		"release_name":     release.Name,
		"release_notes":    release.Body,
		"published_at":     release.PublishedAt,
		"html_url":         release.HTMLURL,
	})
}

// InstallUpdate handles POST /admin/update/install
func (h *Handler) InstallUpdate(w http.ResponseWriter, r *http.Request) {
	// Check if update is already in progress
	h.mu.RLock()
	if h.currentStatus.Stage != "idle" && h.currentStatus.Stage != "complete" && h.currentStatus.Stage != "failed" {
		h.mu.RUnlock()
		http.Error(w, "Update already in progress", http.StatusConflict)
		return
	}
	h.mu.RUnlock()

	// Start update in background
	go h.performUpdate(context.Background())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Update started",
		"status":  "in_progress",
	})
}

// performUpdate performs the actual update process
func (h *Handler) performUpdate(ctx context.Context) {
	h.updateStatus("checking", 0, "Checking for updates...")

	// Step 1: Check for updates
	release, err := h.UpdateManager.CheckUpdate(ctx)
	if err != nil {
		h.updateStatus("failed", 0, "Failed to check for updates", err.Error())
		return
	}

	if release == nil {
		h.updateStatus("complete", 100, "Already up to date")
		return
	}

	h.updateStatus("downloading", 10, "Downloading update...")

	// Step 2: Download update
	archivePath, err := h.UpdateManager.DownloadUpdate(ctx, release)
	if err != nil {
		h.updateStatus("failed", 10, "Failed to download update", err.Error())
		return
	}

	h.updateStatus("backing_up", 40, "Creating backup...")

	// Step 3: Create backup
	backupPath, err := h.UpdateManager.BackupCurrent(ctx)
	if err != nil {
		h.updateStatus("failed", 40, "Failed to create backup", err.Error())
		return
	}

	h.updateStatus("installing", 60, "Installing update...")

	// Step 4: Install update
	if err := h.UpdateManager.InstallUpdate(ctx, archivePath, release.TagName); err != nil {
		h.updateStatus("failed", 60, "Failed to install update", err.Error())
		// Attempt rollback
		config.Logger.Warn("[update] installation failed, attempting rollback")
		if rollbackErr := h.UpdateManager.Rollback(ctx, backupPath); rollbackErr != nil {
			config.Logger.Error("[update] rollback failed", "error", rollbackErr)
		}
		return
	}

	h.updateStatus("verifying", 80, "Verifying installation...")

	// Step 5: Verify health
	if err := h.UpdateManager.VerifyHealth(ctx); err != nil {
		h.updateStatus("failed", 80, "Health check failed", err.Error())
		// Attempt rollback
		config.Logger.Warn("[update] health check failed, attempting rollback")
		if rollbackErr := h.UpdateManager.Rollback(ctx, backupPath); rollbackErr != nil {
			config.Logger.Error("[update] rollback failed", "error", rollbackErr)
		}
		return
	}

	h.updateStatus("complete", 100, "Update complete! Server will restart...")

	// Step 6: Restart server (delayed to allow response to be sent)
	time.Sleep(2 * time.Second)
	// TODO: Implement graceful restart
	config.Logger.Info("[update] update complete, restart required")
}

// GetStatus handles GET /admin/update/status
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	status := *h.currentStatus
	h.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// ListBackups handles GET /admin/update/backups
func (h *Handler) ListBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := h.UpdateManager.ListBackups()
	if err != nil {
		config.Logger.Warn("[update] failed to list backups", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get info for each backup
	backupInfos := make([]map[string]interface{}, 0, len(backups))
	for _, backup := range backups {
		info, err := h.UpdateManager.GetBackupInfo(backup)
		if err != nil {
			config.Logger.Warn("[update] failed to get backup info", "backup", backup, "error", err)
			continue
		}
		backupInfos = append(backupInfos, info)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"backups": backupInfos,
		"count":   len(backupInfos),
	})
}

// Rollback handles POST /admin/update/rollback
func (h *Handler) Rollback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req struct {
		BackupName string `json:"backup_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.BackupName == "" {
		http.Error(w, "backup_name is required", http.StatusBadRequest)
		return
	}

	config.Logger.Info("[update] rolling back", "backup", req.BackupName)

	// Perform rollback
	backupPath := h.UpdateManager.BackupDir + "/" + req.BackupName
	if err := h.UpdateManager.Rollback(ctx, backupPath); err != nil {
		config.Logger.Warn("[update] rollback failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Rollback complete! Server will restart...",
		"backup":  req.BackupName,
	})

	// Restart server (delayed)
	time.Sleep(2 * time.Second)
	// TODO: Implement graceful restart
	config.Logger.Info("[update] rollback complete, restart required")
}

// updateStatus updates the current update status
func (h *Handler) updateStatus(stage string, progress int, message string, errorMsg ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.currentStatus.Stage = stage
	h.currentStatus.Progress = progress
	h.currentStatus.Message = message

	if len(errorMsg) > 0 {
		h.currentStatus.Error = errorMsg[0]
	} else {
		h.currentStatus.Error = ""
	}

	if stage == "checking" {
		h.currentStatus.StartedAt = time.Now()
	}

	if stage == "complete" || stage == "failed" {
		h.currentStatus.CompletedAt = time.Now()
	}

	config.Logger.Info("[update] status", "stage", stage, "progress", progress, "message", message)
}
