package usecases

import (
	"context"
	"time"

	"bank-api/mocks/providers/domain"
)

type providerUsecase struct{}

func NewProviderUsecase() *providerUsecase {
	return &providerUsecase{}
}

func (u *providerUsecase) Execute(ctx context.Context, req domain.ExecuteRequest) (domain.ExecuteResponse, error) {
	// Default mock logic: unconditionally approve.
	// Failure scenarios are intercepted and handled by the MockInterceptorMiddleware in the handler layer.
	return domain.ExecuteResponse{
		TransactionID: "txn-789",
		Status:        "APPROVED",
		Balance:       5500.00,
		ExecutedAt:    time.Now().Format(time.RFC3339),
	}, nil
}
