package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"ds2api/internal/account"
	"ds2api/internal/config"
	"ds2api/internal/database"
)

var (
	ErrNoUserAccount = errors.New("no accounts configured for this user")
)

// MultiUserResolver resolves authentication for multi-user mode
// It reads accounts from database instead of config.json
type MultiUserResolver struct {
	db             *database.DB
	legacyStore    *config.Store
	legacyPool     *account.Pool
	legacyLogin    LoginFunc
	legacyResolver *Resolver
	logger         *slog.Logger
	mu             sync.Mutex
	nextByUser     map[int64]int
}

// NewMultiUserResolver creates a new multi-user auth resolver
func NewMultiUserResolver(db *database.DB, legacyStore *config.Store, legacyPool *account.Pool, legacyLogin LoginFunc) *MultiUserResolver {
	return &MultiUserResolver{
		db:             db,
		legacyStore:    legacyStore,
		legacyPool:     legacyPool,
		legacyLogin:    legacyLogin,
		legacyResolver: NewResolver(legacyStore, legacyPool, legacyLogin),
		logger:         slog.Default(),
		nextByUser:     make(map[int64]int),
	}
}

// Determine resolves authentication from request
// In multi-user mode, it checks:
// 1. If user is authenticated via JWT (user_id in context) → use user's accounts from database
// 2. If token is DS2API key (in user_api_keys table) → use user's accounts from database
// 3. If token is DeepSeek token → pass through (direct mode)
func (r *MultiUserResolver) Determine(req *http.Request) (*RequestAuth, error) {
	ctx := req.Context()

	// Check if user is authenticated via JWT (from middleware)
	userID, hasUserID := GetUserID(ctx)
	if hasUserID {
		r.logger.Info("[MultiUserResolver] User authenticated via JWT",
			"user_id", userID,
			"path", req.URL.Path)

		callerKey := extractCallerToken(req)
		callerID := callerTokenID(callerKey)
		if callerID == "" {
			callerID = "jwt-user"
		}

		// User is authenticated via JWT - use their accounts
		return r.acquireManagedRequestAuthFromDB(ctx, userID, callerID, req)
	}

	// No JWT - check API key
	callerKey := extractCallerToken(req)
	r.logger.Info("[MultiUserResolver] Determine called (no JWT)",
		"has_token", callerKey != "",
		"token_prefix", truncateToken(callerKey),
		"path", req.URL.Path)

	if callerKey == "" {
		r.logger.Warn("[MultiUserResolver] No token provided")
		return nil, ErrUnauthorized
	}

	callerID := callerTokenID(callerKey)

	// Check if this is a DS2API key (from user_api_keys table)
	apiKey, err := r.db.GetKeyByValue(callerKey)
	if err == nil {
		r.logger.Info("[MultiUserResolver] Found DS2API key in database",
			"user_id", apiKey.UserID,
			"key_name", apiKey.Name)
		// This is a DS2API key - get user's accounts from database
		return r.acquireManagedRequestAuthFromDB(ctx, apiKey.UserID, callerID, req)
	}
	r.logger.Info("[MultiUserResolver] Token not found in database", "error", err)

	// Not a DS2API key - check if it's in legacy config.json
	if r.legacyStore.HasAPIKey(callerKey) {
		r.logger.Info("[MultiUserResolver] Found in legacy config, using legacy resolver")
		// Use legacy resolver for config.json keys
		return r.legacyResolver.Determine(req)
	}
	r.logger.Info("[MultiUserResolver] Token not in legacy config")

	// Not in database or config - assume it's a DeepSeek token (direct mode)
	r.logger.Info("[MultiUserResolver] Treating as DeepSeek token (direct mode)")
	return &RequestAuth{
		UseConfigToken: false,
		DeepSeekToken:  callerKey,
		CallerID:       callerID,
		resolver:       r.legacyResolver,
		TriedAccounts:  map[string]bool{},
	}, nil
}

func truncateToken(token string) string {
	if len(token) > 20 {
		return token[:20] + "..."
	}
	return token
}

// acquireManagedRequestAuthFromDB gets an account from user's database accounts
func (r *MultiUserResolver) acquireManagedRequestAuthFromDB(ctx context.Context, userID int64, callerID string, req *http.Request) (*RequestAuth, error) {
	// Get user's accounts from database
	accounts, err := r.db.GetAccountsByUserID(userID)
	if err != nil {
		r.logger.Error("[MultiUserResolver] Failed to get user accounts", "user_id", userID, "error", err)
		return nil, err
	}

	enabledAccounts := enabledUserAccounts(accounts)
	if len(enabledAccounts) == 0 {
		r.logger.Warn("[MultiUserResolver] User has no accounts", "user_id", userID)
		return nil, ErrNoUserAccount
	}

	r.logger.Info("[MultiUserResolver] Found user accounts", "user_id", userID, "count", len(accounts), "enabled_count", len(enabledAccounts))

	// IMPORTANT: Set user context in request so chat history can capture user_id
	// This ensures analytics can filter by user_id
	if !hasUserContext(ctx) {
		user, err := r.db.GetUserByID(userID)
		if err == nil {
			*req = *req.WithContext(SetUserContext(ctx, user.ID, user.Username, user.Role))
			r.logger.Info("[MultiUserResolver] Set user context for API key request", "user_id", userID)
		}
	}

	// Check if user specified a target account
	target := strings.TrimSpace(req.Header.Get("X-Ds2-Target-Account"))

	var candidates []*database.UserAccount
	if target != "" {
		r.logger.Info("[MultiUserResolver] Target account specified", "target", target)
		// User specified target account
		for _, acc := range accounts {
			if acc.Email == target || acc.Mobile == target {
				if !acc.Enabled {
					r.logger.Warn("[MultiUserResolver] Target account disabled", "target", target)
					return nil, errors.New("target account disabled")
				}
				candidates = []*database.UserAccount{acc}
				break
			}
		}
		if len(candidates) == 0 {
			r.logger.Warn("[MultiUserResolver] Target account not found", "target", target)
			return nil, errors.New("target account not found")
		}
	} else {
		candidates = r.roundRobinCandidates(userID, enabledAccounts)
	}

	var lastErr error
	for _, selectedAccount := range candidates {
		// Convert database account to config.Account format
		configAccount := config.Account{
			Email:    selectedAccount.Email,
			Mobile:   selectedAccount.Mobile,
			Password: selectedAccount.Password,
			Token:    "", // Will be refreshed if needed
			Name:     selectedAccount.Name,
			Remark:   selectedAccount.Remark,
		}

		accountID := selectedAccount.Email
		if accountID == "" {
			accountID = selectedAccount.Mobile
		}

		// Create RequestAuth
		a := &RequestAuth{
			UseConfigToken: true,
			CallerID:       callerID,
			AccountID:      accountID,
			Account:        configAccount,
			TriedAccounts:  map[string]bool{},
			resolver:       r.legacyResolver,
		}

		r.logger.Info("[MultiUserResolver] Ensuring token for account", "account_id", accountID)

		// Ensure token is valid (login if needed)
		if err := r.ensureToken(ctx, a, selectedAccount.ID); err != nil {
			lastErr = err
			r.logger.Warn("[MultiUserResolver] Failed to ensure token, trying next account", "account_id", accountID, "error", err)
			continue
		}

		r.logger.Info("[MultiUserResolver] Successfully acquired account", "account_id", accountID, "has_token", a.DeepSeekToken != "")
		return a, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrNoUserAccount
}

func enabledUserAccounts(accounts []*database.UserAccount) []*database.UserAccount {
	enabled := make([]*database.UserAccount, 0, len(accounts))
	for _, acc := range accounts {
		if acc.Enabled {
			enabled = append(enabled, acc)
		}
	}
	return enabled
}

func (r *MultiUserResolver) roundRobinCandidates(userID int64, accounts []*database.UserAccount) []*database.UserAccount {
	r.mu.Lock()
	start := r.nextByUser[userID] % len(accounts)
	r.nextByUser[userID] = (start + 1) % len(accounts)
	r.mu.Unlock()

	candidates := make([]*database.UserAccount, 0, len(accounts))
	for i := 0; i < len(accounts); i++ {
		candidates = append(candidates, accounts[(start+i)%len(accounts)])
	}
	return candidates
}

func hasUserContext(ctx context.Context) bool {
	_, ok := GetUserContext(ctx)
	return ok
}

// ensureToken ensures the account has a valid DeepSeek token
func (r *MultiUserResolver) ensureToken(ctx context.Context, a *RequestAuth, accountID int64) error {
	// Try to login and get token
	token, err := r.legacyLogin(ctx, a.Account)
	if err != nil {
		return err
	}

	a.DeepSeekToken = token
	a.Account.Token = token

	if err := r.db.UpdateAccountRefreshTime(accountID); err != nil {
		r.logger.Warn("[MultiUserResolver] failed to update account refresh time", "account_id", accountID, "error", err)
	}

	return nil
}

// DetermineCaller resolves caller identity without acquiring account
func (r *MultiUserResolver) DetermineCaller(req *http.Request) (*RequestAuth, error) {
	return r.legacyResolver.DetermineCaller(req)
}

// Release releases the account back to pool
func (r *MultiUserResolver) Release(a *RequestAuth) {
	if r.legacyResolver != nil {
		r.legacyResolver.Release(a)
	}
}

// RefreshToken refreshes the token for the account
func (r *MultiUserResolver) RefreshToken(ctx context.Context, a *RequestAuth) bool {
	if r.legacyResolver != nil {
		return r.legacyResolver.RefreshToken(ctx, a)
	}
	return false
}

// SwitchAccount switches to another account
func (r *MultiUserResolver) SwitchAccount(ctx context.Context, a *RequestAuth) bool {
	if r.legacyResolver != nil {
		return r.legacyResolver.SwitchAccount(ctx, a)
	}
	return false
}
