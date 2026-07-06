package ports

import (
	"context"
	"testing"
	"bank-api/srv/crons/domain"
	"github.com/stretchr/testify/assert"
)

func TestMocks(t *testing.T) {
	ctx := context.Background()

	emptyRepo := &MockTransactionRepository{}
	emptyRepo.FindAll(ctx, nil, 0, 0)
	emptyRepo.Update(ctx, nil)

	repo := &MockTransactionRepository{
		FindAllFunc: func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
			return nil, 0, nil
		},
		UpdateFunc: func(ctx context.Context, tx *domain.Transaction) error { return nil },
	}
	txs, _, err := repo.FindAll(ctx, nil, 0, 0)
	assert.Nil(t, txs)
	assert.NoError(t, err)
	assert.NoError(t, repo.Update(ctx, nil))

	emptyProvider := &MockProviderRepository{}
	emptyProvider.Post(ctx, domain.ProviderRequest{})

	providerRepo := &MockProviderRepository{
		PostFunc: func(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error) {
			return nil, nil
		},
	}
	resp, err := providerRepo.Post(ctx, domain.ProviderRequest{})
	assert.Nil(t, resp)
	assert.NoError(t, err)
}
