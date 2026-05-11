package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserListOptions struct {
	Limit  int
	Offset int
	Search string
	Role   string
}

type UserWithCounts struct {
	*User
	AccountsCount int
	KeysCount     int
	SessionsCount int
}

// CreateUser creates a new user
func (db *DB) CreateUser(username, email, password, role string) (*User, error) {
	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().Unix()
	result, err := db.Exec(`
		INSERT INTO users (username, email, password_hash, role, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, username, email, string(hash), role, now, now)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get user id: %w", err)
	}

	return &User{
		ID:           id,
		Username:     username,
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
		CreatedAt:    time.Unix(now, 0),
		UpdatedAt:    time.Unix(now, 0),
	}, nil
}

// GetUserByID gets a user by ID
func (db *DB) GetUserByID(id int64) (*User, error) {
	var u User
	var createdAt, updatedAt int64

	err := db.QueryRow(`
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}

	u.CreatedAt = time.Unix(createdAt, 0)
	u.UpdatedAt = time.Unix(updatedAt, 0)
	return &u, nil
}

// GetUserByUsername gets a user by username
func (db *DB) GetUserByUsername(username string) (*User, error) {
	var u User
	var createdAt, updatedAt int64

	err := db.QueryRow(`
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users WHERE username = ?
	`, username).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}

	u.CreatedAt = time.Unix(createdAt, 0)
	u.UpdatedAt = time.Unix(updatedAt, 0)
	return &u, nil
}

// GetUserByEmail gets a user by email
func (db *DB) GetUserByEmail(email string) (*User, error) {
	var u User
	var createdAt, updatedAt int64

	err := db.QueryRow(`
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users WHERE email = ?
	`, email).Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}

	u.CreatedAt = time.Unix(createdAt, 0)
	u.UpdatedAt = time.Unix(updatedAt, 0)
	return &u, nil
}

// ListUsers lists all users with pagination
func (db *DB) ListUsers(limit, offset int) ([]*User, int, error) {
	// Get total count
	var total int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	// Get users
	rows, err := db.Query(`
		SELECT id, username, email, password_hash, role, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var u User
		var createdAt, updatedAt int64
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role, &createdAt, &updatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		u.CreatedAt = time.Unix(createdAt, 0)
		u.UpdatedAt = time.Unix(updatedAt, 0)
		users = append(users, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate users: %w", err)
	}

	return users, total, nil
}

func (db *DB) ListUsersWithCounts(opts UserListOptions) ([]*UserWithCounts, int, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}

	where, args := buildUserListWhere(opts)

	var total int
	countQuery := "SELECT COUNT(*) FROM users u" + where
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count filtered users: %w", err)
	}

	queryArgs := append([]interface{}{}, args...)
	queryArgs = append(queryArgs, opts.Limit, opts.Offset)
	rows, err := db.Query(`
		SELECT
			u.id,
			u.username,
			u.email,
			u.password_hash,
			u.role,
			u.created_at,
			u.updated_at,
			(SELECT COUNT(*) FROM user_accounts ua WHERE ua.user_id = u.id) AS accounts_count,
			(SELECT COUNT(*) FROM user_api_keys uk WHERE uk.user_id = u.id) AS keys_count,
			(SELECT COUNT(*) FROM sessions s WHERE s.user_id = u.id AND s.expires_at > ?) AS sessions_count
		FROM users u`+where+`
		ORDER BY u.created_at DESC
		LIMIT ? OFFSET ?
	`, append([]interface{}{time.Now().Unix()}, queryArgs...)...)
	if err != nil {
		return nil, 0, fmt.Errorf("query users with counts: %w", err)
	}
	defer rows.Close()

	var users []*UserWithCounts
	for rows.Next() {
		var u User
		var createdAt, updatedAt int64
		item := &UserWithCounts{User: &u}
		if err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.Email,
			&u.PasswordHash,
			&u.Role,
			&createdAt,
			&updatedAt,
			&item.AccountsCount,
			&item.KeysCount,
			&item.SessionsCount,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user with counts: %w", err)
		}
		u.CreatedAt = time.Unix(createdAt, 0)
		u.UpdatedAt = time.Unix(updatedAt, 0)
		users = append(users, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate users with counts: %w", err)
	}

	return users, total, nil
}

func buildUserListWhere(opts UserListOptions) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if opts.Role == "admin" || opts.Role == "user" {
		conditions = append(conditions, "u.role = ?")
		args = append(args, opts.Role)
	}

	search := strings.TrimSpace(opts.Search)
	if search != "" {
		conditions = append(conditions, "(LOWER(u.username) LIKE ? OR LOWER(u.email) LIKE ?)")
		pattern := "%" + strings.ToLower(search) + "%"
		args = append(args, pattern, pattern)
	}

	if len(conditions) == 0 {
		return "", args
	}

	return " WHERE " + strings.Join(conditions, " AND "), args
}

// UpdateUser updates user fields
func (db *DB) UpdateUser(id int64, username, email string) error {
	now := time.Now().Unix()
	result, err := db.Exec(`
		UPDATE users SET username = ?, email = ?, updated_at = ?
		WHERE id = ?
	`, username, email, now, id)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// UpdateUserPassword updates user password
func (db *DB) UpdateUserPassword(id int64, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().Unix()
	result, err := db.Exec(`
		UPDATE users SET password_hash = ?, updated_at = ?
		WHERE id = ?
	`, string(hash), now, id)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// DeleteUser deletes a user (cascades to accounts, keys, sessions)
func (db *DB) DeleteUser(id int64) error {
	result, err := db.Exec("DELETE FROM users WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// VerifyPassword verifies a password against a user's hash
func (db *DB) VerifyPassword(user *User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

// CountUsers returns total number of users
func (db *DB) CountUsers() (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (db *DB) CountAdmins() (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count admins: %w", err)
	}
	return count, nil
}

// UpdateUserRole updates user role
func (db *DB) UpdateUserRole(id int64, role string) error {
	now := time.Now().Unix()
	result, err := db.Exec(`
		UPDATE users SET role = ?, updated_at = ?
		WHERE id = ?
	`, role, now, id)
	if err != nil {
		return fmt.Errorf("update user role: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
