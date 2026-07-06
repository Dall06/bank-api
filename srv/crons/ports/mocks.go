package ports

import (
	"context"
	"bank-api/srv/crons/domain"
)

type MockTransactionRepository struct {
	FindAllFunc func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error)
	UpdateFunc  func(ctx context.Context, tx *domain.Transaction) error
}

func (m *MockTransactionRepository) FindAll(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
	if m.FindAllFunc != nil {
		return m.FindAllFunc(ctx, filter, limit, offset)
	}
	return nil, 0, nil
}

func (m *MockTransactionRepository) Update(ctx context.Context, tx *domain.Transaction) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, tx)
	}
	return nil
}

type MockProviderRepository struct {
	PostFunc func(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error)
}

func (m *MockProviderRepository) Post(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error) {
	if m.PostFunc != nil {
		return m.PostFunc(ctx, req)
	}
	return nil, nil
}
