package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"bank-api/env/consts"
	"bank-api/pkg/cache"
	"bank-api/pkg/crypto"
	"bank-api/pkg/errs"
	"bank-api/srv/transactions/domain"
	"bank-api/srv/transactions/ports"
	"bank-api/pkg/publisher"

	"github.com/stretchr/testify/assert"
)

// mockTimeoutError emula un error de timeout de red (net.Error)
type mockTimeoutError struct{}

func (e mockTimeoutError) Error() string   { return "request timeout" }
func (e mockTimeoutError) Timeout() bool   { return true }
func (e mockTimeoutError) Temporary() bool { return true }

func TestTransactionUsecase_Purchase(t *testing.T) {
	tests := []struct {
		name           string
		req            domain.CreateTransactionRequest
		mockPost       func(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error)
		mockSave       func(ctx context.Context, tx *domain.Transaction) error
		setupMockUsers func(mock *ports.MockUsersRepository)
		wantErrType    error
		wantStatus     string
	}{
		{
			name: "moneda no soportada - no persiste ni llama al proveedor",
			req: domain.CreateTransactionRequest{
				AccountID: "acc-1",
				Type:      "CREDIT",
				Amount:    10.0,
				Currency:  "USD",
			},
			wantErrType: errs.ErrValue,
		},
		{
			name: "monto menor al minimo - no persiste ni llama al proveedor",
			req: domain.CreateTransactionRequest{
				AccountID: "acc-1",
				Type:      "CREDIT",
				Amount:    1.00,
				Currency:  "MXN",
			},
			wantErrType: errs.ErrValue,
		},
		{
			name: "tipo de transaccion invalido - no persiste ni llama al proveedor",
			req: domain.CreateTransactionRequest{
				AccountID: "acc-1",
				Type:      "TRANSFER",
				Amount:    100.0,
				Currency:  "MXN",
			},
			wantErrType: errs.ErrValue,
		},
		{
			name: "debito excede el limite de 10000 - no persiste ni llama al proveedor",
			req: domain.CreateTransactionRequest{
				AccountID: "acc-1",
				Type:      "DEBIT",
				Amount:    10000.01,
				Currency:  "MXN",
			},
			wantErrType: errs.ErrValue,
		},
		{
			name: "debito en el limite maximo de 10000 - exitoso",
			req: domain.CreateTransactionRequest{
				AccountID: "acc-1",
				Type:      "DEBIT",
				Amount:    10000.00,
				Currency:  "MXN",
			},
			mockPost: func(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error) {
				return &domain.ProviderResponse{
					TransactionID: "txn-debit",
					Status:        "APPROVED",
					Balance:       5000.00,
					ExecutedAt:    time.Now(),
				}, nil
			},
			mockSave: func(ctx context.Context, tx *domain.Transaction) error {
				assert.Equal(t, "EXECUTED", tx.Status)
				assert.Equal(t, crypto.EncryptedFloat(5000.00), tx.BalanceAfter)
				return nil
			},
			wantErrType: nil,
			wantStatus:  "EXECUTED",
		},
		{
			name: "llamada al proveedor exitosa (APPROVED)",
			req: domain.CreateTransactionRequest{
				AccountID: "acc-1",
				Type:      "CREDIT",
				Amount:    1500.00,
				Currency:  "MXN",
			},
			mockPost: func(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error) {
				return &domain.ProviderResponse{
					TransactionID: "txn-123",
					Status:        "APPROVED",
					Balance:       3000.00,
					ExecutedAt:    time.Now(),
				}, nil
			},
			mockSave: func(ctx context.Context, tx *domain.Transaction) error {
				assert.Equal(t, "EXECUTED", tx.Status)
				assert.Equal(t, crypto.EncryptedString("txn-123"), tx.ProviderTransactionID)
				return nil
			},
			wantErrType: nil,
			wantStatus:  "EXECUTED",
		},
		{
			name: "rechazo de negocio del proveedor (INSUFFICIENT_FUNDS) - persiste como REJECTED y retorna exito de flujo",
			req: domain.CreateTransactionRequest{
				AccountID: "acc-1",
				Type:      "DEBIT",
				Amount:    5000.00,
				Currency:  "MXN",
			},
			mockPost: func(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error) {
				return nil, errs.ValueError("The account does not have enough balance to complete the transaction")
			},
			mockSave: func(ctx context.Context, tx *domain.Transaction) error {
				assert.Equal(t, "REJECTED", tx.Status)
				return nil
			},
			wantErrType: nil,
			wantStatus:  "REJECTED",
		},
		{
			name: "caida tecnica general del proveedor - persiste como FAILED y retorna error 503",
			req: domain.CreateTransactionRequest{
				AccountID: "acc-1",
				Type:      "CREDIT",
				Amount:    500.00,
				Currency:  "MXN",
			},
			mockPost: func(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error) {
				return nil, errors.New("connection reset by peer")
			},
			mockSave: func(ctx context.Context, tx *domain.Transaction) error {
				assert.Equal(t, "FAILED", tx.Status)
				return nil
			},
			wantErrType: errs.ErrServiceUnavail,
			wantStatus:  "FAILED",
		},
		{
			name: "timeout de red del proveedor - persiste como PENDING y retorna error 503",
			req: domain.CreateTransactionRequest{
				AccountID: "acc-1",
				Type:      "CREDIT",
				Amount:    500.00,
				Currency:  "MXN",
			},
			mockPost: func(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error) {
				return nil, mockTimeoutError{}
			},
			mockSave: func(ctx context.Context, tx *domain.Transaction) error {
				assert.Equal(t, "PENDING", tx.Status)
				return nil
			},
			wantErrType: errs.ErrServiceUnavail,
			wantStatus:  "PENDING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbRepo := &ports.MockTransactionRepository{SaveFunc: tt.mockSave}
			mockProvider := &ports.MockProviderRepository{PostFunc: tt.mockPost}
			mockUser := &ports.MockUsersRepository{}
			if tt.setupMockUsers != nil {
				tt.setupMockUsers(mockUser)
			} else {
				// Por defecto, usuario válido
				mockUser.ValidateUserFunc = func(ctx context.Context, accountID string) error { return nil }
			}

			uc := NewPurchaseUseCase(false, dbRepo, mockProvider, mockUser, cache.NewManager(&ports.MockRedisClient{}), publisher.NewNoOpPublisher())

			resp, err := uc.Purchase(context.Background(), tt.req)

			if tt.wantErrType != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.wantErrType), "debe ser error del tipo %v", tt.wantErrType)
				assert.Nil(t, resp)
				return
			}
			
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.wantStatus, resp.Status)
		})
	}
}

func TestTransactionUsecase_Idempotency(t *testing.T) {
	dbRepo := &ports.MockTransactionRepository{}
	mockProvider := &ports.MockProviderRepository{}
	mockUser := &ports.MockUsersRepository{
		ValidateUserFunc: func(ctx context.Context, accountID string) error { return nil },
	}

	t.Run("retorna respuesta cacheada si existe en Redis sin llamar a DB ni proveedor", func(t *testing.T) {
		mockRedis := &ports.MockRedisClient{
			GetFunc: func(ctx context.Context, key string) (string, error) {
				assert.Equal(t, "idempotency:tx:test-key", key)
				// Retornar una respuesta simulada serializada
				return `{"id":"tx-cached","status":"EXECUTED","amount":100,"currency":"MXN"}`, nil
			},
		}

		uc := NewPurchaseUseCase(false, dbRepo, mockProvider, mockUser, cache.NewManager(mockRedis), publisher.NewNoOpPublisher())
		ctx := context.WithValue(context.Background(), consts.IdempotencyKeyContextKey, "test-key")

		resp, err := uc.Purchase(ctx, domain.CreateTransactionRequest{
			AccountID: "acc-1",
			Type:      "CREDIT",
			Amount:    100.0,
			Currency:  "MXN",
		})

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, "tx-cached", resp.ID)
		assert.Equal(t, "EXECUTED", resp.Status)
		assert.Equal(t, 100.0, resp.Amount)
	})

	t.Run("retorna error de conflicto (409) si la transaccion ya esta en progreso (bloqueada)", func(t *testing.T) {
		mockRedis := &ports.MockRedisClient{
			GetFunc: func(ctx context.Context, key string) (string, error) {
				return "", nil // Miss de caché
			},
			SetFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error) {
				if key == "lock:tx:test-key" && isNX {
					return false, nil // Adquisición de bloqueo falla
				}
				return true, nil
			},
		}

		uc := NewPurchaseUseCase(false, dbRepo, mockProvider, mockUser, cache.NewManager(mockRedis), publisher.NewNoOpPublisher())
		ctx := context.WithValue(context.Background(), consts.IdempotencyKeyContextKey, "test-key")

		resp, err := uc.Purchase(ctx, domain.CreateTransactionRequest{
			AccountID: "acc-1",
			Type:      "CREDIT",
			Amount:    100.0,
			Currency:  "MXN",
		})

		assert.Error(t, err)
		assert.True(t, errors.Is(err, errs.ErrConflict))
		assert.Nil(t, resp)
	})

	t.Run("error de conexion en Redis Get propaga error interno", func(t *testing.T) {
		mockRedis := &ports.MockRedisClient{
			GetFunc: func(ctx context.Context, key string) (string, error) {
				return "", errors.New("redis connection down")
			},
		}

		uc := NewPurchaseUseCase(false, dbRepo, mockProvider, mockUser, cache.NewManager(mockRedis), publisher.NewNoOpPublisher())
		ctx := context.WithValue(context.Background(), consts.IdempotencyKeyContextKey, "test-key")

		resp, err := uc.Purchase(ctx, domain.CreateTransactionRequest{
			AccountID: "acc-1",
			Type:      "CREDIT",
			Amount:    100.0,
			Currency:  "MXN",
		})

		assert.Error(t, err)
		assert.True(t, errors.Is(err, errs.ErrInternal))
		assert.Nil(t, resp)
	})

	t.Run("json malformado en cache propaga error interno", func(t *testing.T) {
		mockRedis := &ports.MockRedisClient{
			GetFunc: func(ctx context.Context, key string) (string, error) {
				return "{invalid json}", nil
			},
		}

		uc := NewPurchaseUseCase(false, dbRepo, mockProvider, mockUser, cache.NewManager(mockRedis), publisher.NewNoOpPublisher())
		ctx := context.WithValue(context.Background(), consts.IdempotencyKeyContextKey, "test-key")

		resp, err := uc.Purchase(ctx, domain.CreateTransactionRequest{
			AccountID: "acc-1",
			Type:      "CREDIT",
			Amount:    100.0,
			Currency:  "MXN",
		})

		assert.Error(t, err)
		assert.True(t, errors.Is(err, errs.ErrInternal))
		assert.Nil(t, resp)
	})

	t.Run("error de conexion en Redis Set al bloquear propaga error interno", func(t *testing.T) {
		mockRedis := &ports.MockRedisClient{
			GetFunc: func(ctx context.Context, key string) (string, error) {
				return "", nil
			},
			SetFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error) {
				return false, errors.New("redis write timeout")
			},
		}

		uc := NewPurchaseUseCase(false, dbRepo, mockProvider, mockUser, cache.NewManager(mockRedis), publisher.NewNoOpPublisher())
		ctx := context.WithValue(context.Background(), consts.IdempotencyKeyContextKey, "test-key")

		resp, err := uc.Purchase(ctx, domain.CreateTransactionRequest{
			AccountID: "acc-1",
			Type:      "CREDIT",
			Amount:    100.0,
			Currency:  "MXN",
		})

		assert.Error(t, err)
		assert.True(t, errors.Is(err, errs.ErrInternal))
		assert.Nil(t, resp)
	})
}
