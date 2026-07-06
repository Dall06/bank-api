package domain

import (
	"time"

	"bank-api/pkg/crypto"
	"github.com/uptrace/bun"
)

// ProviderRequest representa la petición al proveedor externo
type ProviderRequest struct {
	AccountID string  `json:"accountId"`
	Type      string  `json:"type"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
}

// ProviderResponse representa la respuesta exitosa del proveedor externo
type ProviderResponse struct {
	TransactionID string    `json:"transactionId"`
	Status        string    `json:"status"`
	Balance       float64   `json:"balance"`
	ExecutedAt    time.Time `json:"executedAt"`
}

// ProviderErrorResponse representa la respuesta fallida del proveedor externo
type ProviderErrorResponse struct {
	Status  string `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Transaction representa la entidad pura de negocio de una transacción
type Transaction struct {
	bun.BaseModel `bun:"table:transactions"`

	ID                    string    `json:"id" bun:"id,pk,type:uuid"`
	AccountID             string    `json:"accountId" bun:"account_id,notnull"`
	Type                  string    `json:"type" bun:"type,notnull"`
	Amount                crypto.EncryptedFloat  `json:"amount" bun:"amount,type:varchar(255),notnull"`
	Currency              string                 `json:"currency" bun:"currency,notnull"`
	Description           crypto.EncryptedString `json:"description" bun:"description,type:varchar(255)"`
	Status                string                 `json:"status" bun:"status,notnull"`
	ProviderTransactionID crypto.EncryptedString `json:"providerTransactionId,omitempty" bun:"provider_transaction_id,type:varchar(255)"`
	BalanceAfter          crypto.EncryptedFloat  `json:"balanceAfter,omitempty" bun:"balance_after,type:varchar(255)"`
	CreatedAt             time.Time              `json:"createdAt" bun:"created_at,notnull,default:current_timestamp"`
}