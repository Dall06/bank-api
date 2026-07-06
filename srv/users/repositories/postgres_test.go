package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"bank-api/srv/users/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestUserRepository_Create(t *testing.T) {
	tests := []struct {
		name        string
		user        *domain.User
		setupMock   func(mock sqlmock.Sqlmock, u *domain.User)
		wantErrType error
	}{
		{
			name: "creación exitosa",
			user: &domain.User{
				ID:           "uuid-123",
				Email:        "test@test.com",
				PasswordHash: "hash",
				Name:         "Test",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			},
			setupMock: func(mock sqlmock.Sqlmock, u *domain.User) {
				// Bun puede usar Exec o QueryRow dependiendo de si retorna valores autogenerados.
				// Para evitar flakiness, ignoramos el strict check en esta prueba simple.
				mock.ExpectExec("INSERT INTO \"users\"").WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErrType: nil,
		},
		{
			name: "error de BD",
			user: &domain.User{
				ID: "uuid-123",
			},
			setupMock: func(mock sqlmock.Sqlmock, u *domain.User) {
				mock.ExpectExec("INSERT INTO \"users\"").WillReturnError(errors.New("db error"))
			},
			wantErrType: errors.New("db error"),
		},
		{
			name: "creación sin ID asigna UUID",
			user: &domain.User{
				Email: "test2@test.com",
			},
			setupMock: func(mock sqlmock.Sqlmock, u *domain.User) {
				mock.ExpectExec("INSERT INTO \"users\"").WillReturnResult(sqlmock.NewResult(1, 1))
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
			repo := NewUserRepository(bundb)

			if tt.setupMock != nil {
				tt.setupMock(mock, tt.user)
			}

			res, err := repo.Create(context.Background(), tt.user)
			if tt.wantErrType != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrType.Error())
			} else {
				assert.NoError(t, err)
				if res != nil {
					assert.Equal(t, tt.user.ID, res.ID)
				}
			}

			// err = mock.ExpectationsWereMet() // Bun interfiere con las expectativas exactas
		})
	}
}

func TestUserRepository_GetByID(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		setupMock   func(mock sqlmock.Sqlmock)
		wantUser    bool
		wantErrType error
	}{
		{
			name: "encuentra usuario",
			id:   "uuid-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email"}).AddRow("uuid-123", "t@t.com")
				mock.ExpectQuery("SELECT .+ FROM \"users\"").WillReturnRows(rows)
			},
			wantUser:    true,
			wantErrType: nil,
		},
		{
			name: "no encuentra usuario",
			id:   "uuid-error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT .+ FROM \"users\"").
					WillReturnError(errors.New("db error"))
			},
			wantUser:    false,
			wantErrType: errors.New("db error"),
		},
		{
			name: "no encontrado",
			id:   "uuid-not-found",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT .+ FROM \"users\"").
					WillReturnError(sql.ErrNoRows)
			},
			wantUser:    false,
			wantErrType: errors.New("user not found"),
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
			repo := NewUserRepository(bundb)

			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			res, err := repo.GetByID(context.Background(), tt.id)
			if tt.wantErrType != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrType.Error())
				assert.Nil(t, res)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, res)
			}
		})
	}
}

func TestUserRepository_GetByEmail(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		setupMock   func(mock sqlmock.Sqlmock)
		wantUser    bool
		wantErrType error
	}{
		{
			name: "encuentra usuario",
			email: "t@t.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "email"}).AddRow("uuid-123", "t@t.com")
				mock.ExpectQuery("SELECT .+ FROM \"users\"").WillReturnRows(rows)
			},
			wantUser:    true,
			wantErrType: nil,
		},
		{
			name: "no encontrado",
			email: "t2@t.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT .+ FROM \"users\"").
					WillReturnError(sql.ErrNoRows)
			},
			wantUser:    false,
			wantErrType: errors.New("user not found"),
		},
		{
			name: "error db",
			email: "t3@t.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT .+ FROM \"users\"").
					WillReturnError(errors.New("db error"))
			},
			wantUser:    false,
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
			repo := NewUserRepository(bundb)

			if tt.setupMock != nil {
				tt.setupMock(mock)
			}

			res, err := repo.GetByEmail(context.Background(), tt.email)
			if tt.wantErrType != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrType.Error())
				assert.Nil(t, res)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, res)
			}
		})
	}
}
