package domain

import "time"

// TransactionCompletedEvent representa el evento publicado en Kafka
// al completarse una transacción con éxito.
type TransactionCompletedEvent struct {
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
