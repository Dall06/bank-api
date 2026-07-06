package ports

import (
	"context"

	"github.com/uptrace/bun"

	domain "bank-api/srv/transactions/domain"
)

// TransactionRepository gestiona la persistencia en PostgreSQL (Puerto de Salida)
type TransactionRepository interface {
	Save(ctx context.Context, tx *domain.Transaction) error
	Update(ctx context.Context, tx *domain.Transaction) error
	FindAll(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error)
}

// DBConnector gestiona la conexión a la base de datos
type DBConnector interface {
	GetDB() *bun.DB
	Close() error
}

// UsersRepository gestiona la validación de usuarios
type UsersRepository interface {
	ValidateUser(ctx context.Context, accountID string) error
}

// ProviderRepository gestiona la llamada HTTP al proveedor externo (Puerto de Salida)
type ProviderRepository interface {
	Post(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error)
}
