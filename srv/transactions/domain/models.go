package domain

import "time"

// CreateTransactionRequest representa el body de la petición para ejecutar una transacción
type CreateTransactionRequest struct {
	AccountID   string  `json:"accountId" validate:"required"`
	Type        string  `json:"type" validate:"required"`
	Amount      float64 `json:"amount" validate:"required"`
	Currency    string  `json:"currency" validate:"required"`
	Description string  `json:"description"`
}

// TransactionResponse representa la respuesta entregada tras crear/ejecutar una transacción
type TransactionResponse struct {
	ID                    string    `json:"id"`
	AccountID             string    `json:"accountId"`
	Type                  string    `json:"type"`
	Amount                float64   `json:"amount"`
	Currency              string    `json:"currency"`
	Description           string    `json:"description"`
	Status                string    `json:"status"`
	ProviderTransactionID string    `json:"providerTransactionId,omitempty"`
	BalanceAfter          float64   `json:"balanceAfter,omitempty"`
	CreatedAt             time.Time `json:"createdAt"`
}

// GetTransactionsRequest define los parámetros de consulta y paginación aceptados en el GET
type GetTransactionsRequest struct {
	AccountID string `json:"accountId"`
	Status    string `json:"status"`
	Type      string `json:"type"`
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
}


// PaginationMeta contiene la metadata para la respuesta paginada
type PaginationMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

// GetTransactionsResponse representa la respuesta estructurada con la lista de transacciones y su paginación
type GetTransactionsResponse struct {
	Data       []*TransactionResponse `json:"data"`
	Pagination PaginationMeta         `json:"pagination"`
}
