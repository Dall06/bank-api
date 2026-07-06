package domain

type ExecuteRequest struct {
	AccountID string  `json:"accountId"`
	Type      string  `json:"type"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	MockID    string  `json:"-"` // Not part of the standard JSON payload, populated from header
}

type ExecuteResponse struct {
	TransactionID string  `json:"transactionId"`
	Status        string  `json:"status"`
	Balance       float64 `json:"balance"`
	ExecutedAt    string  `json:"executedAt"`
}

type ExecuteErrorResponse struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ProviderError struct {
	Code    string
	Message string
}

func (e *ProviderError) Error() string {
	return e.Message
}
