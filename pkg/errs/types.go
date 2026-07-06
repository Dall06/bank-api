package errs

import "errors"

// Exported sentinel errors
var (
	ErrNotFound          = errors.New("not found")
	ErrInternal          = errors.New("internal error")
	ErrValue             = errors.New("value error")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrNotValid          = errors.New("not valid")
	ErrConflict          = errors.New("conflict")
	ErrServiceUnavail    = errors.New("service unavailable")
	ErrPlanLimitExceeded = errors.New("plan limit exceeded")
)
