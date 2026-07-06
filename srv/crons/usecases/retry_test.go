package usecases

import (
	"context"
	"errors"
	"testing"

	"bank-api/srv/crons/ports"
	"bank-api/srv/crons/domain"
	
	"github.com/stretchr/testify/assert"
)

func TestCronUseCase_RetryPendingTransactions(t *testing.T) {
	tests := []struct {
		name         string
		mockFindAll  func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error)
		mockPost     func(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error)
		mockUpdate   func(ctx context.Context, tx *domain.Transaction) error
		wantErr      bool
		wantUpdates  int // Cuántas veces se debe llamar a Update
	}{
		{
			name: "sin transacciones pendientes",
			mockFindAll: func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
				return []*domain.Transaction{}, 0, nil
			},
			mockPost:    nil,
			mockUpdate:  nil,
			wantErr:     false,
			wantUpdates: 0,
		},
		{
			name: "error obteniendo transacciones",
			mockFindAll: func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
				return nil, 0, errors.New("db error")
			},
			mockPost:    nil,
			mockUpdate:  nil,
			wantErr:     true,
			wantUpdates: 0,
		},
		{
			name: "aprueba una transaccion correctamente",
			mockFindAll: func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
				return []*domain.Transaction{
					{ID: "tx-1", Status: "PENDING", AccountID: "acc-1", Amount: 100},
				}, 1, nil
			},
			mockPost: func(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error) {
				return &domain.ProviderResponse{Status: "APPROVED"}, nil
			},
			mockUpdate: func(ctx context.Context, tx *domain.Transaction) error {
				assert.Equal(t, "EXECUTED", tx.Status) // proveedor APPROVED → sistema EXECUTED
				return nil
			},
			wantErr:     false,
			wantUpdates: 1,
		},
		{
			name: "rechaza transaccion por error del provider",
			mockFindAll: func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
				return []*domain.Transaction{
					{ID: "tx-2", Status: "PENDING", AccountID: "acc-1", Amount: 100},
				}, 1, nil
			},
			mockPost: func(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error) {
				return nil, errors.New("provider rejected: fondos insuficientes")
			},
			mockUpdate: func(ctx context.Context, tx *domain.Transaction) error {
				assert.Equal(t, "REJECTED", tx.Status)
				return nil
			},
			wantErr:     false,
			wantUpdates: 1,
		},
		{
			name: "omite actualizacion por timeout temporal",
			mockFindAll: func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
				return []*domain.Transaction{
					{ID: "tx-3", Status: "PENDING"},
				}, 1, nil
			},
			mockPost: func(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error) {
				return nil, errors.New("timeout connecting to server") // No dice provider ni rejected
			},
			mockUpdate: func(ctx context.Context, tx *domain.Transaction) error {
				return nil
			},
			wantErr:     false,
			wantUpdates: 0, // Como es timeout, le da continue y no llama update
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatesCount := 0

			mockDB := &ports.MockTransactionRepository{
				FindAllFunc: tt.mockFindAll,
				UpdateFunc: func(ctx context.Context, tx *domain.Transaction) error {
					updatesCount++
					if tt.mockUpdate != nil {
						return tt.mockUpdate(ctx, tx)
					}
					return nil
				},
			}

			mockProvider := &ports.MockProviderRepository{
				PostFunc: tt.mockPost,
			}

			uc := NewCronUseCase(mockDB, mockProvider)
			err := uc.RetryPendingTransactions(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			assert.NoError(t, err)
			assert.Equal(t, tt.wantUpdates, updatesCount, "numero de updates no coincide")
		})
	}
}
