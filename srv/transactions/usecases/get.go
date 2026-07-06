package usecases

import (
	"bank-api/env/consts"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"bank-api/opt/redis"
	"bank-api/pkg/publisher"
	"bank-api/srv/transactions/domain"
	"bank-api/srv/transactions/ports"
)

type GetUseCase struct {
	dbRepo       ports.TransactionRepository
	providerRepo ports.ProviderRepository
	usersRepo    ports.UsersRepository
	rdb          redis.Client
	pub          publisher.Publisher
}

func NewGetUseCase(
	dbRepo ports.TransactionRepository,
	providerRepo ports.ProviderRepository,
	usersRepo ports.UsersRepository,
	rdb redis.Client,
	pub publisher.Publisher,
) ports.GetUseCase {
	return &GetUseCase{
		dbRepo:       dbRepo,
		providerRepo: providerRepo,
		usersRepo:    usersRepo,
		rdb:          rdb,
		pub:          pub,
	}
}

func (uc *GetUseCase) GetAll(ctx context.Context, req domain.GetTransactionsRequest) (*domain.GetTransactionsResponse, error) {
	bypassCache, _ := ctx.Value(consts.BypassCacheContextKey).(bool)

	var cacheKey string
	if req.AccountID != "" {
		versionKey := "cache:tx:version:" + req.AccountID
		version, err := uc.rdb.Get(ctx, versionKey)
		if err != nil || version == "" {
			version = "0"
		}

		cacheKey = fmt.Sprintf("cache:tx:query:acc:%s:v:%s:status:%s:type:%s:page:%d:limit:%d",
			req.AccountID, version, req.Status, req.Type, req.Page, req.Limit)

		if !bypassCache {
			cachedVal, err := uc.rdb.Get(ctx, cacheKey)
			if err == nil && cachedVal != "" {
				var cachedResp domain.GetTransactionsResponse
				if jsonErr := json.Unmarshal([]byte(cachedVal), &cachedResp); jsonErr == nil {
					return &cachedResp, nil
				}
			}
		}
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}
	page := req.Page
	if page <= 0 {
		page = 1
	}

	offset := (page - 1) * limit

	filter := &domain.Transaction{
		AccountID: req.AccountID,
		Status:    req.Status,
		Type:      req.Type,
	}

	dbTxs, count, err := uc.dbRepo.FindAll(ctx, filter, limit, offset)
	if err != nil {
		return nil, err
	}

	txsResponse := make([]*domain.TransactionResponse, len(dbTxs))
	for i, dbTx := range dbTxs {
		txsResponse[i] = &domain.TransactionResponse{
			ID:                    dbTx.ID,
			AccountID:             dbTx.AccountID,
			Type:                  dbTx.Type,
			Amount:                float64(dbTx.Amount),
			Currency:              dbTx.Currency,
			Description:           string(dbTx.Description),
			Status:                dbTx.Status,
			ProviderTransactionID: string(dbTx.ProviderTransactionID),
			BalanceAfter:          float64(dbTx.BalanceAfter),
			CreatedAt:             dbTx.CreatedAt,
		}
	}

	resp := &domain.GetTransactionsResponse{
		Data: txsResponse,
		Pagination: domain.PaginationMeta{
			Page:  page,
			Limit: limit,
			Total: int64(count),
		},
	}

	if cacheKey != "" {
		respBytes, jsonErr := json.Marshal(resp)
		if jsonErr == nil {
			_, _ = uc.rdb.Set(ctx, cacheKey, string(respBytes), 5*time.Minute, false)
		}
	}

	return resp, nil
}
