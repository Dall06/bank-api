package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bank-api/pkg/errs"
	domain "bank-api/srv/transactions/domain"
	"bank-api/srv/transactions/ports"
	"bank-api/env/consts"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestTransactionHandler_Create(t *testing.T) {
	tests := []struct {
		name         string
		reqPayload   interface{}
		missingIdemKey bool
		mockPurchase func(ctx context.Context, req domain.CreateTransactionRequest) (*domain.TransactionResponse, error)
		wantCode     int
		wantStatus   string
		wantError    string
	}{
		{
			name:       "cuerpo del request invalido (JSON malformado) - HTTP 400",
			reqPayload: "invalid json string",
			wantCode:   http.StatusBadRequest,
			wantError:  "invalid request body",
		},
		{
			name: "error de negocio del caso de uso (monto invalido) - HTTP 400",
			reqPayload: domain.CreateTransactionRequest{
				AccountID: "acc-1",
				Type:      "CREDIT",
				Amount:    0.50,
				Currency:  "MXN",
			},
			mockPurchase: func(ctx context.Context, req domain.CreateTransactionRequest) (*domain.TransactionResponse, error) {
				return nil, errs.ValueError("amount must be greater than 1.00")
			},
			wantCode:  http.StatusBadRequest,
			wantError: "amount must be greater than 1.00",
		},
		{
			name: "compra exitosa (APPROVED) - HTTP 201",
			reqPayload: domain.CreateTransactionRequest{
				AccountID: "acc-1",
				Type:      "CREDIT",
				Amount:    100.00,
				Currency:  "MXN",
			},
			mockPurchase: func(ctx context.Context, req domain.CreateTransactionRequest) (*domain.TransactionResponse, error) {
				return &domain.TransactionResponse{
					ID:           "txn-123",
					AccountID:    req.AccountID,
					Type:         req.Type,
					Amount:       req.Amount,
					Currency:     req.Currency,
					Status:       "EXECUTED",
					BalanceAfter: 5000.00,
					CreatedAt:    time.Now(),
				}, nil
			},
			wantCode:   http.StatusCreated,
			wantStatus: "EXECUTED",
		},
		{
			name: "compra rechazada por el proveedor (REJECTED) - HTTP 201",
			reqPayload: domain.CreateTransactionRequest{
				AccountID: "acc-insufficient",
				Type:      "DEBIT",
				Amount:    15000.00,
				Currency:  "MXN",
			},
			mockPurchase: func(ctx context.Context, req domain.CreateTransactionRequest) (*domain.TransactionResponse, error) {
				return &domain.TransactionResponse{
					ID:        "txn-123",
					AccountID: req.AccountID,
					Type:      req.Type,
					Amount:    req.Amount,
					Currency:  req.Currency,
					Status:    "REJECTED",
					CreatedAt: time.Now(),
				}, nil
			},
			wantCode:   http.StatusCreated,
			wantStatus: "REJECTED",
		},
		{
			name: "caida del proveedor externo - HTTP 503",
			reqPayload: domain.CreateTransactionRequest{
				AccountID: "acc-500",
				Type:      "CREDIT",
				Amount:    100.00,
				Currency:  "MXN",
			},
			mockPurchase: func(ctx context.Context, req domain.CreateTransactionRequest) (*domain.TransactionResponse, error) {
				return nil, errs.ServiceUnavailableError("external provider unavailable: server error")
			},
			wantCode:  http.StatusServiceUnavailable,
			wantError: "internal error",
		},
		{
			name: "falta idempotency key genera uno - HTTP 201",
			reqPayload: domain.CreateTransactionRequest{
				AccountID: "acc-1",
				Type:      "CREDIT",
				Amount:    100.00,
				Currency:  "MXN",
			},
			missingIdemKey: true,
			mockPurchase: func(ctx context.Context, req domain.CreateTransactionRequest) (*domain.TransactionResponse, error) {
				// We can check if IdempotencyKeyContextKey is set
				idemKey := ctx.Value(consts.IdempotencyKeyContextKey)
				if idemKey == nil || idemKey == "" {
					return nil, errors.New("no idempotency key generated")
				}
				return &domain.TransactionResponse{
					ID:        "txn-123",
					Status:    "EXECUTED",
				}, nil
			},
			wantCode:   http.StatusCreated,
			wantStatus: "EXECUTED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup Echo
			e := echo.New()
			e.HTTPErrorHandler = errs.ErrorHandler
			var reqBody []byte
			if strPayload, ok := tt.reqPayload.(string); ok {
				reqBody = []byte(strPayload)
			} else {
				reqBody, _ = json.Marshal(tt.reqPayload)
			}

			req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewReader(reqBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			if !tt.missingIdemKey {
				req.Header.Set("X-Idempotency-Key", "test-idem-key")
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUsecase := &ports.MockTransactionUsecase{
				PurchaseFunc: tt.mockPurchase,
			}
			handler := NewTransactionHandler(mockUsecase, mockUsecase, mockUsecase)

			err := handler.Create(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}

			assert.Equal(t, tt.wantCode, rec.Code)

			if tt.wantError != "" {
				var errResp errs.Response
				_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
				assert.Contains(t, errResp.Error, tt.wantError)
			}

			if tt.wantStatus != "" {
				var txResp domain.TransactionResponse
				_ = json.Unmarshal(rec.Body.Bytes(), &txResp)
				assert.Equal(t, tt.wantStatus, txResp.Status)
			}
		})
	}
}

func TestTransactionHandler_GetTransactions(t *testing.T) {
	tests := []struct {
		name                string
		queryParams         string
		headerCache         string
		userID              string
		mockGetTransactions func(ctx context.Context, req domain.GetTransactionsRequest) (*domain.GetTransactionsResponse, error)
		wantCode            int
		wantError           string
	}{
		{
			name:        "page invalida (no entero) - HTTP 400",
			queryParams: "?page=abc",
			wantCode:    http.StatusBadRequest,
			wantError:   "invalid page value",
		},
		{
			name:        "limit invalido (no entero) - HTTP 400",
			queryParams: "?limit=xyz",
			wantCode:    http.StatusBadRequest,
			wantError:   "invalid limit value",
		},
		{
			name:        "error del caso de uso - HTTP 500",
			queryParams: "?accountId=acc-1",
			mockGetTransactions: func(ctx context.Context, req domain.GetTransactionsRequest) (*domain.GetTransactionsResponse, error) {
				return nil, errors.New("db error connection lost")
			},
			wantCode:  http.StatusInternalServerError,
			wantError: "internal error",
		},
		{
			name:        "consulta exitosa con accountId en query param - HTTP 200",
			queryParams: "?accountId=acc-123",
			mockGetTransactions: func(ctx context.Context, req domain.GetTransactionsRequest) (*domain.GetTransactionsResponse, error) {
				if req.AccountID != "acc-123" {
					return nil, errors.New("accountId no coincide")
				}
				return &domain.GetTransactionsResponse{
					Data:       []*domain.TransactionResponse{},
					Pagination: domain.PaginationMeta{Page: 1, Limit: 10, Total: 0},
				}, nil
			},
			wantCode: http.StatusOK,
		},
		{
			name:        "consulta sin filtros - HTTP 200",
			queryParams: "",
			mockGetTransactions: func(ctx context.Context, req domain.GetTransactionsRequest) (*domain.GetTransactionsResponse, error) {
				return &domain.GetTransactionsResponse{
					Data:       []*domain.TransactionResponse{},
					Pagination: domain.PaginationMeta{Page: 1, Limit: 10, Total: 0},
				}, nil
			},
			wantCode: http.StatusOK,
		},
		{
			name:        "consulta con bypass cache - HTTP 200",
			queryParams: "",
			headerCache: "no-cache",
			mockGetTransactions: func(ctx context.Context, req domain.GetTransactionsRequest) (*domain.GetTransactionsResponse, error) {
				bypass := ctx.Value(consts.BypassCacheContextKey)
				if bypass != true {
					return nil, errors.New("cache not bypassed")
				}
				return &domain.GetTransactionsResponse{
					Data:       []*domain.TransactionResponse{},
					Pagination: domain.PaginationMeta{Page: 1, Limit: 10, Total: 0},
				}, nil
			},
			wantCode: http.StatusOK,
		},
		{
			name:        "consulta exitosa con resultado de array vacio - HTTP 200",
			queryParams: "",
			mockGetTransactions: func(ctx context.Context, req domain.GetTransactionsRequest) (*domain.GetTransactionsResponse, error) {
				return &domain.GetTransactionsResponse{
					Data:       []*domain.TransactionResponse{},
					Pagination: domain.PaginationMeta{Page: 1, Limit: 10, Total: 0},
				}, nil
			},
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.HTTPErrorHandler = errs.ErrorHandler
			req := httptest.NewRequest(http.MethodGet, "/transactions"+tt.queryParams, nil)
			if tt.headerCache != "" {
				req.Header.Set("Cache-Control", tt.headerCache)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUsecase := &ports.MockTransactionUsecase{
				GetAllFunc: tt.mockGetTransactions,
			}
			handler := NewTransactionHandler(mockUsecase, mockUsecase, mockUsecase)

			err := handler.Get(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}

			assert.Equal(t, tt.wantCode, rec.Code)

			if tt.wantError != "" {
				var errResp errs.Response
				_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
				assert.Contains(t, errResp.Error, tt.wantError)
			}
		})
	}
}

func TestTransactionHandler_GetInternal(t *testing.T) {
	tests := []struct {
		name                string
		queryParams         string
		headerCache         string
		mockGetTransactions func(ctx context.Context, req domain.GetTransactionsRequest) (*domain.GetTransactionsResponse, error)
		wantCode            int
		wantError           string
	}{
		{
			name:        "page invalida (no entero) - HTTP 400",
			queryParams: "?page=abc",
			wantCode:    http.StatusBadRequest,
			wantError:   "invalid page value",
		},
		{
			name:        "limit invalido (no entero) - HTTP 400",
			queryParams: "?limit=xyz",
			wantCode:    http.StatusBadRequest,
			wantError:   "invalid limit value",
		},
		{
			name:        "error del caso de uso - HTTP 500",
			queryParams: "?status=PENDING",
			mockGetTransactions: func(ctx context.Context, req domain.GetTransactionsRequest) (*domain.GetTransactionsResponse, error) {
				return nil, errors.New("db error")
			},
			wantCode:  http.StatusInternalServerError,
			wantError: "internal error",
		},
		{
			name:        "consulta exitosa sin cache - HTTP 200",
			queryParams: "?status=PENDING",
			headerCache: "no-cache",
			mockGetTransactions: func(ctx context.Context, req domain.GetTransactionsRequest) (*domain.GetTransactionsResponse, error) {
				bypass := ctx.Value(consts.BypassCacheContextKey)
				if bypass != true {
					return nil, errors.New("cache not bypassed")
				}
				return &domain.GetTransactionsResponse{
					Data:       []*domain.TransactionResponse{},
					Pagination: domain.PaginationMeta{Page: 1, Limit: 10, Total: 0},
				}, nil
			},
			wantCode: http.StatusOK,
		},
		{
			name:        "consulta exitosa sin filtros - HTTP 200",
			queryParams: "",
			mockGetTransactions: func(ctx context.Context, req domain.GetTransactionsRequest) (*domain.GetTransactionsResponse, error) {
				return &domain.GetTransactionsResponse{
					Data:       []*domain.TransactionResponse{},
					Pagination: domain.PaginationMeta{Page: 1, Limit: 10, Total: 0},
				}, nil
			},
			wantCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.HTTPErrorHandler = errs.ErrorHandler
			req := httptest.NewRequest(http.MethodGet, "/internal/transactions"+tt.queryParams, nil)
			if tt.headerCache != "" {
				req.Header.Set("Cache-Control", tt.headerCache)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUsecase := &ports.MockTransactionUsecase{
				GetAllFunc: tt.mockGetTransactions,
			}
			handler := NewTransactionHandler(mockUsecase, mockUsecase, mockUsecase)

			err := handler.GetInternal(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}

			assert.Equal(t, tt.wantCode, rec.Code)

			if tt.wantError != "" {
				var errResp errs.Response
				_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
				assert.Contains(t, errResp.Error, tt.wantError)
			}
		})
	}
}


func TestTransactionHandler_UpdateStatusInternal(t *testing.T) {
	tests := []struct {
		name       string
		txID       string
		reqPayload any
		mockUpdate func(ctx context.Context, txID, status string) error
		wantCode   int
		wantError  string
	}{
		{
			name:       "missing tx id - HTTP 400",
			txID:       "",
			reqPayload: map[string]string{"status": "EXECUTED"},
			wantCode:   http.StatusBadRequest,
			wantError:  "missing transaction id",
		},
		{
			name:       "invalid body - HTTP 400",
			txID:       "tx-1",
			reqPayload: "invalid-json",
			wantCode:   http.StatusBadRequest,
			wantError:  "invalid body",
		},
		{
			name:       "usecase error - HTTP 500",
			txID:       "tx-1",
			reqPayload: map[string]string{"status": "EXECUTED"},
			mockUpdate: func(ctx context.Context, txID, status string) error {
				return errors.New("db error")
			},
			wantCode:  http.StatusInternalServerError,
			wantError: "internal error",
		},
		{
			name:       "success - HTTP 204",
			txID:       "tx-1",
			reqPayload: map[string]string{"status": "EXECUTED"},
			mockUpdate: func(ctx context.Context, txID, status string) error {
				return nil
			},
			wantCode: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.HTTPErrorHandler = errs.ErrorHandler
			var reqBody []byte
			if strPayload, ok := tt.reqPayload.(string); ok {
				reqBody = []byte(strPayload)
			} else {
				reqBody, _ = json.Marshal(tt.reqPayload)
			}

			req := httptest.NewRequest(http.MethodPatch, "/internal/transactions/", bytes.NewReader(reqBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("id")
			c.SetParamValues(tt.txID)

			mockUsecase := &ports.MockTransactionUsecase{
				UpdateStatusFunc: tt.mockUpdate,
			}
			handler := NewTransactionHandler(mockUsecase, mockUsecase, mockUsecase)

			err := handler.UpdateStatusInternal(c)
			if err != nil {
				e.HTTPErrorHandler(err, c)
			}

			assert.Equal(t, tt.wantCode, rec.Code)

			if tt.wantError != "" {
				var errResp errs.Response
				_ = json.Unmarshal(rec.Body.Bytes(), &errResp)
				assert.Contains(t, errResp.Error, tt.wantError)
			}
		})
	}
}
