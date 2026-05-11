package database

import (
	"database/sql"
	"fmt"
	"time"
)

type UserAccount struct {
	ID              int64   `json:"id"`
	UserID          int64   `json:"user_id"`
	Name            string  `json:"name"`
	Remark          string  `json:"remark"`
	Email           string  `json:"email"`
	Mobile          string  `json:"mobile"`
	Password        string  `json:"password"`
	ProxyID         string  `json:"proxy_id"`
	CreatedAt       int64   `json:"created_at"`
	LastRefreshedAt *int64  `json:"last_refreshed_at,omitempty"`
}

// CreateAccount creates a new account for a user
func (db *DB) CreateAccount(userID int64, name, remark, email, mobile, password, proxyID string) (*UserAccount, error) {
	now := time.Now().Unix()
	result, err := db.Exec(`
		INSERT INTO user_accounts (user_id, name, remark, email, mobile, password, proxy_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, name, remark, email, mobile, password, proxyID, now)
	if err != nil {
		return nil, fmt.Errorf("insert account: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get account id: %w", err)
	}

	return &UserAccount{
		ID:        id,
		UserID:    userID,
		Name:      name,
		Remark:    remark,
		Email:     email,
		Mobile:    mobile,
		Password:  password,
		ProxyID:   proxyID,
		CreatedAt: now,
	}, nil
}

// GetAccountsByUserID gets all accounts for a user
func (db *DB) GetAccountsByUserID(userID int64) ([]*UserAccount, error) {
	rows, err := db.Query(`
		SELECT id, user_id, name, remark, email, mobile, password, proxy_id, created_at, last_refreshed_at
		FROM user_accounts
		WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*UserAccount
	for rows.Next() {
		var a UserAccount
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Remark, &a.Email, &a.Mobile, &a.Password, &a.ProxyID, &a.CreatedAt, &a.LastRefreshedAt); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, &a)
	}

	return accounts, nil
}

// GetAccountByID gets an account by ID
func (db *DB) GetAccountByID(id int64) (*UserAccount, error) {
	var a UserAccount
	err := db.QueryRow(`
		SELECT id, user_id, name, remark, email, mobile, password, proxy_id, created_at, last_refreshed_at
		FROM user_accounts WHERE id = ?
	`, id).Scan(&a.ID, &a.UserID, &a.Name, &a.Remark, &a.Email, &a.Mobile, &a.Password, &a.ProxyID, &a.CreatedAt, &a.LastRefreshedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query account: %w", err)
	}
	return &a, nil
}

// UpdateAccount updates an account (with ownership check)
func (db *DB) UpdateAccount(id, userID int64, name, remark, email, mobile, password, proxyID string) error {
	result, err := db.Exec(`
		UPDATE user_accounts
		SET name = ?, remark = ?, email = ?, mobile = ?, password = ?, proxy_id = ?
		WHERE id = ? AND user_id = ?
	`, name, remark, email, mobile, password, proxyID, id, userID)
	if err != nil {
		return fmt.Errorf("update account: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("account not found or access denied")
	}

	return nil
}

// DeleteAccount deletes an account (with ownership check)
func (db *DB) DeleteAccount(id, userID int64) error {
	result, err := db.Exec("DELETE FROM user_accounts WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return fmt.Errorf("delete account: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("account not found or access denied")
	}

	return nil
}

// GetAllAccounts gets all accounts (admin only)
func (db *DB) GetAllAccounts() ([]*UserAccount, error) {
	rows, err := db.Query(`
		SELECT id, user_id, name, remark, email, mobile, password, proxy_id, created_at, last_refreshed_at
		FROM user_accounts
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*UserAccount
	for rows.Next() {
		var a UserAccount
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Remark, &a.Email, &a.Mobile, &a.Password, &a.ProxyID, &a.CreatedAt, &a.LastRefreshedAt); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, &a)
	}

	return accounts, nil
}

// CountUserAccounts returns the number of accounts for a user
func (db *DB) CountUserAccounts(userID int64) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM user_accounts WHERE user_id = ?", userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count user accounts: %w", err)
	}
	return count, nil
}

// UpdateAccountRefreshTime updates the last_refreshed_at timestamp for an account
func (db *DB) UpdateAccountRefreshTime(id int64) error {
	now := time.Now().Unix()
	_, err := db.Exec(`
		UPDATE user_accounts
		SET last_refreshed_at = ?
		WHERE id = ?
	`, now, id)
	if err != nil {
		return fmt.Errorf("update refresh time: %w", err)
	}
	return nil
}
