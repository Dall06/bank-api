package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"bank-api/env/consts"
	"bank-api/pkg/publisher"
	"bank-api/srv/transactions/domain"
	"bank-api/srv/transactions/ports"

	"github.com/stretchr/testify/assert"
)

func TestTransactionUsecase_GetTransactions(t *testing.T) {
	tests := []struct {
		name        string
		req         domain.GetTransactionsRequest
		mockFindAll func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error)
		wantCount   int
		wantErrType error
	}{
		{
			name: "obtener lista exitosamente",
			req: domain.GetTransactionsRequest{
				AccountID: "acc-123",
				Limit:     2,
				Page:      1,
			},
			mockFindAll: func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
				return []*domain.Transaction{
					{ID: "tx1", AccountID: "acc-123"},
					{ID: "tx2", AccountID: "acc-123"},
				}, 2, nil
			},
			wantCount:   2,
			wantErrType: nil,
		},
		{
			name: "error del repositorio",
			req: domain.GetTransactionsRequest{
				AccountID: "acc-error",
			},
			mockFindAll: func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
				return nil, 0, errors.New("db error")
			},
			wantCount:   0,
			wantErrType: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbRepo := &ports.MockTransactionRepository{FindAllFunc: tt.mockFindAll}
			provRepo := &ports.MockProviderRepository{}
			uc := NewGetUseCase(dbRepo, provRepo, &ports.MockUsersRepository{}, &ports.MockRedisClient{}, publisher.NewNoOpPublisher())

			resp, err := uc.GetAll(context.Background(), tt.req)

			if tt.wantErrType != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrType.Error())
				assert.Nil(t, resp)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, resp.Data, tt.wantCount)
		})
	}
}

func TestTransactionUsecase_GetTransactionsCache(t *testing.T) {
	dbRepo := &ports.MockTransactionRepository{}
	provRepo := &ports.MockProviderRepository{}
	mockUser := &ports.MockUsersRepository{}

	t.Run("retorna listado de cache si existe en Redis", func(t *testing.T) {
		mockRedis := &ports.MockRedisClient{
			GetFunc: func(ctx context.Context, key string) (string, error) {
				if key == "cache:tx:version:acc-cached" {
					return "3", nil
				}
				return `{"data":[{"id":"cached-tx-1","accountId":"acc-cached","amount":500}],"pagination":{"page":1,"limit":10,"total":1}}`, nil
			},
		}

		uc := NewGetUseCase(dbRepo, provRepo, mockUser, mockRedis, publisher.NewNoOpPublisher())

		resp, err := uc.GetAll(context.Background(), domain.GetTransactionsRequest{
			AccountID: "acc-cached",
			Page:      1,
			Limit:     10,
		})

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Data, 1)
		assert.Equal(t, "cached-tx-1", resp.Data[0].ID)
	})

	t.Run("ignora cache si se envia bypass en el contexto", func(t *testing.T) {
		mockRedis := &ports.MockRedisClient{
			GetFunc: func(ctx context.Context, key string) (string, error) {
				if key == "cache:tx:version:acc-cached" {
					return "3", nil
				}
				t.Fatalf("no debería buscarse el query de cache en Redis al usar bypass")
				return "", nil
			},
			SetFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error) {
				return true, nil
			},
		}

		dbRepoBypass := &ports.MockTransactionRepository{
			FindAllFunc: func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
				return []*domain.Transaction{
					{ID: "db-tx-1", AccountID: "acc-cached"},
				}, 1, nil
			},
		}

		uc := NewGetUseCase(dbRepoBypass, provRepo, mockUser, mockRedis, publisher.NewNoOpPublisher())
		ctx := context.WithValue(context.Background(), consts.BypassCacheContextKey, true)

		resp, err := uc.GetAll(ctx, domain.GetTransactionsRequest{
			AccountID: "acc-cached",
			Page:      1,
			Limit:     10,
		})

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Data, 1)
		assert.Equal(t, "db-tx-1", resp.Data[0].ID)
	})
}
