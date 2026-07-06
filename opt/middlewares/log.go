package middlewares

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
)

func LoggerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			startTime := time.Now()

			traceID := ctx.Request().Header.Get("trace-id")
			if traceID == "" {
				if val := ctx.Get("trace-id"); val != nil {
					traceID = val.(string)
				}
			}

			requestID := ctx.Request().Header.Get("request-id")
			if requestID == "" {
				if val := ctx.Get("request-id"); val != nil {
					requestID = val.(string)
				}
			}

			slog.Info("http request started",
				slog.String("method", ctx.Request().Method),
				slog.String("path", ctx.Path()),
				slog.String("remote_ip", ctx.RealIP()),
				slog.String("trace_id", traceID),
				slog.String("request_id", requestID),
			)

			err := next(ctx)

			attrs := []slog.Attr{
				slog.String("method", ctx.Request().Method),
				slog.String("path", ctx.Path()),
				slog.Int("status", ctx.Response().Status),
				slog.String("trace_id", traceID),
				slog.String("request_id", requestID),
				slog.Duration("duration", time.Since(startTime)),
			}

			if err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
			}

			args := make([]any, len(attrs))
			for i, a := range attrs {
				args[i] = a
			}

			status := ctx.Response().Status
			if status >= 500 {
				slog.Error("http request finished", args...)
			} else if status >= 400 {
				slog.Warn("http request finished", args...)
			} else if status < 400 {
				slog.Info("http request finished", args...)
			}

			return err
		}
	}
}
