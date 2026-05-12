package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type UserAccount struct {
	ID              int64  `json:"id"`
	UserID          int64  `json:"user_id"`
	Name            string `json:"name"`
	Remark          string `json:"remark"`
	Email           string `json:"email"`
	Mobile          string `json:"mobile"`
	Password        string `json:"password"`
	ProxyID         string `json:"proxy_id"`
	Enabled         bool   `json:"enabled"`
	CreatedAt       int64  `json:"created_at"`
	LastRefreshedAt *int64 `json:"last_refreshed_at,omitempty"`
}

type UserAccountListOptions struct {
	Limit  int
	Offset int
	Search string
}

// CreateAccount creates a new account for a user
func (db *DB) CreateAccount(userID int64, name, remark, email, mobile, password, proxyID string) (*UserAccount, error) {
	now := time.Now().Unix()
	result, err := db.Exec(`
		INSERT INTO user_accounts (user_id, name, remark, email, mobile, password, proxy_id, enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, 1, ?)
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
		Enabled:   true,
		CreatedAt: now,
	}, nil
}

// GetAccountsByUserID gets all accounts for a user
func (db *DB) GetAccountsByUserID(userID int64) ([]*UserAccount, error) {
	rows, err := db.Query(`
		SELECT id, user_id, name, remark, email, mobile, password, proxy_id, enabled, created_at, last_refreshed_at
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
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Remark, &a.Email, &a.Mobile, &a.Password, &a.ProxyID, &a.Enabled, &a.CreatedAt, &a.LastRefreshedAt); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accounts: %w", err)
	}

	return accounts, nil
}

func (db *DB) ListAccountsByUserID(userID int64, opts UserAccountListOptions) ([]*UserAccount, int, error) {
	if opts.Limit <= 0 {
		opts.Limit = 10
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}

	where, args := buildUserAccountListWhere(userID, opts.Search)

	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM user_accounts"+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count user accounts: %w", err)
	}

	queryArgs := append([]interface{}{}, args...)
	queryArgs = append(queryArgs, opts.Limit, opts.Offset)
	rows, err := db.Query(`
		SELECT id, user_id, name, remark, email, mobile, password, proxy_id, enabled, created_at, last_refreshed_at
		FROM user_accounts`+where+`
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query user accounts: %w", err)
	}
	defer rows.Close()

	var accounts []*UserAccount
	for rows.Next() {
		var a UserAccount
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Remark, &a.Email, &a.Mobile, &a.Password, &a.ProxyID, &a.Enabled, &a.CreatedAt, &a.LastRefreshedAt); err != nil {
			return nil, 0, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, &a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate user accounts: %w", err)
	}

	return accounts, total, nil
}

func buildUserAccountListWhere(userID int64, search string) (string, []interface{}) {
	conditions := []string{"user_id = ?"}
	args := []interface{}{userID}

	search = strings.TrimSpace(search)
	if search != "" {
		conditions = append(conditions, "(LOWER(name) LIKE ? OR LOWER(remark) LIKE ? OR LOWER(email) LIKE ? OR LOWER(mobile) LIKE ?)")
		pattern := "%" + strings.ToLower(search) + "%"
		args = append(args, pattern, pattern, pattern, pattern)
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

// GetAccountByID gets an account by ID
func (db *DB) GetAccountByID(id int64) (*UserAccount, error) {
	var a UserAccount
	err := db.QueryRow(`
		SELECT id, user_id, name, remark, email, mobile, password, proxy_id, enabled, created_at, last_refreshed_at
		FROM user_accounts WHERE id = ?
	`, id).Scan(&a.ID, &a.UserID, &a.Name, &a.Remark, &a.Email, &a.Mobile, &a.Password, &a.ProxyID, &a.Enabled, &a.CreatedAt, &a.LastRefreshedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("account not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query account: %w", err)
	}
	return &a, nil
}

func (db *DB) SetAccountEnabled(id, userID int64, enabled bool) error {
	result, err := db.Exec(`
		UPDATE user_accounts
		SET enabled = ?
		WHERE id = ? AND user_id = ?
	`, enabled, id, userID)
	if err != nil {
		return fmt.Errorf("set account enabled: %w", err)
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
		SELECT id, user_id, name, remark, email, mobile, password, proxy_id, enabled, created_at, last_refreshed_at
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
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Remark, &a.Email, &a.Mobile, &a.Password, &a.ProxyID, &a.Enabled, &a.CreatedAt, &a.LastRefreshedAt); err != nil {
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

// ClearAccountRefreshTime clears the last_refreshed_at timestamp for an account.
func (db *DB) ClearAccountRefreshTime(id int64) error {
	_, err := db.Exec(`
		UPDATE user_accounts
		SET last_refreshed_at = NULL
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("clear refresh time: %w", err)
	}
	return nil
}
