package usecases

import (
	"context"

	"bank-api/opt/redis"
	"bank-api/pkg/errs"
	"bank-api/pkg/publisher"
	"bank-api/srv/transactions/domain"
	"bank-api/srv/transactions/ports"
)

type UpdateUseCase struct {
	dbRepo       ports.TransactionRepository
	providerRepo ports.ProviderRepository
	usersRepo    ports.UsersRepository
	rdb          redis.Client
	pub          publisher.Publisher
}

func NewUpdateUseCase(
	dbRepo ports.TransactionRepository,
	providerRepo ports.ProviderRepository,
	usersRepo ports.UsersRepository,
	rdb redis.Client,
	pub publisher.Publisher,
) ports.UpdateUseCase {
	return &UpdateUseCase{
		dbRepo:       dbRepo,
		providerRepo: providerRepo,
		usersRepo:    usersRepo,
		rdb:          rdb,
		pub:          pub,
	}
}

func (uc *UpdateUseCase) UpdateStatus(ctx context.Context, txID, status string) error {
	txs, _, err := uc.dbRepo.FindAll(ctx, &domain.Transaction{ID: txID}, 1, 0)
	if err != nil {
		return err
	}
	if len(txs) == 0 {
		return errs.NotFoundError("transaction not found")
	}

	tx := &domain.Transaction{ID: txID, Status: status}
	if err := uc.dbRepo.Update(ctx, tx); err != nil {
		return err
	}

	uc.invalidateCache(ctx, txs[0].AccountID)

	if status == "EXECUTED" {
		event := domain.TransactionCompletedEvent{
			ID:                    txs[0].ID,
			AccountID:             txs[0].AccountID,
			Type:                  txs[0].Type,
			Amount:                float64(txs[0].Amount),
			Currency:              txs[0].Currency,
			Description:           string(txs[0].Description),
			Status:                status,
			ProviderTransactionID: string(txs[0].ProviderTransactionID),
			BalanceAfter:          float64(txs[0].BalanceAfter),
			CreatedAt:             txs[0].CreatedAt,
		}
		_ = uc.pub.Publish("", event)
	}

	return nil
}

func (uc *UpdateUseCase) invalidateCache(ctx context.Context, accountID string) {
	if accountID == "" {
		return
	}
	versionKey := "cache:tx:version:" + accountID
	_ = uc.rdb.Del(ctx, versionKey)
}
