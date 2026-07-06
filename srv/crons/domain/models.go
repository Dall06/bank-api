package domain

// Transaction representa la estructura de una transacción tal cual la procesa el Cron
type Transaction struct {
	ID        string  `json:"id"`
	AccountID string  `json:"accountId"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Status    string  `json:"status"`
}

// GetTransactionsResponse estructura de la respuesta del GET /internal/transactions
type GetTransactionsResponse struct {
	Data       []*Transaction `json:"data"`
	Pagination struct {
		Total int `json:"total"`
	} `json:"pagination"`
}

// ProviderRequest DTO para el Proveedor de Pagos
type ProviderRequest struct {
	AccountID string  `json:"accountId"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
}

// ProviderResponse DTO de éxito del Proveedor
type ProviderResponse struct {
	Status string `json:"status"`
}

// ProviderErrorResponse DTO de fallo del Proveedor
type ProviderErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
