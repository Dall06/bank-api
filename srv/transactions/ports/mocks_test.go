package ports

import (
	"context"
	"testing"
	"time"
	"bank-api/srv/transactions/domain"
	"github.com/stretchr/testify/assert"
)

func TestMocks(t *testing.T) {
	ctx := context.Background()

	emptyRepo := &MockTransactionRepository{}
	emptyRepo.Save(ctx, nil)
	emptyRepo.Update(ctx, nil)
	emptyRepo.FindAll(ctx, nil, 0, 0)

	repo := &MockTransactionRepository{
		SaveFunc: func(ctx context.Context, tx *domain.Transaction) error { return nil },
		UpdateFunc: func(ctx context.Context, tx *domain.Transaction) error { return nil },
		FindAllFunc: func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) { return nil, 0, nil },
	}
	assert.NoError(t, repo.Save(ctx, nil))
	assert.NoError(t, repo.Update(ctx, nil))
	txs, count, err := repo.FindAll(ctx, nil, 0, 0)
	assert.Nil(t, txs)
	assert.Equal(t, 0, count)
	assert.NoError(t, err)

	emptyProvider := &MockProviderRepository{}
	emptyProvider.Post(ctx, domain.ProviderRequest{})

	providerRepo := &MockProviderRepository{
		PostFunc: func(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error) { return nil, nil },
	}
	resp, err := providerRepo.Post(ctx, domain.ProviderRequest{})
	assert.Nil(t, resp)
	assert.NoError(t, err)

	usersRepo := &MockUsersRepository{
		ValidateUserFunc: func(ctx context.Context, accountID string) error { return nil },
	}
	assert.NoError(t, usersRepo.ValidateUser(ctx, ""))

	txUsecase := &MockTransactionUsecase{
		PurchaseFunc: func(ctx context.Context, req domain.CreateTransactionRequest) (*domain.TransactionResponse, error) { return nil, nil },
		GetAllFunc: func(ctx context.Context, req domain.GetTransactionsRequest) (*domain.GetTransactionsResponse, error) { return nil, nil },
		UpdateStatusFunc: func(ctx context.Context, txID, status string) error { return nil },
	}
	resp2, err := txUsecase.Purchase(ctx, domain.CreateTransactionRequest{})
	assert.Nil(t, resp2)
	assert.NoError(t, err)
	resp3, err := txUsecase.GetAll(ctx, domain.GetTransactionsRequest{})
	assert.Nil(t, resp3)
	assert.NoError(t, err)
	assert.NoError(t, txUsecase.UpdateStatus(ctx, "", ""))

	redisMock := &MockRedisClient{
		GetFunc: func(ctx context.Context, key string) (string, error) { return "", nil },
		SetFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error) { return true, nil },
		DelFunc: func(ctx context.Context, keys ...string) error { return nil },
		ConnectFunc: func() error { return nil },
		DisconnectFunc: func() error { return nil },
	}
	val, err := redisMock.Get(ctx, "")
	assert.Equal(t, "", val)
	assert.NoError(t, err)
	ok, err := redisMock.Set(ctx, "", "", 0, false)
	assert.True(t, ok)
	assert.NoError(t, err)
	assert.NoError(t, redisMock.Del(ctx, ""))
	assert.NoError(t, redisMock.Connect())
	assert.NoError(t, redisMock.Disconnect())
}
