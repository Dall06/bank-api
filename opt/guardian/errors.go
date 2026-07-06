package guardian

import "errors"

var (
	// ErrTokenRevoked is returned when a token has been revoked
	ErrTokenRevoked = errors.New("token has been revoked")

	// ErrInvalidRefreshToken is returned when the refresh token is invalid
	ErrInvalidRefreshToken = errors.New("invalid refresh token")

	// ErrPermissionDenied is returned when the user lacks the required permission
	ErrPermissionDenied = errors.New("permission denied")
)
