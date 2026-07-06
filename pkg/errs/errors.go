package errs

import (
	"fmt"
)

// NotFoundError is a generic error for not found errors.
func NotFoundError(msg string, args ...any) error {
	return &customError{
		sentinel: ErrNotFound,
		msg:      fmt.Sprintf(msg, args...),
	}
}

// InternalError is a generic error for internal server errors.
func InternalError(msg string, args ...any) error {
	return &customError{
		sentinel: ErrInternal,
		msg:      fmt.Sprintf(msg, args...),
	}
}

// ValueError is a generic error for invalid values.
func ValueError(msg string, args ...any) error {
	return &customError{
		sentinel: ErrValue,
		msg:      fmt.Sprintf(msg, args...),
	}
}

// UnauthorizedError is a generic error for unauthorized errors.
func UnauthorizedError(msg string, args ...any) error {
	return &customError{
		sentinel: ErrUnauthorized,
		msg:      fmt.Sprintf(msg, args...),
	}
}

// ForbiddenError is a generic error for forbidden errors.
func ForbiddenError(msg string, args ...any) error {
	return &customError{
		sentinel: ErrForbidden,
		msg:      fmt.Sprintf(msg, args...),
	}
}

// NotValidError is a generic error for invalid values.
func NotValidError(msg string, args ...any) error {
	return &customError{
		sentinel: ErrNotValid,
		msg:      fmt.Sprintf(msg, args...),
	}
}

// ConflictError is a generic error for conflict errors (e.g., duplicate resources).
func ConflictError(msg string, args ...any) error {
	return &customError{
		sentinel: ErrConflict,
		msg:      fmt.Sprintf(msg, args...),
	}
}

// ServiceUnavailableError is a generic error for when a feature is not configured
// or temporarily unavailable. Maps to HTTP 503.
func ServiceUnavailableError(msg string, args ...any) error {
	return &customError{
		sentinel: ErrServiceUnavail,
		msg:      fmt.Sprintf(msg, args...),
	}
}
