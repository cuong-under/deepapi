package errors

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// AppError represents a structured application error
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	Status  int    `json:"-"`
}

func (e *AppError) Error() string {
	return e.Message
}

// Common errors
var (
	// Authentication errors
	ErrInvalidCredentials = &AppError{
		Code:    "INVALID_CREDENTIALS",
		Message: "Invalid username or password",
		Status:  http.StatusUnauthorized,
	}

	ErrUnauthorized = &AppError{
		Code:    "UNAUTHORIZED",
		Message: "Authentication required",
		Status:  http.StatusUnauthorized,
	}

	ErrForbidden = &AppError{
		Code:    "FORBIDDEN",
		Message: "Access denied",
		Status:  http.StatusForbidden,
	}

	ErrSessionExpired = &AppError{
		Code:    "SESSION_EXPIRED",
		Message: "Session expired, please login again",
		Status:  http.StatusUnauthorized,
	}

	// User errors
	ErrUserNotFound = &AppError{
		Code:    "USER_NOT_FOUND",
		Message: "User not found",
		Status:  http.StatusNotFound,
	}

	ErrDuplicateUsername = &AppError{
		Code:    "DUPLICATE_USERNAME",
		Message: "Username already exists",
		Status:  http.StatusConflict,
	}

	ErrDuplicateEmail = &AppError{
		Code:    "DUPLICATE_EMAIL",
		Message: "Email already exists",
		Status:  http.StatusConflict,
	}

	// Validation errors
	ErrInvalidInput = &AppError{
		Code:    "INVALID_INPUT",
		Message: "Invalid input data",
		Status:  http.StatusBadRequest,
	}

	ErrInvalidUsername = &AppError{
		Code:    "INVALID_USERNAME",
		Message: "Invalid username format",
		Status:  http.StatusBadRequest,
	}

	ErrInvalidEmail = &AppError{
		Code:    "INVALID_EMAIL",
		Message: "Invalid email format",
		Status:  http.StatusBadRequest,
	}

	ErrWeakPassword = &AppError{
		Code:    "WEAK_PASSWORD",
		Message: "Password does not meet security requirements",
		Status:  http.StatusBadRequest,
	}

	// Resource errors
	ErrResourceNotFound = &AppError{
		Code:    "RESOURCE_NOT_FOUND",
		Message: "Resource not found",
		Status:  http.StatusNotFound,
	}

	ErrAccessDenied = &AppError{
		Code:    "ACCESS_DENIED",
		Message: "You don't have permission to access this resource",
		Status:  http.StatusForbidden,
	}

	// Server errors
	ErrInternalServer = &AppError{
		Code:    "INTERNAL_ERROR",
		Message: "Internal server error",
		Status:  http.StatusInternalServerError,
	}

	ErrDatabaseError = &AppError{
		Code:    "DATABASE_ERROR",
		Message: "Database operation failed",
		Status:  http.StatusInternalServerError,
	}
)

// NewAppError creates a new AppError with custom message
func NewAppError(code, message string, status int) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

// WithDetails adds details to an error
func (e *AppError) WithDetails(details string) *AppError {
	return &AppError{
		Code:    e.Code,
		Message: e.Message,
		Details: details,
		Status:  e.Status,
	}
}

// RespondError writes an error response
func RespondError(w http.ResponseWriter, err error, logger *slog.Logger) {
	appErr, ok := err.(*AppError)
	if !ok {
		// Unknown error - log full details but return generic message
		if logger != nil {
			logger.Error("unexpected error", "error", err)
		}
		appErr = ErrInternalServer
	} else {
		// Log structured error
		if logger != nil {
			logger.Error("request failed",
				"code", appErr.Code,
				"message", appErr.Message,
				"details", appErr.Details,
				"status", appErr.Status,
			)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.Status)
	json.NewEncoder(w).Encode(appErr)
}

// RespondSuccess writes a success response
func RespondSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(data)
}

// RespondCreated writes a created response
func RespondCreated(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(data)
}
