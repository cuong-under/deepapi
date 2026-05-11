package update

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ds2api/internal/config"
	"ds2api/internal/version"
)

const (
	defaultUpstreamRemote = "upstream"
	defaultUpstreamURL    = "https://github.com/CJackHwang/ds2api.git"
	defaultUpstreamBranch = "main"
)

// GitManager quản lý Git-based update process
type GitManager struct {
	repoDir        string
	currentBranch  string
	upstreamRemote string
	upstreamURL    string
	upstreamBranch string // "upstream/main"
	BackupDir      string
	dataDir        string
}

// GitUpdateInfo represents Git update information
type GitUpdateInfo struct {
	HasUpdate      bool      `json:"has_update"`
	CurrentCommit  string    `json:"current_commit"`
	LatestCommit   string    `json:"latest_commit"`
	CommitsBehind  int       `json:"commits_behind"`
	CommitMessages []string  `json:"commit_messages"`
	FetchedAt      time.Time `json:"fetched_at"`
}

// NewGitManager creates a new Git-based update manager
func NewGitManager(repoDir string) (*GitManager, error) {
	// Verify this is a Git repository
	gitDir := filepath.Join(repoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a git repository: %s", repoDir)
	}

	// Get current branch
	currentBranch, err := getCurrentBranch(repoDir)
	if err != nil {
		return nil, fmt.Errorf("get current branch: %w", err)
	}

	dataDir := filepath.Join(repoDir, "data")
	backupDir := filepath.Join(dataDir, "backups")

	upstreamRemote := strings.TrimSpace(os.Getenv("DS2API_UPSTREAM_REMOTE"))
	if upstreamRemote == "" {
		upstreamRemote = defaultUpstreamRemote
	}
	upstreamURL := strings.TrimSpace(os.Getenv("DS2API_UPSTREAM_URL"))
	if upstreamURL == "" {
		upstreamURL = defaultUpstreamURL
	}
	upstreamBranchName := strings.TrimSpace(os.Getenv("DS2API_UPSTREAM_BRANCH"))
	if upstreamBranchName == "" {
		upstreamBranchName = defaultUpstreamBranch
	}
	upstreamBranch := fmt.Sprintf("%s/%s", upstreamRemote, upstreamBranchName)

	return &GitManager{
		repoDir:        repoDir,
		currentBranch:  currentBranch,
		upstreamRemote: upstreamRemote,
		upstreamURL:    upstreamURL,
		upstreamBranch: upstreamBranch,
		BackupDir:      backupDir,
		dataDir:        dataDir,
	}, nil
}

// CheckUpdate checks if there are new commits on upstream
func (m *GitManager) CheckUpdate(ctx context.Context) (*GitUpdateInfo, error) {
	config.Logger.Info("[git-update] checking for updates", "branch", m.currentBranch, "upstream", m.upstreamBranch)

	if err := m.ensureUpstreamRemote(ctx); err != nil {
		return nil, fmt.Errorf("ensure upstream remote: %w", err)
	}

	// Fetch latest from remote
	if err := m.gitFetch(ctx); err != nil {
		return nil, fmt.Errorf("git fetch: %w", err)
	}

	// Get current commit
	currentCommit, err := m.getCurrentCommit(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current commit: %w", err)
	}

	// Get latest upstream commit
	latestCommit, err := m.getUpstreamCommit(ctx)
	if err != nil {
		return nil, fmt.Errorf("get upstream commit: %w", err)
	}

	// Get commits behind (commits in upstream/main not in current branch)
	commitsBehind, commitMessages, err := m.getCommitsBehind(ctx)
	if err != nil {
		config.Logger.Warn("[git-update] failed to get commits behind", "error", err)
		commitsBehind = 0
		commitMessages = []string{}
	}

	// Check if there are new commits to merge
	if commitsBehind == 0 {
		config.Logger.Info("[git-update] already up to date", "commit", currentCommit[:7])
		return &GitUpdateInfo{
			HasUpdate:      false,
			CurrentCommit:  currentCommit,
			LatestCommit:   latestCommit,
			CommitsBehind:  0,
			CommitMessages: []string{},
			FetchedAt:      time.Now(),
		}, nil
	}

	config.Logger.Info("[git-update] new commits available",
		"current", currentCommit[:7],
		"latest", latestCommit[:7],
		"behind", commitsBehind)

	return &GitUpdateInfo{
		HasUpdate:      true,
		CurrentCommit:  currentCommit,
		LatestCommit:   latestCommit,
		CommitsBehind:  commitsBehind,
		CommitMessages: commitMessages,
		FetchedAt:      time.Now(),
	}, nil
}

// BackupCurrent creates backup of current code
func (m *GitManager) BackupCurrent(ctx context.Context) (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(m.BackupDir, fmt.Sprintf("backup-%s", timestamp))

	config.Logger.Info("[git-update] creating backup", "path", backupPath)

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return "", fmt.Errorf("create backup directory: %w", err)
	}

	// Save current commit hash
	currentCommit, err := m.getCurrentCommit(ctx)
	if err != nil {
		return "", fmt.Errorf("get current commit: %w", err)
	}

	commitFile := filepath.Join(backupPath, "commit.txt")
	if err := os.WriteFile(commitFile, []byte(currentCommit), 0644); err != nil {
		return "", fmt.Errorf("write commit file: %w", err)
	}

	// Backup binary if exists
	binaryPath := filepath.Join(m.repoDir, "ds2api.exe")
	if _, err := os.Stat(binaryPath); err == nil {
		backupBinary := filepath.Join(backupPath, "ds2api.exe")
		if err := copyFile(binaryPath, backupBinary); err != nil {
			config.Logger.Warn("[git-update] failed to backup binary", "error", err)
		}
	}

	// Backup config
	configPath := filepath.Join(m.repoDir, "config.json")
	if _, err := os.Stat(configPath); err == nil {
		backupConfig := filepath.Join(backupPath, "config.json")
		if err := copyFile(configPath, backupConfig); err != nil {
			config.Logger.Warn("[git-update] failed to backup config", "error", err)
		}
	}

	config.Logger.Info("[git-update] backup complete", "path", backupPath, "commit", currentCommit[:7])

	// Cleanup old backups (keep only last 5)
	if err := m.cleanupOldBackups(5); err != nil {
		config.Logger.Warn("[git-update] failed to cleanup old backups", "error", err)
	}

	return backupPath, nil
}

// cleanupOldBackups removes old backups, keeping only the most recent N
func (m *GitManager) cleanupOldBackups(keepCount int) error {
	entries, err := os.ReadDir(m.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Filter only backup directories
	var backups []os.DirEntry
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "backup-") {
			backups = append(backups, entry)
		}
	}

	// If we have fewer backups than keepCount, nothing to delete
	if len(backups) <= keepCount {
		return nil
	}

	// Sort by name (which is timestamp-based, so newest first when reversed)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Name() > backups[j].Name()
	})

	// Delete old backups (keep only keepCount newest)
	for i := keepCount; i < len(backups); i++ {
		backupPath := filepath.Join(m.BackupDir, backups[i].Name())
		config.Logger.Info("[git-update] removing old backup", "path", backupPath)
		if err := os.RemoveAll(backupPath); err != nil {
			config.Logger.Warn("[git-update] failed to remove backup", "path", backupPath, "error", err)
		}
	}

	config.Logger.Info("[git-update] cleanup complete", "kept", keepCount, "removed", len(backups)-keepCount)
	return nil
}

// MergeUpstream merges upstream changes into current branch
func (m *GitManager) MergeUpstream(ctx context.Context) error {
	config.Logger.Info("[git-update] merging upstream", "from", m.upstreamBranch, "to", m.currentBranch)

	if err := m.ensureUpstreamRemote(ctx); err != nil {
		return fmt.Errorf("ensure upstream remote: %w", err)
	}

	// Check for uncommitted changes
	hasChanges, err := m.hasUncommittedChanges(ctx)
	if err != nil {
		return fmt.Errorf("check uncommitted changes: %w", err)
	}

	if hasChanges {
		return fmt.Errorf("working tree has uncommitted changes; commit or backup your custom changes before updating")
	}

	// Merge upstream
	if err := m.gitMerge(ctx, m.upstreamBranch); err != nil {
		config.Logger.Error("[git-update] merge failed", "error", err)

		// Check if merge conflict
		if strings.Contains(err.Error(), "CONFLICT") {
			return fmt.Errorf("merge conflict detected: %w", err)
		}

		return fmt.Errorf("git merge: %w", err)
	}

	config.Logger.Info("[git-update] merge complete")
	return nil
}

// ValidateMerge checks whether upstream can merge before creating backups or
// changing the working tree.
func (m *GitManager) ValidateMerge(ctx context.Context) error {
	if err := m.ensureUpstreamRemote(ctx); err != nil {
		return fmt.Errorf("ensure upstream remote: %w", err)
	}
	hasChanges, err := m.hasUncommittedChanges(ctx)
	if err != nil {
		return fmt.Errorf("check uncommitted changes: %w", err)
	}
	if hasChanges {
		return fmt.Errorf("working tree has uncommitted changes; commit or backup your custom changes before updating")
	}
	cmd := exec.CommandContext(ctx, "git", "merge-tree", "--write-tree", "HEAD", m.upstreamBranch)
	cmd.Dir = m.repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("upstream update has merge conflicts; manual merge required\n%s", strings.TrimSpace(string(output)))
	}
	return nil
}

// Rebuild rebuilds the application
func (m *GitManager) Rebuild(ctx context.Context) error {
	config.Logger.Info("[git-update] rebuilding application")

	// Build frontend first
	config.Logger.Info("[git-update] building frontend")
	webuiDir := filepath.Join(m.repoDir, "webui")
	if err := m.runCommand(ctx, webuiDir, "npm", "run", "build"); err != nil {
		return fmt.Errorf("build frontend: %w", err)
	}

	// Build backend
	config.Logger.Info("[git-update] building backend")
	outputBinary := filepath.Join(m.repoDir, "ds2api.exe")
	if err := m.runCommand(ctx, m.repoDir, "go", "build", "-o", outputBinary, "./cmd/ds2api"); err != nil {
		return fmt.Errorf("build backend: %w", err)
	}

	config.Logger.Info("[git-update] rebuild complete")
	return nil
}

// UpdateVersion updates VERSION file with latest commit
func (m *GitManager) UpdateVersion(ctx context.Context) error {
	// Get latest commit hash (short)
	commit, err := m.getCurrentCommit(ctx)
	if err != nil {
		return fmt.Errorf("get current commit: %w", err)
	}

	shortCommit := commit
	if len(commit) > 7 {
		shortCommit = commit[:7]
	}

	// Get commit date
	commitDate, err := m.getCommitDate(ctx, commit)
	if err != nil {
		config.Logger.Warn("[git-update] failed to get commit date", "error", err)
		commitDate = time.Now().Format("20060102")
	}

	// Format: v{date}-{commit}
	newVersion := fmt.Sprintf("v%s-%s", commitDate, shortCommit)

	versionFile := filepath.Join(m.repoDir, "VERSION")
	if err := os.WriteFile(versionFile, []byte(newVersion), 0644); err != nil {
		return fmt.Errorf("write version file: %w", err)
	}

	config.Logger.Info("[git-update] updated version", "version", newVersion)

	// Reload version cache
	version.Reload()

	return nil
}

// Rollback rolls back to a previous commit
func (m *GitManager) Rollback(ctx context.Context, backupPath string) error {
	config.Logger.Info("[git-update] rolling back", "backup", backupPath)

	// Read commit from backup
	commitFile := filepath.Join(backupPath, "commit.txt")
	commitBytes, err := os.ReadFile(commitFile)
	if err != nil {
		return fmt.Errorf("read commit file: %w", err)
	}

	targetCommit := strings.TrimSpace(string(commitBytes))

	// Reset to commit
	if err := m.gitReset(ctx, targetCommit); err != nil {
		return fmt.Errorf("git reset: %w", err)
	}

	// Restore binary if exists
	backupBinary := filepath.Join(backupPath, "ds2api.exe")
	if _, err := os.Stat(backupBinary); err == nil {
		binaryPath := filepath.Join(m.repoDir, "ds2api.exe")
		if err := copyFile(backupBinary, binaryPath); err != nil {
			config.Logger.Warn("[git-update] failed to restore binary", "error", err)
		}
	}

	config.Logger.Info("[git-update] rollback complete", "commit", targetCommit[:7])
	return nil
}

// Helper functions

func getCurrentBranch(repoDir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (m *GitManager) ensureUpstreamRemote(ctx context.Context) error {
	remoteURL, err := m.getRemoteURL(ctx, m.upstreamRemote)
	if err != nil {
		config.Logger.Info("[git-update] adding upstream remote", "remote", m.upstreamRemote, "url", m.upstreamURL)
		return m.runCommand(ctx, m.repoDir, "git", "remote", "add", m.upstreamRemote, m.upstreamURL)
	}
	if sameRemoteURL(remoteURL, m.upstreamURL) {
		return nil
	}
	return fmt.Errorf("remote %q points to %q, expected %q", m.upstreamRemote, remoteURL, m.upstreamURL)
}

func (m *GitManager) getRemoteURL(ctx context.Context, remote string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", remote)
	cmd.Dir = m.repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func sameRemoteURL(a, b string) bool {
	return normalizeRemoteURL(a) == normalizeRemoteURL(b)
}

func normalizeRemoteURL(raw string) string {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "/"))
	raw = strings.TrimSuffix(raw, ".git")
	if strings.HasPrefix(raw, "git@github.com:") {
		raw = "https://github.com/" + strings.TrimPrefix(raw, "git@github.com:")
	}
	if parsed, err := url.Parse(raw); err == nil && parsed.Host != "" {
		return strings.ToLower(parsed.Host + strings.TrimSuffix(parsed.Path, ".git"))
	}
	return strings.ToLower(raw)
}

func (m *GitManager) gitFetch(ctx context.Context) error {
	return m.runCommand(ctx, m.repoDir, "git", "fetch", m.upstreamRemote)
}

func (m *GitManager) getCurrentCommit(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = m.repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (m *GitManager) getUpstreamCommit(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", m.upstreamBranch)
	cmd.Dir = m.repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (m *GitManager) getCommitsBehind(ctx context.Context) (int, []string, error) {
	// Get commits in upstream/main that are NOT in current branch.
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--count", fmt.Sprintf("%s..%s", m.currentBranch, m.upstreamBranch))
	cmd.Dir = m.repoDir
	output, err := cmd.Output()
	if err != nil {
		return 0, nil, err
	}

	var count int
	fmt.Sscanf(string(output), "%d", &count)

	// Get commit messages (commits in upstream/main not in current branch)
	cmd = exec.CommandContext(ctx, "git", "log", "--oneline", fmt.Sprintf("%s..%s", m.currentBranch, m.upstreamBranch))
	cmd.Dir = m.repoDir
	output, err = cmd.Output()
	if err != nil {
		return count, nil, err
	}

	messages := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(messages) == 1 && messages[0] == "" {
		messages = []string{}
	}

	return count, messages, nil
}

func (m *GitManager) hasUncommittedChanges(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = m.repoDir
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return len(strings.TrimSpace(string(output))) > 0, nil
}

func (m *GitManager) gitMerge(ctx context.Context, branch string) error {
	return m.runCommand(ctx, m.repoDir, "git", "merge", branch, "--no-edit")
}

func (m *GitManager) gitReset(ctx context.Context, commit string) error {
	return m.runCommand(ctx, m.repoDir, "git", "reset", "--hard", commit)
}

func (m *GitManager) getCommitDate(ctx context.Context, commit string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "show", "-s", "--format=%ci", commit)
	cmd.Dir = m.repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse date (format: 2026-05-07 14:51:41 +0700)
	dateStr := strings.TrimSpace(string(output))
	parts := strings.Fields(dateStr)
	if len(parts) > 0 {
		// Return YYYYMMDD format
		date := strings.ReplaceAll(parts[0], "-", "")
		return date, nil
	}

	return "", fmt.Errorf("invalid date format")
}

func (m *GitManager) runCommand(ctx context.Context, dir string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		config.Logger.Error("[git-update] command failed",
			"cmd", name,
			"args", args,
			"output", string(output),
			"error", err)
		return fmt.Errorf("%s: %w\nOutput: %s", name, err, string(output))
	}

	config.Logger.Info("[git-update] command success", "cmd", name, "args", args)
	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
