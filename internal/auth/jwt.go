package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JWTClaims represents the JWT payload
type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
}

// JWTManager handles JWT token operations
type JWTManager struct {
	secret []byte
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secret string) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
	}
}

// GenerateToken generates a JWT token for a user
func (m *JWTManager) GenerateToken(userID int64, username, role string, expiresIn time.Duration) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		Exp:      now.Add(expiresIn).Unix(),
		Iat:      now.Unix(),
	}

	// Create header
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}

	// Base64 encode
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Create signature
	message := headerB64 + "." + claimsB64
	signature := m.sign(message)
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	// Combine all parts
	token := message + "." + signatureB64

	return token, nil
}

// ValidateToken validates a JWT token and returns the claims
func (m *JWTManager) ValidateToken(token string) (*JWTClaims, error) {
	// Split token into parts
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	headerB64 := parts[0]
	claimsB64 := parts[1]
	signatureB64 := parts[2]

	// Verify signature
	message := headerB64 + "." + claimsB64
	expectedSignature := m.sign(message)
	expectedSignatureB64 := base64.RawURLEncoding.EncodeToString(expectedSignature)

	if signatureB64 != expectedSignatureB64 {
		return nil, fmt.Errorf("invalid signature")
	}

	// Decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(claimsB64)
	if err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal claims: %w", err)
	}

	// Check expiration
	if time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}

	return &claims, nil
}

// sign creates HMAC-SHA256 signature
func (m *JWTManager) sign(message string) []byte {
	h := hmac.New(sha256.New, m.secret)
	h.Write([]byte(message))
	return h.Sum(nil)
}

// RefreshToken generates a new token with extended expiration
func (m *JWTManager) RefreshToken(token string, expiresIn time.Duration) (string, error) {
	claims, err := m.ValidateToken(token)
	if err != nil {
		return "", fmt.Errorf("validate token: %w", err)
	}

	return m.GenerateToken(claims.UserID, claims.Username, claims.Role, expiresIn)
}
