package middlewares

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

// GatewayRequestLogger logs incoming requests and their responses.
// Includes tenant slug and staff ID when available in context.
func GatewayRequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			start := time.Now()

			// Process request
			err := next(ctx)

			// Log after response
			duration := time.Since(start)
			req := ctx.Request()
			res := ctx.Response()

			attrs := []any{
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Int("status", res.Status),
				slog.Duration("duration", duration),
				slog.String("remote_ip", ctx.RealIP()),
			}

			// Add tenant slug if present
			if slug, ok := ctx.Get("tenant_slug").(string); ok {
				attrs = append(attrs, slog.String("tenant", slug))
			}

			// Add staff ID if present
			if staffID, ok := ctx.Get("staff_id").(string); ok {
				attrs = append(attrs, slog.String("staff_id", staffID))
			}

			// Add error if present
			if err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
				slog.Error("request failed", attrs...)
				return err
			}

			// Log based on status code
			switch {
			case res.Status >= 500:
				slog.Error("server error", attrs...)
			case res.Status >= 400:
				slog.Warn("client error", attrs...)
			default:
				slog.Info("request completed", attrs...)
			}

			return nil
		}
	}
}
