package usecases

import (
	"context"
	"errors"
	"testing"

	"bank-api/srv/transactions/domain"
	"bank-api/srv/transactions/ports"

	"github.com/stretchr/testify/assert"
)

type MockPublisher struct {
	PublishFunc func(topic string, message interface{}) error
}

func (m *MockPublisher) Publish(topic string, message interface{}) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(topic, message)
	}
	return nil
}

func (m *MockPublisher) Close() {
}

func TestUpdateUseCase_UpdateStatus(t *testing.T) {
	tests := []struct {
		name          string
		txID          string
		status        string
		mockFindAll   func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error)
		mockUpdate    func(ctx context.Context, tx *domain.Transaction) error
		mockDel       func(ctx context.Context, keys ...string) error
		mockPublish   func(topic string, message interface{}) error
		expectedError string
	}{
		{
			name:   "error in FindAll",
			txID:   "tx-1",
			status: "EXECUTED",
			mockFindAll: func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
				return nil, 0, errors.New("db error")
			},
			expectedError: "db error",
		},
		{
			name:   "transaction not found",
			txID:   "tx-1",
			status: "EXECUTED",
			mockFindAll: func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
				return []*domain.Transaction{}, 0, nil
			},
			expectedError: "transaction not found",
		},
		{
			name:   "error in Update",
			txID:   "tx-1",
			status: "EXECUTED",
			mockFindAll: func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
				return []*domain.Transaction{
					{ID: "tx-1", AccountID: "acc-1"},
				}, 1, nil
			},
			mockUpdate: func(ctx context.Context, tx *domain.Transaction) error {
				return errors.New("update error")
			},
			expectedError: "update error",
		},
		{
			name:   "success with EXECUTED status",
			txID:   "tx-1",
			status: "EXECUTED",
			mockFindAll: func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
				return []*domain.Transaction{
					{ID: "tx-1", AccountID: "acc-1", Amount: 100},
				}, 1, nil
			},
			mockUpdate: func(ctx context.Context, tx *domain.Transaction) error {
				return nil
			},
			mockDel: func(ctx context.Context, keys ...string) error {
				return nil
			},
			mockPublish: func(topic string, message interface{}) error {
				return nil
			},
			expectedError: "",
		},
		{
			name:   "success with REJECTED status",
			txID:   "tx-1",
			status: "REJECTED",
			mockFindAll: func(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
				return []*domain.Transaction{
					{ID: "tx-1", AccountID: "acc-1", Amount: 100},
				}, 1, nil
			},
			mockUpdate: func(ctx context.Context, tx *domain.Transaction) error {
				return nil
			},
			mockDel: func(ctx context.Context, keys ...string) error {
				return nil
			},
			expectedError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbRepo := &ports.MockTransactionRepository{
				FindAllFunc: tt.mockFindAll,
				UpdateFunc:  tt.mockUpdate,
			}
			rdb := &ports.MockRedisClient{
				DelFunc: tt.mockDel,
			}
			pub := &MockPublisher{
				PublishFunc: tt.mockPublish,
			}

			uc := NewUpdateUseCase(dbRepo, nil, nil, rdb, pub)
			err := uc.UpdateStatus(context.Background(), tt.txID, tt.status)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
