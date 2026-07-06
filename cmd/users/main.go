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
	"bank-api/opt/middlewares"
	"bank-api/opt/router"
	"bank-api/pkg/errs"
	"bank-api/pkg/jwt"
	"bank-api/pkg/logs"
	"bank-api/pkg/sigil"
	"bank-api/srv/users/domain"
	"bank-api/srv/users/handlers"
	"bank-api/srv/users/repositories"
	"bank-api/srv/users/usecases"

	"github.com/labstack/echo/v4"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("error cargando configuración", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := cfg.Validate("DatabaseURL", "JWTSecret", "SigilSecret"); err != nil {
		slog.Error("configuración incompleta", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logs.Setup(cfg.LogLevel, "users", cfg.Env)
	slog.Info("iniciando servicio de usuarios...", slog.Int("port", cfg.Port))

	// Base de datos
	db, err := database.Connect(cfg.DatabaseURL, 30)
	if err != nil {
		slog.Error("error conectando a la base de datos", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	// Automigración de la tabla de usuarios
	if err = database.AutoMigrate[domain.User](db); err != nil {
		slog.Error("error en automigración de usuarios", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// JWT
	jwtGen := jwt.NewGenerator(jwt.Config{
		Secret:     cfg.JWTSecret,
		Expiration: 24 * time.Hour,
	})

	// Capas arquitectónicas
	userRepo := repositories.NewUserRepository(db)
	getUC := usecases.NewUserUseCase(userRepo, jwtGen)
	signupUC := usecases.NewSignupUseCase(userRepo, jwtGen)
	loginUC := usecases.NewLoginUseCase(userRepo, jwtGen)
	userHandler := handlers.NewUserHandler(getUC, signupUC, loginUC)

	// Echo
	e := echo.New()
	e.HTTPErrorHandler = errs.ErrorHandler
	e.HideBanner = true
	e.HidePort = true

	router.SetAppUse(e)

	// Sigil: acepta peticiones de gateway y de transactions (S2S)
	allowedServices := strings.Split(cfg.AllowedServices, ",")
	sigilVerifier := sigil.NewVerifier(sigil.DefaultConfig(cfg.SigilSecret, "users"), allowedServices)
	e.Use(middlewares.NewSigilVerifier(sigilVerifier))

	// Rutas públicas (proxiadas por Gateway)
	h := userHandler.(*handlers.UserHandler)
	api := e.Group("/api/v1")
	api.POST("/auth/signup", h.Signup)
	api.POST("/auth/login", h.Login)
	api.GET("/users/me", h.GetMe, middlewares.NewJWTAuth(cfg.JWTSecret))

	// Rutas internas (solo red interna, protegidas por Sigil)
	e.GET("/internal/users/:id", h.GetMe)

	// Salud
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": "users"})
	})

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			slog.Error("error en servidor de usuarios", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("deteniendo servicio de usuarios...")
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := e.Shutdown(ctxShutdown); err != nil {
		slog.Error("error al apagar servicio de usuarios", slog.String("error", err.Error()))
	}

	slog.Info("servicio de usuarios detenido")
}
