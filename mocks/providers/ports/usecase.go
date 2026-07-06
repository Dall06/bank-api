package ports

import (
	"context"

	"bank-api/mocks/providers/domain"
)

type ProviderUsecase interface {
	Execute(ctx context.Context, req domain.ExecuteRequest) (domain.ExecuteResponse, error)
}
