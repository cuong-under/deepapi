package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	userauth "ds2api/internal/auth"
	"ds2api/internal/config"
	"ds2api/internal/database"
	adminupdate "ds2api/internal/httpapi/admin/update"
	useradmin "ds2api/internal/httpapi/admin/users"
	authhandler "ds2api/internal/httpapi/auth"
	useraccounts "ds2api/internal/httpapi/user/accounts"
	userkeys "ds2api/internal/httpapi/user/keys"
	userprofile "ds2api/internal/httpapi/user/profile"
	"ds2api/internal/middleware"
)

// RegisterMultiUserRoutes registers multi-user authentication routes
func RegisterMultiUserRoutes(r chi.Router, app *App) error {
	// Check if multi-user mode is enabled
	multiUserEnabled := os.Getenv("DS2API_MULTI_USER") == "true"
	if !multiUserEnabled {
		config.Logger.Info("[multi-user] disabled (set DS2API_MULTI_USER=true to enable)")
		return nil
	}

	config.Logger.Info("[multi-user] enabled, initializing database")

	// Initialize database
	dbPath := filepath.Join(filepath.Dir(config.ConfigPath()), "ds2api.db")
	db, err := database.Open(dbPath)
	if err != nil {
		return err
	}

	config.Logger.Info("[multi-user] database initialized", "path", dbPath)

	// Check if we need to create default admin user
	count, err := db.CountUsers()
	if err != nil {
		return err
	}

	if count == 0 {
		config.Logger.Info("[multi-user] creating default admin user")
		adminPassword := os.Getenv("DS2API_ADMIN_PASSWORD")
		if adminPassword == "" {
			// Generate random password
			randomBytes := make([]byte, 16)
			rand.Read(randomBytes)
			adminPassword = base64.URLEncoding.EncodeToString(randomBytes)

			config.Logger.Warn("╔════════════════════════════════════════════════════════════════╗")
			config.Logger.Warn("║  IMPORTANT: Generated random admin password                   ║")
			config.Logger.Warn("║  Username: admin                                               ║")
			config.Logger.Warn(fmt.Sprintf("║  Password: %-52s ║", adminPassword))
			config.Logger.Warn("║  Please save this password! It will not be shown again.        ║")
			config.Logger.Warn("║  Set DS2API_ADMIN_PASSWORD env to use custom password.        ║")
			config.Logger.Warn("╚════════════════════════════════════════════════════════════════╝")
		}

		_, err = db.CreateUser("admin", "admin@ds2api.local", adminPassword, "admin")
		if err != nil {
			return err
		}
		config.Logger.Info("[multi-user] default admin user created (username: admin)")
	}

	jwtSecret, err := loadOrCreateJWTSecret()
	if err != nil {
		config.Logger.Error("[multi-user] failed to load JWT secret", "error", err)
		return err
	}
	jwtManager := userauth.NewJWTManager(jwtSecret)

	// Initialize auth middleware
	authMiddleware := userauth.NewMiddleware(jwtManager, db)

	// Initialize handlers
	authHandler := authhandler.NewHandler(db, jwtManager)
	accountsHandler := useraccounts.NewHandler(db, app.DS)
	keysHandler := userkeys.NewHandler(db)
	profileHandler := userprofile.NewHandler(db)
	usersHandler := useradmin.NewHandler(db)
	var updateHandler *adminupdate.GitHandler
	if app.GitManager != nil {
		updateHandler = adminupdate.NewGitHandler(app.GitManager)
	}

	// Get analytics handler from app (already initialized in router.go)
	// We'll register it with OptionalAuth so both admin and regular users can access it
	// (filtered by user_id in the handler itself)

	// Initialize rate limiters
	// Login/Register: 5 requests per second, burst 10
	authLimiter := middleware.NewRateLimiter(5, 10)

	// Register public auth routes (no authentication required)
	r.Route("/api/auth", func(ar chi.Router) {
		// Apply rate limiting to login/register
		ar.With(authLimiter.Middleware).Post("/login", authHandler.Login)
		ar.With(authLimiter.Middleware).Post("/register", authHandler.Register)

		// Protected routes (require authentication)
		ar.Group(func(protected chi.Router) {
			protected.Use(authMiddleware.Authenticate)
			protected.Post("/logout", authHandler.Logout)
			protected.Get("/me", authHandler.Me)
			protected.Post("/refresh", authHandler.RefreshToken)
		})
	})

	// Register protected user routes (authentication required)
	r.Route("/api/user", func(ur chi.Router) {
		ur.Use(authMiddleware.Authenticate)
		ur.Use(authMiddleware.RequireUser)
		accountsHandler.RegisterRoutes(ur)
		keysHandler.RegisterRoutes(ur)
		userprofile.RegisterRoutes(ur, profileHandler)
	})

	// Register admin-only routes (admin authentication required)
	r.Route("/api/admin", func(ar chi.Router) {
		ar.Use(authMiddleware.Authenticate)
		ar.Use(authMiddleware.RequireAdmin)
		useradmin.RegisterRoutes(ar, usersHandler)
		adminupdate.RegisterRoutes(ar, updateHandler)
	})

	config.Logger.Info("[multi-user] routes registered successfully")
	return nil
}
