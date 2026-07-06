package errs

import (
	"errors"
	"log/slog"
	"runtime"

	"github.com/labstack/echo/v4"
)

// Response is the standard error response format
type Response struct {
	Error string `json:"error"`
}

// ErrorHandler is a custom HTTP error handler for Echo.
// For 5xx responses it ALSO logs the underlying error and unwrap chain plus the
// caller location, so server-side operators can see the real cause even when
// the client body is a generic "internal error" message.
func ErrorHandler(err error, ctx echo.Context) {
	if ctx.Response().Committed {
		return
	}
	status := RestCode(err)
	if status >= 500 {
		// Capture the call site of Handle so we can see who emitted the error.
		_, file, line, _ := runtime.Caller(1)
		// Walk the wrapped errors to get the full chain.
		chain := err.Error()
		for inner := errors.Unwrap(err); inner != nil; inner = errors.Unwrap(inner) {
			chain += " | " + inner.Error()
		}
		slog.Error("internal error in handler",
			slog.String("path", ctx.Request().URL.Path),
			slog.String("method", ctx.Request().Method),
			slog.Int("status", status),
			slog.String("error", err.Error()),
			slog.String("chain", chain),
			slog.String("caller", file),
			slog.Int("line", line),
		)
	}
	msg := err.Error()
	if status >= 500 {
		msg = "internal error"
	}
	ctx.JSON(status, Response{Error: msg})
}

func RestCode(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return 404
	case errors.Is(err, ErrInternal):
		return 500
	case errors.Is(err, ErrValue):
		return 400
	case errors.Is(err, ErrUnauthorized):
		return 401
	case errors.Is(err, ErrForbidden):
		return 403
	case errors.Is(err, ErrNotValid):
		return 424
	case errors.Is(err, ErrConflict):
		return 409
	case errors.Is(err, ErrPlanLimitExceeded):
		return 402 // Payment Required - upgrade plan needed
	case errors.Is(err, ErrServiceUnavail):
		return 503 // Service Unavailable - feature not configured/available
	default:
		return 500 // Internal Server Error for unhandled errors
	}
}
