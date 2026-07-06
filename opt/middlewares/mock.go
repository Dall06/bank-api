package middlewares

import (
	"context"

	"bank-api/env/consts"

	"github.com/labstack/echo/v4"
)

func MockHeaderMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		if mockID := ctx.Request().Header.Get(consts.MockIDHeaderKey); mockID != "" {
			c := context.WithValue(ctx.Request().Context(), consts.MockIDContextKey, mockID)
			ctx.SetRequest(ctx.Request().WithContext(c))
		}
		return next(ctx)
	}
}