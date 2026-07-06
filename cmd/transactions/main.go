package main

import (
	"context"
	"encoding/hex"
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
	"bank-api/opt/redis"
	"bank-api/opt/router"
	"bank-api/pkg/cache"
	"bank-api/pkg/crypto"
	"bank-api/pkg/errs"
	"bank-api/pkg/logs"
	"bank-api/pkg/publisher"
	"bank-api/pkg/sigil"
	"bank-api/srv/transactions/domain"
	"bank-api/srv/transactions/handlers"
	"bank-api/srv/transactions/repositories"
	"bank-api/srv/transactions/usecases"

	"github.com/labstack/echo/v4"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("error cargando configuración", slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := cfg.Validate("DatabaseURL", "JWTSecret", "SigilSecret", "EncryptionKey", "HMACKey", "RedisURL", "KafkaBrokers", "KafkaTopic"); err != nil {
		slog.Error("configuración incompleta", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logs.Setup(cfg.LogLevel, "transactions", cfg.Env)
	slog.Info("iniciando servicio de transacciones...", slog.Int("port", cfg.Port))

	// Base de datos
	dbConn, err := database.Connect(cfg.DatabaseURL, 30)
	if err != nil {
		slog.Error("error conectando a la base de datos", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbConn.Close()

	// Base de datos de cifrado
	encKeyBytes, err := hex.DecodeString(cfg.EncryptionKey)
	if err != nil {
		slog.Error("error decodificando ENCRYPTION_KEY", slog.String("error", err.Error()))
		os.Exit(1)
	}
	hmacKeyBytes, err := hex.DecodeString(cfg.HMACKey)
	if err != nil {
		slog.Error("error decodificando HMAC_KEY", slog.String("error", err.Error()))
		os.Exit(1)
	}

	encryptor, err := crypto.NewFieldEncryptor(encKeyBytes, hmacKeyBytes)
	if err != nil {
		slog.Error("error inicializando cifrador", slog.String("error", err.Error()))
		os.Exit(1)
	}
	crypto.SetGlobalEncryptor(encryptor)

	// Automigración de la tabla de transacciones
	if err = database.AutoMigrate[domain.Transaction](dbConn); err != nil {
		slog.Error("error en automigración de transacciones", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Inicializar Redis con 5 segundos de timeout
	redisClient := redis.NewRedisClient(cfg.RedisURL, 5)
	if err = redisClient.Connect(); err != nil {
		slog.Error("error conectando a Redis", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Inicializar publicador de Kafka
	brokers := strings.Split(cfg.KafkaBrokers, ",")
	kafkaPub := publisher.NewKafkaPublisher(brokers, cfg.KafkaTopic)

	// Sigil: firma S2S salientes (a users) y verifica entrantes (gateway y crons)
	sigilSigner := sigil.NewSigner(sigil.DefaultConfig(cfg.SigilSecret, "transactions"))
	allowedServices := strings.Split(cfg.AllowedServices, ",")
	sigilVerifier := sigil.NewVerifier(sigil.DefaultConfig(cfg.SigilSecret, "transactions"), allowedServices)

	// Repositorios
	txRepo := repositories.NewTransactionRepository(dbConn)
	providerRepo := repositories.NewProviderRepository(cfg.ProviderExecuteURL, 10*time.Second)
	usersRepo := repositories.NewUsersRepository(cfg.UsersInternalURL, sigilSigner)

	// Casos de uso y handler
	cacheMgr := cache.NewManager(redisClient)
	purchaseUC := usecases.NewPurchaseUseCase(cfg.AuthRequired, txRepo, providerRepo, usersRepo, cacheMgr, kafkaPub)
	getUC := usecases.NewGetUseCase(txRepo, providerRepo, usersRepo, redisClient, kafkaPub)
	updateUC := usecases.NewUpdateUseCase(txRepo, providerRepo, usersRepo, redisClient, kafkaPub)
	txHandler := handlers.NewTransactionHandler(purchaseUC, getUC, updateUC)

	// Echo
	e := echo.New()
	e.HTTPErrorHandler = errs.ErrorHandler
	e.HideBanner = true
	e.HidePort = true

	router.SetAppUse(e)
	e.Use(middlewares.NewSigilVerifier(sigilVerifier))
	e.Use(middlewares.MockHeaderMiddleware)

	// Rutas públicas (JWT opcional según AUTH_REQUIRED)
	api := e.Group("/api/v1")
	if cfg.AuthRequired {
		slog.Info("JWT habilitado: AUTH_REQUIRED=true")
		api.POST("/transactions", txHandler.Create, middlewares.NewJWTAuth(cfg.JWTSecret))
		api.GET("/transactions", txHandler.Get, middlewares.NewJWTAuth(cfg.JWTSecret))
	}
	if !cfg.AuthRequired {
		slog.Warn("JWT deshabilitado: AUTH_REQUIRED=false (modo evaluación)")
		api.POST("/transactions", txHandler.Create)
		api.GET("/transactions", txHandler.Get)
	}

	// Rutas internas (solo red interna, protegidas por Sigil)
	e.GET("/internal/transactions", txHandler.GetInternal)
	e.PATCH("/internal/transactions/:id/status", txHandler.UpdateStatusInternal)

	// Salud
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": "transactions"})
	})

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			slog.Error("error en servidor de transacciones", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("deteniendo servicio de transacciones...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		slog.Error("error al apagar servicio de transacciones", slog.String("error", err.Error()))
	}

	// Apagar publicador de Kafka ordenadamente
	slog.Info("deteniendo publicador de Kafka...")
	kafkaPub.Close()

	// Apagar conexión a Redis ordenadamente
	slog.Info("desconectando de Redis...")
	if err := redisClient.Disconnect(); err != nil {
		slog.Error("error al desconectar de Redis", slog.String("error", err.Error()))
	}

	slog.Info("servicio de transacciones detenido")
}
