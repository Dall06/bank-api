package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bank-api/env/config"
	"bank-api/pkg/logs"
	"bank-api/pkg/sigil"
	"bank-api/srv/crons/repositories"
	cronUsecases "bank-api/srv/crons/usecases"

	"github.com/robfig/cron/v3"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("error cargando configuración", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Crons no necesita DB ni JWT directamente, solo Sigil para llamadas S2S
	if err := cfg.Validate("SigilSecret", "TransactionsURL", "ProviderURL"); err != nil {
		slog.Error("configuración incompleta", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logs.Setup(cfg.LogLevel, "crons", cfg.Env)
	slog.Info("iniciando microservicio de crons...")

	// Sigil: firma las llamadas S2S salientes a Transactions
	sigilSigner := sigil.NewSigner(sigil.DefaultConfig(cfg.SigilSecret, "crons"))

	// Adaptadores HTTP (S2S directo, sin pasar por Gateway)
	txRepo := repositories.NewTransactionRepository(cfg.TransactionsInternalURL, sigilSigner)
	providerRepo := repositories.NewProviderRepository(cfg.ProviderExecuteURL, 15*time.Second)

	// Caso de uso
	uc := cronUsecases.NewCronUseCase(txRepo, providerRepo)

	// Planificador
	c := cron.New()

	_, err = c.AddFunc("@every 1m", func() {
		slog.Info("ejecutando job: RetryPendingTransactions")
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		if retryErr := uc.RetryPendingTransactions(ctx); retryErr != nil {
			slog.Error("fallo en job de reintentos", slog.String("error", retryErr.Error()))
		}
	})
	if err != nil {
		slog.Error("error registrando cron job", slog.String("error", err.Error()))
		os.Exit(1)
	}

	c.Start()
	slog.Info("crons en ejecución, esperando señales...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("deteniendo crons...")
	ctx := c.Stop()
	<-ctx.Done()
	slog.Info("crons detenido limpiamente")
}
