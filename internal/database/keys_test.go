package database

import (
	"path/filepath"
	"testing"
)

func TestCountUserKeysUsesUserAPIKeysTable(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	user, err := db.CreateUser("user", "user@example.com", "password123", "user")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := db.CreateAPIKey(user.ID, "sk-one", "one", ""); err != nil {
		t.Fatalf("create key one: %v", err)
	}
	if _, err := db.CreateAPIKey(user.ID, "sk-two", "two", ""); err != nil {
		t.Fatalf("create key two: %v", err)
	}

	count, err := db.CountUserKeys(user.ID)
	if err != nil {
		t.Fatalf("count keys: %v", err)
	}
	if count != 2 {
		t.Fatalf("count=%d, want 2", count)
	}
}
