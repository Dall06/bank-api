package repositories

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bank-api/pkg/errs"
	"bank-api/srv/crons/domain"

	"github.com/stretchr/testify/assert"
)

func TestProviderRepository_Post(t *testing.T) {
	tests := []struct {
		name           string
		req            domain.ProviderRequest
		mockServerResp func(w http.ResponseWriter, r *http.Request)
		wantStatus     string
		wantErr        bool
		wantErrType    error
	}{
		{
			name: "éxito - 200 OK",
			req: domain.ProviderRequest{
				AccountID: "acc-123",
				Amount:    100.0,
				Currency:  "MXN",
			},
			mockServerResp: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(domain.ProviderResponse{
					Status:        "APPROVED",
				})
			},
			wantStatus:  "APPROVED",
			wantErr:     false,
			wantErrType: nil,
		},
		{
			name: "rechazado - REJECTED (ValueError)",
			req: domain.ProviderRequest{
				AccountID: "acc-123",
				Amount:    100.0,
				Currency:  "MXN",
			},
			mockServerResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest) // Cualquier no-200 funciona si el body tiene REJECTED
				json.NewEncoder(w).Encode(domain.ProviderErrorResponse{
					Status:  "REJECTED",
					Message: "fondos insuficientes",
				})
			},
			wantStatus:  "",
			wantErr:     true,
			wantErrType: errs.ErrValue,
		},
		{
			name: "error del servidor - 500 (ServiceUnavailableError)",
			req: domain.ProviderRequest{
				AccountID: "acc-123",
				Amount:    100.0,
				Currency:  "MXN",
			},
			mockServerResp: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("internal server error"))
			},
			wantStatus:  "",
			wantErr:     true,
			wantErrType: errs.ErrServiceUnavail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.mockServerResp))
			defer server.Close()

			repo := NewProviderRepository(server.URL, 5*time.Second)
			resp, err := repo.Post(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.ErrorIs(t, err, tt.wantErrType)
				}
				assert.Nil(t, resp)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.wantStatus, resp.Status)
		})
	}
}

func TestProviderRepository_Post_NetworkError(t *testing.T) {
	repo := NewProviderRepository("http://localhost:0", 1*time.Millisecond) // URL inválida/puerto cerrado
	resp, err := repo.Post(context.Background(), domain.ProviderRequest{})

	assert.Error(t, err)
	assert.ErrorIs(t, err, errs.ErrServiceUnavail)
	assert.Nil(t, resp)
}
