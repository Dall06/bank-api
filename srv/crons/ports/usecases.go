package ports

import (
	"context"
)

type CronUseCase interface {
	RetryPendingTransactions(ctx context.Context) error
}
