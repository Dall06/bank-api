package router

import (
	"bank-api/opt/middlewares"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// SetAppUse configures global middleware for the application
// Note: CORS is NOT set here because the gateway handles CORS.
// Adding CORS here would cause duplicate headers when requests are proxied.
//
// IMPORTANT: IDsMiddleware and LoggerMiddleware are registered GLOBALLY on
// `app` rather than on a local `/api/v1` group. The previous implementation
// created a local group variable that was discarded, while each service's
// SetupXxxRoutes function created its OWN `e.Group("/api/v1")` for routes,
// leaving the routes without any of this middleware. Making these global
// ensures every request gets an ID and is logged regardless of which group
// the routes end up on.
func SetAppUse(app *echo.Echo) {
	// Global middleware (applied to ALL routes, including /api/v1/* added later)
	app.Use(
		middleware.Recover(),
		middleware.SecureWithConfig(middleware.DefaultSecureConfig),
		middlewares.IDsMiddleware(),
		middlewares.LoggerMiddleware(),
	)
}
