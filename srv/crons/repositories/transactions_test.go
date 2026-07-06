package repositories

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bank-api/pkg/errs"
	"bank-api/pkg/sigil"
	"bank-api/srv/crons/domain"

	"github.com/stretchr/testify/assert"
)

func TestTransactionRepository_FindAll(t *testing.T) {
	signer := sigil.NewSigner(sigil.DefaultConfig("test-secret", "test-app"))

	tests := []struct {
		name           string
		filter         *domain.Transaction
		mockServerResp func(w http.ResponseWriter, r *http.Request)
		wantCount      int
		wantTotal      int
		wantErr        bool
		wantErrType    error
	}{
		{
			name:   "éxito - 200 OK",
			filter: &domain.Transaction{Status: "PENDING"},
			mockServerResp: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "PENDING", r.URL.Query().Get("status"))
				assert.Equal(t, "10", r.URL.Query().Get("limit"))
				assert.Equal(t, "0", r.URL.Query().Get("offset"))
				
				// Validar header de Sigil
				assert.NotEmpty(t, r.Header.Get("X-Service-Signature"))

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(domain.GetTransactionsResponse{
					Data: []*domain.Transaction{
						{ID: "tx-1", Status: "PENDING"},
						{ID: "tx-2", Status: "PENDING"},
					},
					Pagination: struct {
						Total int `json:"total"`
					}{Total: 2},
				})
			},
			wantCount:   2,
			wantTotal:   2,
			wantErr:     false,
			wantErrType: nil,
		},
		{
			name:   "error del servidor - 500",
			filter: &domain.Transaction{Status: "PENDING"},
			mockServerResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantCount:   0,
			wantTotal:   0,
			wantErr:     true,
			wantErrType: errs.ErrServiceUnavail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.mockServerResp))
			defer server.Close()

			repo := NewTransactionRepository(server.URL, signer)
			txs, total, err := repo.FindAll(context.Background(), tt.filter, 10, 0)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.ErrorIs(t, err, tt.wantErrType)
				}
				assert.Nil(t, txs)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, txs, tt.wantCount)
			assert.Equal(t, tt.wantTotal, total)
		})
	}
}

func TestTransactionRepository_Update(t *testing.T) {
	signer := sigil.NewSigner(sigil.DefaultConfig("test-secret", "test-app"))

	tests := []struct {
		name           string
		tx             *domain.Transaction
		mockServerResp func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		wantErrType    error
	}{
		{
			name: "éxito - 200 OK",
			tx:   &domain.Transaction{ID: "tx-1", Status: "EXECUTED"},
			mockServerResp: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.NotEmpty(t, r.Header.Get("X-Service-Signature"))

				w.WriteHeader(http.StatusOK)
			},
			wantErr:     false,
			wantErrType: nil,
		},
		{
			name: "error del servidor - 500",
			tx:   &domain.Transaction{ID: "tx-1", Status: "EXECUTED"},
			mockServerResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr:     true,
			wantErrType: errs.ErrServiceUnavail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.mockServerResp))
			defer server.Close()

			repo := NewTransactionRepository(server.URL, signer)
			err := repo.Update(context.Background(), tt.tx)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.ErrorIs(t, err, tt.wantErrType)
				}
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestTransactionRepository_NetworkError(t *testing.T) {
	signer := sigil.NewSigner(sigil.DefaultConfig("test-secret", "test-app"))
	repo := NewTransactionRepository("http://localhost:0", signer) // URL inválida/puerto cerrado

	txs, total, err := repo.FindAll(context.Background(), &domain.Transaction{Status: "PENDING"}, 10, 0)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrServiceUnavail)
	assert.Nil(t, txs)
	assert.Equal(t, 0, total)

	err = repo.Update(context.Background(), &domain.Transaction{ID: "tx-1"})
	assert.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrServiceUnavail)
}
