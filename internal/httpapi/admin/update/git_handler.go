package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ds2api/internal/config"
	"ds2api/internal/update"
)

// GitHandler handles Git-based update requests
type GitHandler struct {
	GitManager    *update.GitManager
	mu            sync.RWMutex
	currentStatus *update.UpdateStatus
}

// NewGitHandler creates a new Git update handler
func NewGitHandler(gitManager *update.GitManager) *GitHandler {
	return &GitHandler{
		GitManager: gitManager,
		currentStatus: &update.UpdateStatus{
			Stage:    "idle",
			Progress: 0,
			Message:  "No update in progress",
		},
	}
}

// CheckUpdate handles GET /admin/update/check
func (h *GitHandler) CheckUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	config.Logger.Info("[git-update] checking for updates")

	updateInfo, err := h.GitManager.CheckUpdate(ctx)
	if err != nil {
		config.Logger.Warn("[git-update] check failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if !updateInfo.HasUpdate {
		// Already up to date
		json.NewEncoder(w).Encode(map[string]interface{}{
			"update_available": false,
			"current_commit":   updateInfo.CurrentCommit[:7],
			"message":          "Already up to date",
		})
		return
	}

	// New commits available
	json.NewEncoder(w).Encode(map[string]interface{}{
		"update_available": true,
		"current_commit":   updateInfo.CurrentCommit[:7],
		"latest_commit":    updateInfo.LatestCommit[:7],
		"commits_behind":   updateInfo.CommitsBehind,
		"commit_messages":  updateInfo.CommitMessages,
		"fetched_at":       updateInfo.FetchedAt,
	})
}

// InstallUpdate handles POST /admin/update/install
func (h *GitHandler) InstallUpdate(w http.ResponseWriter, r *http.Request) {
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

// performUpdate performs the actual Git-based update process
func (h *GitHandler) performUpdate(ctx context.Context) {
	h.updateStatus("checking", 0, "Checking for updates...")

	// Step 1: Check for updates
	updateInfo, err := h.GitManager.CheckUpdate(ctx)
	if err != nil {
		h.updateStatus("failed", 0, "Failed to check for updates", err.Error())
		return
	}

	if !updateInfo.HasUpdate {
		h.updateStatus("complete", 100, "Already up to date")
		return
	}

	h.updateStatus("backing_up", 20, "Creating backup...")

	// Step 2: Create backup
	backupPath, err := h.GitManager.BackupCurrent(ctx)
	if err != nil {
		h.updateStatus("failed", 20, "Failed to create backup", err.Error())
		return
	}

	h.updateStatus("merging", 40, "Merging upstream changes...")

	// Step 3: Merge upstream
	if err := h.GitManager.MergeUpstream(ctx); err != nil {
		h.updateStatus("failed", 40, "Failed to merge upstream", err.Error())
		// Attempt rollback
		config.Logger.Warn("[git-update] merge failed, attempting rollback")
		if rollbackErr := h.GitManager.Rollback(ctx, backupPath); rollbackErr != nil {
			config.Logger.Error("[git-update] rollback failed", "error", rollbackErr)
		}
		return
	}

	h.updateStatus("building", 60, "Rebuilding application...")

	// Step 4: Rebuild
	if err := h.GitManager.Rebuild(ctx); err != nil {
		h.updateStatus("failed", 60, "Failed to rebuild", err.Error())
		// Attempt rollback
		config.Logger.Warn("[git-update] rebuild failed, attempting rollback")
		if rollbackErr := h.GitManager.Rollback(ctx, backupPath); rollbackErr != nil {
			config.Logger.Error("[git-update] rollback failed", "error", rollbackErr)
		}
		return
	}

	h.updateStatus("updating_version", 80, "Updating version...")

	// Step 5: Update version
	if err := h.GitManager.UpdateVersion(ctx); err != nil {
		config.Logger.Warn("[git-update] failed to update version", "error", err)
		// Not critical, continue
	}

	h.updateStatus("complete", 100, "Update complete! Server will restart in 3 seconds...")

	config.Logger.Info("[git-update] update complete, restarting server in 3 seconds")

	// Delay restart to allow response to be sent
	go func() {
		time.Sleep(3 * time.Second)
		config.Logger.Info("[git-update] initiating server restart")
		if err := restartServer(); err != nil {
			config.Logger.Error("[git-update] failed to restart server", "error", err)
		}
	}()
}

// GetStatus handles GET /admin/update/status
func (h *GitHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	status := *h.currentStatus
	h.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// ListBackups handles GET /admin/update/backups
func (h *GitHandler) ListBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := listBackups(h.GitManager.BackupDir)
	if err != nil {
		config.Logger.Warn("[git-update] failed to list backups", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get info for each backup
	backupInfos := make([]map[string]interface{}, 0, len(backups))
	for _, backup := range backups {
		info, err := getBackupInfo(h.GitManager.BackupDir, backup)
		if err != nil {
			config.Logger.Warn("[git-update] failed to get backup info", "backup", backup, "error", err)
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
func (h *GitHandler) Rollback(w http.ResponseWriter, r *http.Request) {
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

	config.Logger.Info("[git-update] rolling back", "backup", req.BackupName)

	// Perform rollback
	backupPath := h.GitManager.BackupDir + "/" + req.BackupName
	if err := h.GitManager.Rollback(ctx, backupPath); err != nil {
		config.Logger.Warn("[git-update] rollback failed", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Rollback complete! Please restart the server.",
		"backup":  req.BackupName,
	})
}

// updateStatus updates the current update status
func (h *GitHandler) updateStatus(stage string, progress int, message string, errorMsg ...string) {
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

	config.Logger.Info("[git-update] status", "stage", stage, "progress", progress, "message", message)
}

// Helper functions
func listBackups(backupDir string) ([]string, error) {
	// Reuse from old manager
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	backups := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			backups = append(backups, entry.Name())
		}
	}

	return backups, nil
}

func getBackupInfo(backupDir, backupName string) (map[string]interface{}, error) {
	backupPath := filepath.Join(backupDir, backupName)

	info := map[string]interface{}{
		"name": backupName,
	}

	// Get creation time from directory
	stat, err := os.Stat(backupPath)
	if err != nil {
		return nil, err
	}
	info["created_at"] = stat.ModTime()

	// Check if commit file exists
	commitFile := filepath.Join(backupPath, "commit.txt")
	if commitData, err := os.ReadFile(commitFile); err == nil {
		commit := strings.TrimSpace(string(commitData))
		if len(commit) > 7 {
			commit = commit[:7]
		}
		info["commit"] = commit
	}

	// Check if binary exists
	binaryPath := filepath.Join(backupPath, "ds2api.exe")
	if binaryStat, err := os.Stat(binaryPath); err == nil {
		info["binary_exists"] = true
		info["binary_size"] = binaryStat.Size()
	} else {
		info["binary_exists"] = false
	}

	return info, nil
}

// restartServer performs graceful server restart
func restartServer() error {
	// Get current executable path
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable: %w", err)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Prepare restart command
	var cmd *exec.Cmd

	// Check if running with "go run"
	if strings.Contains(executable, "go-build") {
		// Running with "go run" - restart with "go run"
		cmd = exec.Command("go", "run", "./cmd/ds2api")
		cmd.Dir = cwd
	} else {
		// Running compiled binary - restart with same binary
		cmd = exec.Command(executable, os.Args[1:]...)
		cmd.Dir = cwd
	}

	// Inherit environment
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start new process
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start new process: %w", err)
	}

	config.Logger.Info("[git-update] new process started", "pid", cmd.Process.Pid)

	// Exit current process to allow new one to take over
	// Use os.Exit instead of syscall.Kill to ensure graceful shutdown
	go func() {
		time.Sleep(500 * time.Millisecond)
		config.Logger.Info("[git-update] shutting down current process")
		os.Exit(0)
	}()

	return nil
}
