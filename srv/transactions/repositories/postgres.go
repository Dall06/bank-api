package repositories

import (
	"context"
	"time"

	domain "bank-api/srv/transactions/domain"
	"bank-api/srv/transactions/ports"

	"github.com/uptrace/bun"
)

type TransactionRepository struct {
	db bun.IDB
}

func NewTransactionRepository(db bun.IDB) ports.TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Save(ctx context.Context, tx *domain.Transaction) error {
	if tx.CreatedAt.IsZero() {
		tx.CreatedAt = time.Now()
	}

	_, err := r.db.NewInsert().Model(tx).Exec(ctx)
	return err
}

func (r *TransactionRepository) Update(ctx context.Context, tx *domain.Transaction) error {
	_, err := r.db.NewUpdate().Model(tx).WherePK().Exec(ctx)
	return err
}

func (r *TransactionRepository) FindAll(ctx context.Context, filter *domain.Transaction, limit, offset int) ([]*domain.Transaction, int, error) {
	var txs []*domain.Transaction

	q := r.db.NewSelect().Model(&txs)

	if filter.AccountID != "" {
		q = q.Where("account_id = ?", filter.AccountID)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.Type != "" {
		q = q.Where("type = ?", filter.Type)
	}

	count, err := q.Order("created_at DESC").Limit(limit).Offset(offset).ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return txs, count, nil
}
