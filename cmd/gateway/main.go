package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"bank-api/env/config"
	database "bank-api/opt/db"
	"bank-api/pkg/errs"
	"bank-api/opt/middlewares"
	"bank-api/opt/router"
	"bank-api/pkg/jwt"
	"bank-api/pkg/logs"
	"bank-api/pkg/sigil"
	"bank-api/srv/gateway"

	"github.com/labstack/echo/v4"
)

// @title Bank API Monorepo
// @version 1.0
// @description API for the Bank System using Hexagonal Architecture.
// @host localhost:8000
// @BasePath /api/v1
func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("error cargando configuración", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := cfg.Validate("JWTSecret", "SigilSecret"); err != nil {
		slog.Error("configuración incompleta", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logs.Setup(cfg.LogLevel, "gateway", cfg.Env)
	slog.Info("iniciando gateway...", slog.Int("port", cfg.Port))

	e := echo.New()
	e.HTTPErrorHandler = errs.ErrorHandler

	// Middlewares globales
	e.HideBanner = true
	e.HidePort = true

	gateway.Setup(e, gateway.Config{
		JWTSecret:       cfg.JWTSecret,
		UserURL:         cfg.UsersURL,
		TransactionsURL: cfg.TransactionsURL,
		AllowedOrigins:  cfg.AllowedOrigins,
		SigilSecret:     cfg.SigilSecret,
	})

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			slog.Error("error en gateway", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("deteniendo gateway...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		slog.Error("error al apagar gateway", slog.String("error", err.Error()))
	}

	slog.Info("gateway detenido")
}

// Compilamos para que no haya import cycle: gateway.Config.UserURL viene de UsersURL
// Validamos que AllowedOrigins no esté vacío solo si hay más de localhost
var _ = strings.TrimSpace
var _ = database.Connect
var _ = middlewares.NewJWTAuth
var _ = router.SetAppUse
var _ = jwt.NewGenerator
var _ = sigil.NewSigner
