package ports

import (
	domain "bank-api/srv/transactions/domain"
	"context"
)

// TransactionUsecase define la interfaz que expone las operaciones de negocio (Puerto de Entrada)
type PurchaseUseCase interface {
	Purchase(ctx context.Context, req domain.CreateTransactionRequest) (*domain.TransactionResponse, error)
}

type GetUseCase interface {
	GetAll(ctx context.Context, req domain.GetTransactionsRequest) (*domain.GetTransactionsResponse, error)
}

type UpdateUseCase interface {
	UpdateStatus(ctx context.Context, txID, status string) error
}
