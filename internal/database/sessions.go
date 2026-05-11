package database

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type Session struct {
	ID        int64  `json:"id"`
	UserID    int64  `json:"user_id"`
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	CreatedAt int64  `json:"created_at"`
}

// GenerateToken generates a secure random token
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// CreateSession creates a new session for a user with the given JWT token
func (db *DB) CreateSession(userID int64, jwtToken string, expiresIn time.Duration) (*Session, error) {
	now := time.Now().Unix()
	expiresAt := time.Now().Add(expiresIn).Unix()

	result, err := db.Exec(`
		INSERT INTO sessions (user_id, token, expires_at, created_at)
		VALUES (?, ?, ?, ?)
	`, userID, jwtToken, expiresAt, now)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get session id: %w", err)
	}

	return &Session{
		ID:        id,
		UserID:    userID,
		Token:     jwtToken,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

// GetSessionByToken retrieves a session by token
func (db *DB) GetSessionByToken(token string) (*Session, error) {
	var s Session
	err := db.QueryRow(`
		SELECT id, user_id, token, expires_at, created_at
		FROM sessions WHERE token = ?
	`, token).Scan(&s.ID, &s.UserID, &s.Token, &s.ExpiresAt, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query session: %w", err)
	}

	// Check if expired
	if time.Now().Unix() > s.ExpiresAt {
		// Delete expired session
		db.DeleteSession(s.ID)
		return nil, fmt.Errorf("session expired")
	}

	return &s, nil
}

// GetSessionsByUserID retrieves all sessions for a user
func (db *DB) GetSessionsByUserID(userID int64) ([]*Session, error) {
	rows, err := db.Query(`
		SELECT id, user_id, token, expires_at, created_at
		FROM sessions
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.UserID, &s.Token, &s.ExpiresAt, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, &s)
	}

	return sessions, nil
}

// DeleteSession deletes a session by ID
func (db *DB) DeleteSession(id int64) error {
	_, err := db.Exec("DELETE FROM sessions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteSessionByToken deletes a session by token
func (db *DB) DeleteSessionByToken(token string) error {
	_, err := db.Exec("DELETE FROM sessions WHERE token = ?", token)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

// DeleteUserSessions deletes all sessions for a user
func (db *DB) DeleteUserSessions(userID int64) error {
	_, err := db.Exec("DELETE FROM sessions WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("delete user sessions: %w", err)
	}
	return nil
}

// CleanupExpiredSessions deletes all expired sessions
func (db *DB) CleanupExpiredSessions() (int64, error) {
	now := time.Now().Unix()
	result, err := db.Exec("DELETE FROM sessions WHERE expires_at < ?", now)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired sessions: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return count, nil
}

// ExtendSession extends the expiration time of a session
func (db *DB) ExtendSession(token string, expiresIn time.Duration) error {
	expiresAt := time.Now().Add(expiresIn).Unix()
	result, err := db.Exec(`
		UPDATE sessions SET expires_at = ?
		WHERE token = ?
	`, expiresAt, token)
	if err != nil {
		return fmt.Errorf("extend session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("session not found")
	}

	return nil
}

// CountActiveSessions returns the number of active (non-expired) sessions
func (db *DB) CountActiveSessions() (int, error) {
	now := time.Now().Unix()
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM sessions WHERE expires_at > ?", now).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active sessions: %w", err)
	}
	return count, nil
}

// CountUserActiveSessions returns the number of active sessions for a user
func (db *DB) CountUserActiveSessions(userID int64) (int, error) {
	now := time.Now().Unix()
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM sessions
		WHERE user_id = ? AND expires_at > ?
	`, userID, now).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user active sessions: %w", err)
	}
	return count, nil
}
