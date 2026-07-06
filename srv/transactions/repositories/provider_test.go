package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bank-api/pkg/errs"
	domain "bank-api/srv/transactions/domain"

	"github.com/stretchr/testify/assert"
)

func TestProviderRepository_Post(t *testing.T) {
	tests := []struct {
		name         string
		req          domain.ProviderRequest
		serverResponse func(w http.ResponseWriter, r *http.Request, attempt int)
		timeout      time.Duration
		ctxTimeout   time.Duration
		wantErrType  error
		wantStatus   string
	}{
		{
			name: "llamada exitosa al proveedor (APPROVED)",
			req: domain.ProviderRequest{
				AccountID: "acc-123",
				Type:      "CREDIT",
				Amount:    1500.00,
				Currency:  "MXN",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request, attempt int) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"transactionId": "txn-789",
					"status":        "APPROVED",
					"balance":       5500.00,
					"executedAt":    "2026-07-04T10:30:00Z",
				})
			},
			timeout:     1 * time.Second,
			wantErrType: nil,
			wantStatus:  "APPROVED",
		},
		{
			name: "rechazo de negocio del proveedor (INSUFFICIENT_FUNDS) - retorna ProviderErrorResponse",
			req: domain.ProviderRequest{
				AccountID: "acc-insufficient",
				Type:      "DEBIT",
				Amount:    15000.00,
				Currency:  "MXN",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request, attempt int) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnprocessableEntity)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"status":  "REJECTED",
					"code":    "INSUFFICIENT_FUNDS",
					"message": "The account does not have enough balance to complete the transaction",
				})
			},
			timeout:     1 * time.Second,
			wantErrType: errs.ErrValue,
		},
		{
			name: "error interno del proveedor (500) - reintenta y finalmente falla",
			req: domain.ProviderRequest{
				AccountID: "acc-500",
				Type:      "CREDIT",
				Amount:    100.00,
				Currency:  "MXN",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request, attempt int) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			timeout:     100 * time.Millisecond, // timeout corto
			wantErrType: errs.ErrServiceUnavail,
		},
		{
			name: "timeout de red del proveedor - retorna error de timeout",
			req: domain.ProviderRequest{
				AccountID: "acc-timeout",
				Type:      "CREDIT",
				Amount:    100.00,
				Currency:  "MXN",
			},
			serverResponse: func(w http.ResponseWriter, r *http.Request, attempt int) {
				time.Sleep(200 * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			},
			timeout:     50 * time.Millisecond, // client timeout
			wantErrType: context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempt := 0
			// Servidor de pruebas HTTP local
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempt++
				tt.serverResponse(w, r, attempt)
			}))
			defer server.Close()

			repo := NewProviderRepository(server.URL, tt.timeout)

			ctx := context.Background()
			if tt.ctxTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tt.ctxTimeout)
				defer cancel()
			}

			resp, err := repo.Post(ctx, tt.req)

			if tt.wantErrType != nil {
				assert.Error(t, err)
				if errors.Is(tt.wantErrType, errs.ErrValue) {
					assert.True(t, errors.Is(err, errs.ErrValue))
					assert.Contains(t, err.Error(), "does not have enough balance")
				} else if errors.Is(tt.wantErrType, context.DeadlineExceeded) {
					var netErr net.Error
					isTimeout := errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout())
					assert.True(t, isTimeout, "el error de retorno debe indicar timeout")
				} else {
					assert.True(t, errors.Is(err, tt.wantErrType))
				}
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.wantStatus, resp.Status)
			}
		})
	}
}
