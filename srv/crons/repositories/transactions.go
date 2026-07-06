package repositories

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"bank-api/opt/middlewares"
	"bank-api/pkg/errs"
	"bank-api/pkg/sigil"
	
	"bank-api/srv/crons/ports"
	"bank-api/srv/crons/domain"
)

type transactionRepository struct {
	baseURL     string
	sigilSigner *sigil.Signer
	httpClient  *http.Client
}

func NewTransactionRepository(baseURL string, signer *sigil.Signer) ports.TransactionRepository {
	return &transactionRepository{
		baseURL:     baseURL,
		sigilSigner: signer,
		httpClient:  &http.Client{},
	}
}

func (c *transactionRepository) FindAll(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
	url := fmt.Sprintf("%s?status=%s&limit=%d&offset=%d", c.baseURL, filter.Status, limit, offset)
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, errs.InternalError("error creando request a transactions: %v", err)
	}

	middlewares.NewSigilHeaders(c.sigilSigner).AddHeaders(req, nil)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, errs.ServiceUnavailableError("error llamando a transactions API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, errs.ServiceUnavailableError("transactions API retornó status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, errs.InternalError("error leyendo respuesta de transactions: %v", err)
	}
	var getResp domain.GetTransactionsResponse
	if err := json.Unmarshal(body, &getResp); err != nil {
		return nil, 0, err
	}

	txs := make([]*domain.Transaction, len(getResp.Data))
	for i, r := range getResp.Data {
		txs[i] = &domain.Transaction{
			ID:        r.ID,
			AccountID: r.AccountID,
			Amount:    r.Amount,
			Currency:  r.Currency,
			Status:    r.Status,
		}
	}

	return txs, int(getResp.Pagination.Total), nil
}

func (c *transactionRepository) Update(ctx context.Context, tx *domain.Transaction) error {
	url := fmt.Sprintf("%s/%s/status", c.baseURL, tx.ID)
	
	payload := map[string]string{"status": tx.Status}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return errs.InternalError("error serializando payload: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return errs.InternalError("error creando request update transactions: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	middlewares.NewSigilHeaders(c.sigilSigner).AddHeaders(req, bodyBytes)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errs.ServiceUnavailableError("error llamando a transactions update API: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return errs.ServiceUnavailableError("transactions update API retornó status %d", resp.StatusCode)
	}

	return nil
}
