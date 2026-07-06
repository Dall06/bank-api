package repositories

import (
	"bank-api/env/consts"
	"bank-api/pkg/errs"
	"bank-api/pkg/requestor"
	"bank-api/srv/transactions/domain"
	"bank-api/srv/transactions/ports"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type ProviderRepository struct {
	client  *requestor.Client // <-- Cambiado a tu nuevo Wrapper de requestor
	baseURL string
}

// NewProviderRepository inicializa el repositorio envolviendo el http.Client nativo
func NewProviderRepository(baseURL string, timeout time.Duration) ports.ProviderRepository {
	nativeClient := &http.Client{
		Timeout: timeout,
	}

	return &ProviderRepository{
		client:  requestor.NewClient(nativeClient), // <-- Envolvemos el cliente aquí
		baseURL: baseURL,
	}
}

func (r *ProviderRepository) Post(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error) {
	bodyBytes, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errs.InternalError("failed to create request: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if mockID, ok := ctx.Value(consts.MockIDContextKey).(string); ok && mockID != "" {
		httpReq.Header.Set(consts.MockIDHeaderKey, mockID)
	}

	// Invocación limpia: dos retornos
	res, err := r.client.Do(ctx, httpReq, true)
	if err != nil {
		return nil, err
	}

	// Manejo de errores de negocio usando res.StatusCode y res.Bytes
	if res.StatusCode != http.StatusOK {
		var errResp domain.ProviderErrorResponse
		if jsonErr := json.Unmarshal(res.Bytes, &errResp); jsonErr == nil && errResp.Status == "REJECTED" {
			return nil, errs.ValueError("%s", errResp.Message)
		}
		return nil, errs.ServiceUnavailableError("provider error (status %d): %s", res.StatusCode, string(res.Bytes))
	}

	var successResp domain.ProviderResponse
	if err := json.Unmarshal(res.Bytes, &successResp); err != nil {
		return nil, errs.InternalError("failed to unmarshal response: %v", err)
	}

	return &successResp, nil
}
