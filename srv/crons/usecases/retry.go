package usecases

import (
	"context"
	"log/slog"
	"strings"

	"bank-api/srv/crons/ports"
	"bank-api/srv/crons/domain"
)

// mapProviderStatus convierte el estado del proveedor al estado interno.
// APPROVED (proveedor) → EXECUTED (nuestro sistema)
// Cualquier otro valor → REJECTED
func mapProviderStatus(providerStatus string) string {
	if providerStatus == "APPROVED" {
		return "EXECUTED"
	}
	return "REJECTED"
}

type cronUseCase struct {
	txRepo       ports.TransactionRepository
	providerRepo ports.ProviderRepository
}

func NewCronUseCase(txRepo ports.TransactionRepository, providerRepo ports.ProviderRepository) ports.CronUseCase {
	return &cronUseCase{
		txRepo:       txRepo,
		providerRepo: providerRepo,
	}
}

func (uc *cronUseCase) RetryPendingTransactions(ctx context.Context) error {
	// 1. Buscar transacciones en estado PENDING (limitamos a 100 por iteración para no saturar memoria)
	filter := &domain.Transaction{Status: "PENDING"}
	txs, _, err := uc.txRepo.FindAll(ctx, filter, 100, 0)
	if err != nil {
		slog.Error("error obteniendo transacciones pendientes", slog.String("error", err.Error()))
		return err
	}

	if len(txs) == 0 {
		return nil
	}

	slog.Info("iniciando reintento de transacciones", slog.Int("count", len(txs)))

	// 2. Procesar cada transacción
	for _, tx := range txs {
		// Construir request al proveedor
		req := domain.ProviderRequest{
			AccountID: tx.AccountID,
			Amount:    tx.Amount,
			Currency:  "MXN",
		}

		// Llamar al proveedor
		resp, err := uc.providerRepo.Post(ctx, req)
		
		// Determinar nuevo estatus
		newStatus := "FAILED" // Por defecto, si hay error no controlado
		
		if err != nil {
			// Si el error es un ValueError de los que arroja el provider cuando rechaza
			if strings.Contains(err.Error(), "provider") || strings.Contains(err.Error(), "rejected") {
				newStatus = "REJECTED"
			}
			
			if !strings.Contains(err.Error(), "provider") && !strings.Contains(err.Error(), "rejected") {
				// Si fue un error de red o timeout, lo dejamos PENDING para el próximo ciclo
				slog.Warn("error temporal con proveedor, se reintentará luego", slog.String("tx_id", tx.ID), slog.String("error", err.Error()))
				continue
			}
		}

		if err == nil && resp != nil {
			newStatus = mapProviderStatus(resp.Status)
		}

		// 3. Actualizar transacción
		tx.Status = newStatus
		if err := uc.txRepo.Update(ctx, tx); err != nil {
			slog.Error("error actualizando transacción", slog.String("tx_id", tx.ID), slog.String("error", err.Error()))
			continue
		}
		
		slog.Info("transacción procesada por cron", slog.String("tx_id", tx.ID), slog.String("new_status", newStatus))
	}

	return nil
}
