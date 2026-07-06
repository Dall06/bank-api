package ports

import (
	"context"
	
	"bank-api/srv/crons/domain"
)

// TransactionRepository provee métodos para buscar y actualizar transacciones
type TransactionRepository interface {
	FindAll(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error)
	Update(ctx context.Context, tx *domain.Transaction) error
}

// ProviderRepository provee el método para llamar al proveedor externo
type ProviderRepository interface {
	Post(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error)
}
