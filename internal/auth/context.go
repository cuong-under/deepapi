package auth

import (
	"context"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	userIDKey   contextKey = "user_id"
	usernameKey contextKey = "username"
	roleKey     contextKey = "role"
)

// UserContext holds user information in request context
type UserContext struct {
	UserID   int64
	Username string
	Role     string
}

// SetUserContext sets user information in context
func SetUserContext(ctx context.Context, userID int64, username, role string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, usernameKey, username)
	ctx = context.WithValue(ctx, roleKey, role)
	return ctx
}

// GetUserContext retrieves user information from context
func GetUserContext(ctx context.Context) (*UserContext, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	if !ok {
		return nil, false
	}

	username, ok := ctx.Value(usernameKey).(string)
	if !ok {
		return nil, false
	}

	role, ok := ctx.Value(roleKey).(string)
	if !ok {
		return nil, false
	}

	return &UserContext{
		UserID:   userID,
		Username: username,
		Role:     role,
	}, true
}

// GetUserID retrieves user ID from context
func GetUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDKey).(int64)
	return userID, ok
}

// GetUsername retrieves username from context
func GetUsername(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(usernameKey).(string)
	return username, ok
}

// GetRole retrieves user role from context
func GetRole(ctx context.Context) (string, bool) {
	role, ok := ctx.Value(roleKey).(string)
	return role, ok
}

// IsAdmin checks if user is admin
func IsAdmin(ctx context.Context) bool {
	role, ok := GetRole(ctx)
	return ok && role == "admin"
}

// IsUser checks if user is regular user
func IsUser(ctx context.Context) bool {
	role, ok := GetRole(ctx)
	return ok && role == "user"
}
