package auth

import (
	"context"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"ds2api/internal/account"
	"ds2api/internal/config"
	"ds2api/internal/database"
)

func TestMultiUserResolverRoundRobinsDatabaseAccounts(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	user, err := db.CreateUser("user", "user@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := db.CreateAPIKey(user.ID, "sk-user", "key", ""); err != nil {
		t.Fatalf("create api key: %v", err)
	}
	if _, err := db.CreateAccount(user.ID, "one", "", "one@example.com", "", "pw", ""); err != nil {
		t.Fatalf("create account one: %v", err)
	}
	if _, err := db.CreateAccount(user.ID, "two", "", "two@example.com", "", "pw", ""); err != nil {
		t.Fatalf("create account two: %v", err)
	}

	store := &config.Store{}
	resolver := NewMultiUserResolver(db, store, account.NewPool(store), func(ctx context.Context, acc config.Account) (string, error) {
		return "token-" + acc.Identifier(), nil
	})

	firstReq := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	firstReq.Header.Set("Authorization", "Bearer sk-user")
	first, err := resolver.Determine(firstReq)
	if err != nil {
		t.Fatalf("determine first: %v", err)
	}

	secondReq := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	secondReq.Header.Set("Authorization", "Bearer sk-user")
	second, err := resolver.Determine(secondReq)
	if err != nil {
		t.Fatalf("determine second: %v", err)
	}

	if first.AccountID == "" || second.AccountID == "" {
		t.Fatalf("missing account IDs: first=%q second=%q", first.AccountID, second.AccountID)
	}
	if first.AccountID == second.AccountID {
		t.Fatalf("expected different accounts across round-robin calls, got %q", first.AccountID)
	}
}

func TestMultiUserResolverSkipsDisabledDatabaseAccounts(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	user, err := db.CreateUser("user", "user@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := db.CreateAPIKey(user.ID, "sk-user", "key", ""); err != nil {
		t.Fatalf("create api key: %v", err)
	}
	disabled, err := db.CreateAccount(user.ID, "disabled", "", "disabled@example.com", "", "pw", "")
	if err != nil {
		t.Fatalf("create disabled account: %v", err)
	}
	if err := db.SetAccountEnabled(disabled.ID, user.ID, false); err != nil {
		t.Fatalf("disable account: %v", err)
	}
	if _, err := db.CreateAccount(user.ID, "enabled", "", "enabled@example.com", "", "pw", ""); err != nil {
		t.Fatalf("create enabled account: %v", err)
	}

	store := &config.Store{}
	resolver := NewMultiUserResolver(db, store, account.NewPool(store), func(ctx context.Context, acc config.Account) (string, error) {
		if acc.Identifier() == "disabled@example.com" {
			t.Fatalf("disabled account should not be used")
		}
		return "token-" + acc.Identifier(), nil
	})

	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer sk-user")
	got, err := resolver.Determine(req)
	if err != nil {
		t.Fatalf("determine: %v", err)
	}
	if got.AccountID != "enabled@example.com" {
		t.Fatalf("expected enabled account, got %q", got.AccountID)
	}

	targetReq := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	targetReq.Header.Set("Authorization", "Bearer sk-user")
	targetReq.Header.Set("X-Ds2-Target-Account", "disabled@example.com")
	if _, err := resolver.Determine(targetReq); err == nil || err.Error() != "target account disabled" {
		t.Fatalf("expected disabled target error, got %v", err)
	}

}

func TestMultiUserResolverReturnsNoAccountWhenAllDatabaseAccountsDisabled(t *testing.T) {
	db, err := database.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	user, err := db.CreateUser("user", "user@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := db.CreateAPIKey(user.ID, "sk-user", "key", ""); err != nil {
		t.Fatalf("create api key: %v", err)
	}
	userAccount, err := db.CreateAccount(user.ID, "disabled", "", "disabled@example.com", "", "pw", "")
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if err := db.SetAccountEnabled(userAccount.ID, user.ID, false); err != nil {
		t.Fatalf("disable account: %v", err)
	}

	store := &config.Store{}
	resolver := NewMultiUserResolver(db, store, account.NewPool(store), func(context.Context, config.Account) (string, error) {
		t.Fatalf("login should not be called")
		return "", nil
	})

	req := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	req.Header.Set("Authorization", "Bearer sk-user")
	if _, err := resolver.Determine(req); !errors.Is(err, ErrNoUserAccount) {
		t.Fatalf("expected ErrNoUserAccount, got %v", err)
	}
}
