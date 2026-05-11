package database

import (
	"database/sql"
	"fmt"
	"time"
)

type UserAPIKey struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	APIKey    string `json:"api_key"`
	Name      string `json:"name"`
	Remark    string `json:"remark"`
	CreatedAt int64  `json:"created_at"`
}

// CreateAPIKey creates a new API key for a user
func (db *DB) CreateAPIKey(userID int64, apiKey, name, remark string) (*UserAPIKey, error) {
	now := time.Now().Unix()
	result, err := db.Exec(`
		INSERT INTO user_api_keys (user_id, api_key, name, remark, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, userID, apiKey, name, remark, now)
	if err != nil {
		return nil, fmt.Errorf("insert api key: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get api key id: %w", err)
	}

	return &UserAPIKey{
		ID:        id,
		UserID:    userID,
		APIKey:    apiKey,
		Name:      name,
		Remark:    remark,
		CreatedAt: now,
	}, nil
}

// GetAPIKeysByUserID gets all API keys for a user
func (db *DB) GetAPIKeysByUserID(userID int64) ([]*UserAPIKey, error) {
	rows, err := db.Query(`
		SELECT id, user_id, api_key, name, remark, created_at
		FROM user_api_keys
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query api keys: %w", err)
	}
	defer rows.Close()

	var keys []*UserAPIKey
	for rows.Next() {
		var k UserAPIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.APIKey, &k.Name, &k.Remark, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, &k)
	}

	return keys, nil
}

// GetAPIKeyOwner returns the user_id for an API key
func (db *DB) GetAPIKeyOwner(apiKey string) (int64, error) {
	var userID int64
	err := db.QueryRow("SELECT user_id FROM user_api_keys WHERE api_key = ?", apiKey).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("api key not found")
	}
	if err != nil {
		return 0, fmt.Errorf("query api key: %w", err)
	}
	return userID, nil
}

// GetKeyByValue gets an API key by its value
func (db *DB) GetKeyByValue(apiKey string) (*UserAPIKey, error) {
	var k UserAPIKey
	err := db.QueryRow(`
		SELECT id, user_id, api_key, name, remark, created_at
		FROM user_api_keys
		WHERE api_key = ?
	`, apiKey).Scan(&k.ID, &k.UserID, &k.APIKey, &k.Name, &k.Remark, &k.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("api key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query api key: %w", err)
	}
	return &k, nil
}

// UpdateAPIKey updates an API key (with ownership check)
func (db *DB) UpdateAPIKey(id, userID int64, name, remark string) error {
	result, err := db.Exec(`
		UPDATE user_api_keys
		SET name = ?, remark = ?
		WHERE id = ? AND user_id = ?
	`, name, remark, id, userID)
	if err != nil {
		return fmt.Errorf("update api key: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("api key not found or access denied")
	}

	return nil
}

// DeleteAPIKey deletes an API key (with ownership check)
func (db *DB) DeleteAPIKey(id, userID int64) error {
	result, err := db.Exec("DELETE FROM user_api_keys WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("api key not found or access denied")
	}

	return nil
}

// DeleteAPIKeyByKey deletes an API key by key string (with ownership check)
func (db *DB) DeleteAPIKeyByKey(apiKey string, userID int64) error {
	result, err := db.Exec("DELETE FROM user_api_keys WHERE api_key = ? AND user_id = ?", apiKey, userID)
	if err != nil {
		return fmt.Errorf("delete api key: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("api key not found or access denied")
	}

	return nil
}

// GetAllAPIKeys gets all API keys (admin only)
func (db *DB) GetAllAPIKeys() ([]*UserAPIKey, error) {
	rows, err := db.Query(`
		SELECT id, user_id, api_key, name, remark, created_at
		FROM user_api_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query api keys: %w", err)
	}
	defer rows.Close()

	var keys []*UserAPIKey
	for rows.Next() {
		var k UserAPIKey
		if err := rows.Scan(&k.ID, &k.UserID, &k.APIKey, &k.Name, &k.Remark, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api key: %w", err)
		}
		keys = append(keys, &k)
	}

	return keys, nil
}

// CountUserKeys returns the number of keys for a user
func (db *DB) CountUserKeys(userID int64) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM user_api_keys WHERE user_id = ?", userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user keys: %w", err)
	}
	return count, nil
}
