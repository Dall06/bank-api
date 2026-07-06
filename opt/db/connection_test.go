package database

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestConnect(t *testing.T) {
	tests := []struct {
		name        string
		uri         string
		timeoutSec  int
		wantError   bool
		errContains string
	}{
		{
			name:        "invalid connection uri fails with timeout",
			uri:         "postgres://invalid:invalid@localhost:0/db?sslmode=disable",
			timeoutSec:  1,
			wantError:   true,
			errContains: "failed to connect to database after",
		},
		{
			name:        "invalid connection fast fail if possible",
			uri:         "postgres://invalid:invalid@localhost:0/db?sslmode=disable",
			timeoutSec:  0,
			wantError:   true,
			errContains: "failed to connect to database after", // The timeout check is > 0, so it hits the timeout condition
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := Connect(tt.uri, tt.timeoutSec)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, db)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, db)
			}
		})
	}
}

type dummyModel struct {
	ID   int
	Name string
}

func TestAutoMigrate(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(sqlmock.Sqlmock)
		wantError bool
	}{
		{
			name: "successful migration",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS .*").WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantError: false,
		},
		{
			name: "migration error",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS .*").WillReturnError(errors.New("db error"))
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer mockDB.Close()

			tt.mockSetup(mock)

			db := bun.NewDB(mockDB, pgdialect.New())
			err = AutoMigrate[dummyModel](db)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
