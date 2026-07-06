package middlewares

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func IDsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			traceID := ctx.Request().Header.Get("x-trace-id")
			if traceID == "" {
				slog.Warn("missing trace-id header, generating new one")
				traceUUID, err := uuid.NewUUID()
				if err != nil {
					slog.Error("failed to generate trace ID", "error", err)
					return ctx.String(http.StatusInternalServerError, "internal error")
				}
				traceID = traceUUID.String()
			}

			reqID, err := uuid.NewUUID()
			if err != nil {
				slog.Error("failed to generate request ID", "error", err)
				return ctx.String(http.StatusInternalServerError, "internal error")
			}
			reqIDStr := reqID.String()

			ctx.Response().Header().Set("x-request-id", reqIDStr)
			ctx.Response().Header().Set("x-trace-id", traceID)

			ctx.Set("request-id", reqIDStr)
			ctx.Set("trace-id", traceID)

			return next(ctx)
		}
	}
}
