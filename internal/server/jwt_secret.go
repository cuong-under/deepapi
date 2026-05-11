package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"ds2api/internal/config"
)

func loadOrCreateJWTSecret() (string, error) {
	if jwtSecret := os.Getenv("DS2API_JWT_SECRET"); jwtSecret != "" {
		return jwtSecret, nil
	}

	secretPath := filepath.Join(filepath.Dir(config.ConfigPath()), ".jwt_secret")
	if data, err := os.ReadFile(secretPath); err == nil {
		config.Logger.Info("[multi-user] loaded JWT secret from file", "path", secretPath)
		return string(data), nil
	}

	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("generate JWT secret: %w", err)
	}
	jwtSecret := base64.StdEncoding.EncodeToString(randomBytes)

	if err := os.WriteFile(secretPath, []byte(jwtSecret), 0600); err != nil {
		return "", fmt.Errorf("save JWT secret: %w", err)
	}

	config.Logger.Info("[multi-user] generated new JWT secret and saved to file", "path", secretPath)
	return jwtSecret, nil
}
