package repositories

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "bank-api/srv/transactions/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestTransactionRepository_Save(t *testing.T) {
	tests := []struct {
		name        string
		tx          *domain.Transaction
		setupMock   func(mock sqlmock.Sqlmock, tx *domain.Transaction)
		wantErrType error
	}{
		{
			name: "guardado exitoso",
			tx: &domain.Transaction{
				ID:                    "8bbdf9bc-ebc6-43b9-a9a7-96a0b98eb9bf",
				AccountID:             "acc-123",
				Type:                  "CREDIT",
				Amount:                1500.00,
				Currency:              "MXN",
				Description:           "Abono exitoso",
				Status:                "EXECUTED",
				ProviderTransactionID: "txn-abc",
				BalanceAfter:          5000.00,
				CreatedAt:             time.Now(),
			},
			setupMock: func(mock sqlmock.Sqlmock, tx *domain.Transaction) {
				mock.ExpectExec("INSERT INTO \"transactions\"").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErrType: nil,
		},
		{
			name: "fallo de conexion en base de datos al guardar",
			tx: &domain.Transaction{
				ID:                    "8bbdf9bc-ebc6-43b9-a9a7-96a0b98eb9bf",
				AccountID:             "acc-123",
				Type:                  "CREDIT",
				Amount:                1500.00,
				Currency:              "MXN",
				Status:                "EXECUTED",
				CreatedAt:             time.Now(),
			},
			setupMock: func(mock sqlmock.Sqlmock, tx *domain.Transaction) {
				mock.ExpectExec("INSERT INTO \"transactions\"").
					WillReturnError(errors.New("database connection lost"))
			},
			wantErrType: errors.New("database connection lost"),
		},
		{
			name: "guardado con CreatedAt cero",
			tx: &domain.Transaction{
				ID:                    "8bbdf9bc-ebc6-43b9-a9a7-96a0b98eb9bf",
				AccountID:             "acc-123",
				Type:                  "CREDIT",
				Amount:                1500.00,
				Currency:              "MXN",
				Status:                "EXECUTED",
			},
			setupMock: func(mock sqlmock.Sqlmock, tx *domain.Transaction) {
				mock.ExpectExec("INSERT INTO \"transactions\"").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErrType: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to open sqlmock: %s", err)
			}
			defer db.Close()

			bundb := bun.NewDB(db, pgdialect.New())
			repo := NewTransactionRepository(bundb)

			if tt.setupMock != nil {
				tt.setupMock(mock, tt.tx)
			}

			err = repo.Save(context.Background(), tt.tx)
			if tt.wantErrType != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrType.Error())
			} else {
				assert.NoError(t, err)
			}

			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

func TestTransactionRepository_FindAll(t *testing.T) {
	tests := []struct {
		name        string
		filter      *domain.Transaction
		setupMock   func(mock sqlmock.Sqlmock)
		wantErr     bool
	}{
		{
			name: "error de conexion con filtro accountID y status",
			filter: &domain.Transaction{
				AccountID: "acc-123",
				Status:    "EXECUTED",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name:   "error de conexion sin filtros",
			filter: &domain.Transaction{},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("connection lost"))
			},
			wantErr: true,
		},
		{
			name: "error de conexion con filtro de Type",
			filter: &domain.Transaction{
				Type: "CREDIT",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT`).WillReturnError(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to open sqlmock: %s", err)
			}
			defer db.Close()

			bundb := bun.NewDB(db, pgdialect.New())
			repo := NewTransactionRepository(bundb)

			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			res, count, err := repo.FindAll(context.Background(), tt.filter, 10, 0)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, res)
				assert.Equal(t, 0, count)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransactionRepository_Update(t *testing.T) {
	tests := []struct {
		name        string
		tx          *domain.Transaction
		setupMock   func(mock sqlmock.Sqlmock, tx *domain.Transaction)
		wantErrType error
	}{
		{
			name: "actualizacion exitosa",
			tx: &domain.Transaction{
				ID:     "8bbdf9bc-ebc6-43b9-a9a7-96a0b98eb9bf",
				Status: "EXECUTED",
			},
			setupMock: func(mock sqlmock.Sqlmock, tx *domain.Transaction) {
				mock.ExpectExec("UPDATE \"transactions\"").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErrType: nil,
		},
		{
			name: "error en actualizacion",
			tx: &domain.Transaction{
				ID: "8bbdf9bc-ebc6-43b9-a9a7-96a0b98eb9bf",
			},
			setupMock: func(mock sqlmock.Sqlmock, tx *domain.Transaction) {
				mock.ExpectExec("UPDATE \"transactions\"").
					WillReturnError(errors.New("db error"))
			},
			wantErrType: errors.New("db error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to open sqlmock: %s", err)
			}
			defer db.Close()

			bundb := bun.NewDB(db, pgdialect.New())
			repo := NewTransactionRepository(bundb)

			if tt.setupMock != nil {
				tt.setupMock(mock, tt.tx)
			}

			err = repo.Update(context.Background(), tt.tx)
			if tt.wantErrType != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrType.Error())
			} else {
				assert.NoError(t, err)
			}

			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}
