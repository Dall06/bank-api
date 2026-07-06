package ports

import (
	"context"
	"time"

	domain "bank-api/srv/transactions/domain"
)

// MockTransactionRepository implementa ports.TransactionRepository de forma dinámica
type MockTransactionRepository struct {
	SaveFunc    func(ctx context.Context, tx *domain.Transaction) error
	UpdateFunc  func(ctx context.Context, tx *domain.Transaction) error
	FindAllFunc func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error)
}

func (m *MockTransactionRepository) Save(ctx context.Context, tx *domain.Transaction) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(ctx, tx)
	}
	return nil
}

func (m *MockTransactionRepository) Update(ctx context.Context, tx *domain.Transaction) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, tx)
	}
	return nil
}

func (m *MockTransactionRepository) FindAll(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
	if m.FindAllFunc != nil {
		return m.FindAllFunc(ctx, filter, limit, offset)
	}
	return nil, 0, nil
}

// MockProviderRepository implementa ports.ProviderRepository de forma dinámica
type MockProviderRepository struct {
	PostFunc func(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error)
}

func (m *MockProviderRepository) Post(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error) {
	if m.PostFunc != nil {
		return m.PostFunc(ctx, req)
	}
	return nil, nil
}

type MockUsersRepository struct {
	ValidateUserFunc func(ctx context.Context, accountID string) error
}

func (m *MockUsersRepository) ValidateUser(ctx context.Context, accountID string) error {
	if m.ValidateUserFunc != nil {
		return m.ValidateUserFunc(ctx, accountID)
	}
	return nil
}

// MockTransactionUsecase implementa las interfaces PurchaseUseCase, GetUseCase y UpdateUseCase
type MockTransactionUsecase struct {
	PurchaseFunc     func(ctx context.Context, req domain.CreateTransactionRequest) (*domain.TransactionResponse, error)
	GetAllFunc       func(ctx context.Context, req domain.GetTransactionsRequest) (*domain.GetTransactionsResponse, error)
	UpdateStatusFunc func(ctx context.Context, txID, status string) error
}

func (m *MockTransactionUsecase) Purchase(ctx context.Context, req domain.CreateTransactionRequest) (*domain.TransactionResponse, error) {
	if m.PurchaseFunc != nil {
		return m.PurchaseFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockTransactionUsecase) GetAll(ctx context.Context, req domain.GetTransactionsRequest) (*domain.GetTransactionsResponse, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockTransactionUsecase) UpdateStatus(ctx context.Context, txID, status string) error {
	if m.UpdateStatusFunc != nil {
		return m.UpdateStatusFunc(ctx, txID, status)
	}
	return nil
}

// MockRedisClient implementa redis.Client de forma dinámica para testing.
type MockRedisClient struct {
	GetFunc        func(ctx context.Context, key string) (string, error)
	SetFunc        func(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error)
	DelFunc        func(ctx context.Context, keys ...string) error
	ConnectFunc    func() error
	DisconnectFunc func() error
}

func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, key)
	}
	return "", nil
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error) {
	if m.SetFunc != nil {
		return m.SetFunc(ctx, key, value, expiration, isNX)
	}
	return true, nil
}

func (m *MockRedisClient) Del(ctx context.Context, keys ...string) error {
	if m.DelFunc != nil {
		return m.DelFunc(ctx, keys...)
	}
	return nil
}

func (m *MockRedisClient) Connect() error {
	if m.ConnectFunc != nil {
		return m.ConnectFunc()
	}
	return nil
}

func (m *MockRedisClient) Disconnect() error {
	if m.DisconnectFunc != nil {
		return m.DisconnectFunc()
	}
	return nil
}
