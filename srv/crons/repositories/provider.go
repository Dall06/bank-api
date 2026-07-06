package repositories

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
	
	"bank-api/pkg/errs"
	"bank-api/srv/crons/ports"
	"bank-api/srv/crons/domain"
)

type providerRepository struct {
	baseURL    string
	httpClient *http.Client
}

func NewProviderRepository(baseURL string, timeout time.Duration) ports.ProviderRepository {
	return &providerRepository{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *providerRepository) Post(ctx context.Context, req domain.ProviderRequest) (*domain.ProviderResponse, error) {
	url := c.baseURL

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, errs.InternalError("failed to marshal request: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, errs.InternalError("failed to create http request: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, errs.ServiceUnavailableError("provider call failed: %v", err)
	}
	defer httpResp.Body.Close()

	respBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, errs.InternalError("failed to read provider response: %v", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		var errResp domain.ProviderErrorResponse
		if jsonErr := json.Unmarshal(respBytes, &errResp); jsonErr == nil && errResp.Status == "REJECTED" {
			return nil, errs.ValueError("%s", errResp.Message)
		}
		return nil, errs.ServiceUnavailableError("provider error (http status %d): %s", httpResp.StatusCode, string(respBytes))
	}

	var successResp domain.ProviderResponse
	if err := json.Unmarshal(respBytes, &successResp); err != nil {
		return nil, errs.InternalError("failed to unmarshal provider response: %v", err)
	}

	return &successResp, nil
}
