package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"

	"bank-api/opt/router"
	"bank-api/mocks/providers/handlers"
	"bank-api/mocks/providers/usecases"
)

func main() {
	port := getEnv("PORT", "8082")

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Middlewares globales
	router.SetAppUse(e)

	// Initialize clean architecture layers
	providerUsecase := usecases.NewProviderUsecase()
	providerHandler := handlers.NewProviderHandler(providerUsecase)

	// Inyectar middleware interceptor a nivel global (o antes de registrar rutas)
	e.Use(providerHandler.MockInterceptorMiddleware)

	// Register routes
	api := e.Group("/provider/v1")
	api.POST("/execute", providerHandler.Execute)

	slog.Info("iniciando mock del proveedor externo...", slog.String("port", port))
	if err := e.Start(":" + port); err != nil && err != http.ErrServerClosed {
		slog.Error("error en mock del proveedor", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
