package middlewares

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

// AuditMiddleware logs all mutating HTTP operations (POST, PUT, PATCH, DELETE)
// with structured fields for security auditing: who, what, when, from where, result.
// Read operations (GET, HEAD, OPTIONS) are skipped to reduce noise.
func AuditMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			method := ctx.Request().Method

			// Only audit mutating operations
			if method != "POST" && method != "PUT" && method != "PATCH" && method != "DELETE" {
				return next(ctx)
			}

			start := time.Now()

			// Execute the handler
			err := next(ctx)

			// Extract identity from context (set by auth middleware)
			userID, _ := ctx.Get("user_id").(string)
			staffID, _ := ctx.Get("staff_id").(string)
			companyID, _ := ctx.Get("company_id").(string)
			role, _ := ctx.Get("role").(string)

			status := ctx.Response().Status
			duration := time.Since(start)
			path := ctx.Request().URL.Path
			ip := ctx.RealIP()

			attrs := []slog.Attr{
				slog.String("method", method),
				slog.String("path", path),
				slog.Int("status", status),
				slog.String("ip", ip),
				slog.String("duration", duration.String()),
			}

			// Only include identity fields if present (public endpoints won't have them)
			if userID != "" {
				attrs = append(attrs, slog.String("user_id", userID))
			}
			if staffID != "" {
				attrs = append(attrs, slog.String("staff_id", staffID))
			}
			if companyID != "" {
				attrs = append(attrs, slog.String("company_id", companyID))
			}
			if role != "" {
				attrs = append(attrs, slog.String("role", role))
			}

			// Log level based on status code
			args := make([]any, len(attrs))
			for i, a := range attrs {
				args[i] = a
			}

			if status >= 500 {
				slog.Error("audit", args...)
			} else if status >= 400 {
				slog.Warn("audit", args...)
			} else if status < 400 {
				slog.Info("audit", args...)
			}

			return err
		}
	}
}
