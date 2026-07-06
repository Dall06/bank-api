package usecases

import (
	"bank-api/env/consts"
	"bank-api/pkg/cache"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"bank-api/pkg/crypto"
	"bank-api/pkg/errs"
	"bank-api/pkg/mappers"
	"bank-api/pkg/publisher"
	"bank-api/srv/transactions/domain"
	"bank-api/srv/transactions/ports"

	"github.com/google/uuid"
)

type PurchaseUseCase struct {
	jwtValidationActive bool
	dbRepo              ports.TransactionRepository
	providerRepo        ports.ProviderRepository
	usersRepo           ports.UsersRepository
	cache               *cache.Manager
	pub                 publisher.Publisher
}

func NewPurchaseUseCase(
	jwtValidationActive bool,
	dbRepo ports.TransactionRepository,
	providerRepo ports.ProviderRepository,
	usersRepo ports.UsersRepository,
	cacheMgr *cache.Manager,
	pub publisher.Publisher,
) ports.PurchaseUseCase {
	return &PurchaseUseCase{
		jwtValidationActive: jwtValidationActive,
		dbRepo:              dbRepo,
		providerRepo:        providerRepo,
		usersRepo:           usersRepo,
		cache:               cacheMgr,
		pub:                 pub,
	}
}

func (uc *PurchaseUseCase) Purchase(ctx context.Context, req domain.CreateTransactionRequest) (*domain.TransactionResponse, error) {
	idemKey, _ := ctx.Value(consts.IdempotencyKeyContextKey).(string)

	fullIdemKey := ""
	fullLockKey := ""
	if idemKey != "" {
		fullIdemKey = "idempotency:tx:" + idemKey
		fullLockKey = "lock:tx:" + idemKey
	}

	// 1. Obtener bytes crudos de la caché de idempotencia
	cachedBytes, found, err := uc.cache.GetIdempotency(ctx, fullIdemKey)
	if err != nil {
		return nil, err
	}
	if found {
		var cachedResp domain.TransactionResponse
		if unmarshalErr := json.Unmarshal(cachedBytes, &cachedResp); unmarshalErr != nil {
			return nil, errs.InternalError("failed to unmarshal cached transaction response: %v", unmarshalErr)
		}
		return &cachedResp, nil
	}

	// 2. Bloqueo distribuido
	unlock, err := uc.cache.Lock(ctx, fullLockKey)
	if err != nil {
		return nil, err
	}
	defer unlock()

	// 3. Validación de usuario
	if uc.jwtValidationActive {
		if err := uc.usersRepo.ValidateUser(ctx, req.AccountID); err != nil {
			return nil, err
		}
	}

	// 4. Reglas de negocio
	if req.Currency != "MXN" {
		return nil, errs.ValueError("only MXN currency is supported")
	}
	if req.Amount <= 1.00 {
		return nil, errs.ValueError("amount must be greater than 1.00")
	}
	if req.Type != "CREDIT" && req.Type != "DEBIT" {
		return nil, errs.ValueError("invalid transaction type, must be CREDIT or DEBIT")
	}
	if req.Type == "DEBIT" && req.Amount > 10000.00 {
		return nil, errs.ValueError("debit transaction amount cannot exceed 10000.00")
	}

	txID := uuid.New().String()
	now := time.Now()

	txEntity := &domain.Transaction{
		ID:          txID,
		AccountID:   req.AccountID,
		Type:        req.Type,
		Amount:      crypto.EncryptedFloat(req.Amount),
		Currency:    req.Currency,
		Description: crypto.EncryptedString(req.Description),
		CreatedAt:   now,
	}

	providerCtx := ctx
	if mockID, ok := ctx.Value(consts.MockIDContextKey).(string); ok && mockID != "" {
		providerCtx = context.WithValue(ctx, consts.MockIDContextKey, mockID)
	}

	// 5. Consumo de proveedor
	providerResp, providerErr := uc.providerRepo.Post(providerCtx, domain.ProviderRequest{
		AccountID: req.AccountID,
		Type:      req.Type,
		Amount:    req.Amount,
		Currency:  req.Currency,
	})

	if providerErr != nil {
		status := "FAILED"
		var netErr net.Error
		if errors.Is(providerErr, context.DeadlineExceeded) || (errors.As(providerErr, &netErr) && netErr.Timeout()) {
			status = "PENDING"
		}
		if errors.Is(providerErr, errs.ErrValue) {
			status = "REJECTED"
		}

		txEntity.Status = status
		if saveErr := uc.dbRepo.Save(ctx, txEntity); saveErr != nil {
			return nil, errs.InternalError("failed to save failed/pending transaction: %v", saveErr)
		}

		if status == "REJECTED" {
			return &domain.TransactionResponse{
				ID:          txEntity.ID,
				AccountID:   txEntity.AccountID,
				Type:        txEntity.Type,
				Amount:      float64(txEntity.Amount),
				Currency:    txEntity.Currency,
				Description: string(txEntity.Description),
				Status:      txEntity.Status,
				CreatedAt:   txEntity.CreatedAt,
			}, nil
		}

		return nil, errs.ServiceUnavailableError("external provider unavailable: %s", providerErr.Error())
	}

	// 6. Persistencia local
	txEntity.Status = mappers.MapProviderStatus(providerResp.Status)
	txEntity.ProviderTransactionID = crypto.EncryptedString(providerResp.TransactionID)
	txEntity.BalanceAfter = crypto.EncryptedFloat(providerResp.Balance)

	if saveErr := uc.dbRepo.Save(ctx, txEntity); saveErr != nil {
		return nil, errs.InternalError("failed to save executed transaction: %v", saveErr)
	}

	// 7. Invalidación asíncrona de la versión de la caché
	if txEntity.AccountID != "" {
		go func(accID string) {
			// Usamos Background para que la cancelación del request HTTP no mate a la gorutina
			bgCtx := context.Background()
			versionKey := "cache:tx:version:" + accID
			version, versionErr := uc.cache.GetRaw(bgCtx, versionKey)

			vInt := 0
			if versionErr == nil && version != "" {
				_, _ = fmt.Sscanf(version, "%d", &vInt)
			}
			vInt++

			_ = uc.cache.Save(bgCtx, versionKey, fmt.Sprintf("%d", vInt), 30*24*time.Hour)
		}(txEntity.AccountID)
	}

	resp := &domain.TransactionResponse{
		ID:                    txEntity.ID,
		AccountID:             txEntity.AccountID,
		Type:                  txEntity.Type,
		Amount:                float64(txEntity.Amount),
		Currency:              txEntity.Currency,
		Description:           string(txEntity.Description),
		Status:                txEntity.Status,
		ProviderTransactionID: string(txEntity.ProviderTransactionID),
		BalanceAfter:          float64(txEntity.BalanceAfter),
		CreatedAt:             txEntity.CreatedAt,
	}

	// 8. Publicación del evento en Kafka
	if resp.Status == "EXECUTED" {
		event := domain.TransactionCompletedEvent{
			ID:                    resp.ID,
			AccountID:             resp.AccountID,
			Type:                  resp.Type,
			Amount:                resp.Amount,
			Currency:              resp.Currency,
			Description:           resp.Description,
			Status:                resp.Status,
			ProviderTransactionID: resp.ProviderTransactionID,
			BalanceAfter:          resp.BalanceAfter,
			CreatedAt:             resp.CreatedAt,
		}
		// Publicar asíncronamente para no bloquear la respuesta HTTP
		go func(e domain.TransactionCompletedEvent) {
			_ = uc.pub.Publish("", e)
		}(event)
	}

	// 9. Escritura final de la respuesta en la caché
	if cacheErr := uc.cache.Save(ctx, fullIdemKey, resp, 24*time.Hour); cacheErr != nil {
		return nil, cacheErr
	}

	return resp, nil
}
