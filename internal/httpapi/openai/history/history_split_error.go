package history

import (
	"errors"
	"net/http"

	dsclient "ds2api/internal/deepseek/client"
)

func MapError(err error) (int, string) {
	var tooLarge *CurrentInputUploadTooLargeError
	switch {
	case dsclient.IsManagedUnauthorizedError(err):
		return http.StatusUnauthorized, "Account token is invalid. Please re-login the account in admin."
	case dsclient.IsDirectUnauthorizedError(err):
		return http.StatusUnauthorized, "Invalid token. If this should be a DS2API key, add it to config.keys first."
	case errors.As(err, &tooLarge):
		return http.StatusServiceUnavailable, tooLarge.Error()
	default:
		return http.StatusInternalServerError, err.Error()
	}
}
