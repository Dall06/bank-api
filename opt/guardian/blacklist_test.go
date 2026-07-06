package guardian

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// ============================================================================
// Tests for NoopBlacklist
// ============================================================================

func TestNoopBlacklist_Revoke(t *testing.T) {
	bl := &NoopBlacklist{}
	ctx := context.Background()

	tests := []struct {
		name      string
		jti       string
		expiresAt time.Time
	}{
		{
			name:      "success - valid jti and expiration",
			jti:       "test-jti-123",
			expiresAt: time.Now().Add(time.Hour),
		},
		{
			name:      "success - empty jti",
			jti:       "",
			expiresAt: time.Now().Add(time.Hour),
		},
		{
			name:      "success - past expiration",
			jti:       "test-jti-456",
			expiresAt: time.Now().Add(-time.Hour),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bl.Revoke(ctx, tt.jti, tt.expiresAt)
			assert.NoError(t, err, "NoopBlacklist.Revoke should always return nil")
		})
	}
}

func TestNoopBlacklist_IsRevoked(t *testing.T) {
	bl := &NoopBlacklist{}
	ctx := context.Background()

	tests := []struct {
		name string
		jti  string
	}{
		{
			name: "always false - valid jti",
			jti:  "test-jti-123",
		},
		{
			name: "always false - empty jti",
			jti:  "",
		},
		{
			name: "always false - any jti",
			jti:  "any-random-jti",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			revoked, err := bl.IsRevoked(ctx, tt.jti)
			assert.NoError(t, err, "NoopBlacklist.IsRevoked should never error")
			assert.False(t, revoked, "NoopBlacklist.IsRevoked should always return false")
		})
	}
}

func TestNoopBlacklist_Cleanup(t *testing.T) {
	bl := &NoopBlacklist{}
	ctx := context.Background()

	err := bl.Cleanup(ctx)
	assert.NoError(t, err, "NoopBlacklist.Cleanup should always return nil")

	// Test multiple calls
	err = bl.Cleanup(ctx)
	assert.NoError(t, err, "NoopBlacklist.Cleanup should be idempotent")
}

func TestNoopBlacklist_Interface(t *testing.T) {
	// Verify that NoopBlacklist implements Blacklist interface
	var _ Blacklist = (*NoopBlacklist)(nil)
}

// ============================================================================
// Tests for PostgresBlacklist (with mocked DB using go-sqlmock)
// ============================================================================

func setupMockDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	db := bun.NewDB(mockDB, pgdialect.New())
	return db, mock
}

func TestNewPostgresBlacklist(t *testing.T) {
	bl := NewPostgresBlacklist(nil)
	assert.NotNil(t, bl, "NewPostgresBlacklist should return non-nil instance")
}

func TestPostgresBlacklist_Interface(t *testing.T) {
	// Verify that PostgresBlacklist implements Blacklist interface
	var _ Blacklist = (*PostgresBlacklist)(nil)
}

func TestPostgresBlacklist_Revoke(t *testing.T) {
	tests := []struct {
		name      string
		jti       string
		expiresAt time.Time
		mockSetup func(mock sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name:      "success - revoke token",
			jti:       "test-jti-123",
			expiresAt: time.Now().Add(time.Hour),
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			wantErr: false,
		},
		{
			name:      "success - revoke with conflict (already exists)",
			jti:       "existing-jti",
			expiresAt: time.Now().Add(time.Hour),
			mockSetup: func(mock sqlmock.Sqlmock) {
				// ON CONFLICT DO NOTHING returns 0 rows affected
				mock.ExpectExec("INSERT INTO").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false,
		},
		{
			name:      "error - database failure",
			jti:       "test-jti-456",
			expiresAt: time.Now().Add(time.Hour),
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("INSERT INTO").
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupMockDB(t)
			defer db.Close()

			tt.mockSetup(mock)

			bl := NewPostgresBlacklist(db)
			err := bl.Revoke(context.Background(), tt.jti, tt.expiresAt)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPostgresBlacklist_IsRevoked(t *testing.T) {
	tests := []struct {
		name        string
		jti         string
		mockSetup   func(mock sqlmock.Sqlmock)
		wantRevoked bool
		wantErr     bool
	}{
		{
			name: "token is revoked",
			jti:  "revoked-jti",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
			},
			wantRevoked: true,
			wantErr:     false,
		},
		{
			name: "token is not revoked",
			jti:  "valid-jti",
			mockSetup: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery("SELECT").WillReturnRows(rows)
			},
			wantRevoked: false,
			wantErr:     false,
		},
		{
			name: "database error",
			jti:  "error-jti",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("SELECT").WillReturnError(assert.AnError)
			},
			wantRevoked: false,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupMockDB(t)
			defer db.Close()

			tt.mockSetup(mock)

			bl := NewPostgresBlacklist(db)
			revoked, err := bl.IsRevoked(context.Background(), tt.jti)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantRevoked, revoked)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPostgresBlacklist_Cleanup(t *testing.T) {
	tests := []struct {
		name      string
		mockSetup func(mock sqlmock.Sqlmock)
		wantErr   bool
	}{
		{
			name: "success - cleanup expired tokens",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM").
					WillReturnResult(sqlmock.NewResult(0, 5)) // 5 rows deleted
			},
			wantErr: false,
		},
		{
			name: "success - no expired tokens",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM").
					WillReturnResult(sqlmock.NewResult(0, 0)) // 0 rows deleted
			},
			wantErr: false,
		},
		{
			name: "error - database failure",
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("DELETE FROM").
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := setupMockDB(t)
			defer db.Close()

			tt.mockSetup(mock)

			bl := NewPostgresBlacklist(db)
			err := bl.Cleanup(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPostgresBlacklist_RevokeAndCheck(t *testing.T) {
	// Test the full flow: revoke a token and then check if it's revoked
	db, mock := setupMockDB(t)
	defer db.Close()

	jti := "test-jti-full-flow"

	// Expect insert for revoke
	mock.ExpectExec("INSERT INTO").
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect select for IsRevoked
	rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	bl := NewPostgresBlacklist(db)
	ctx := context.Background()

	// Revoke the token
	err := bl.Revoke(ctx, jti, time.Now().Add(time.Hour))
	assert.NoError(t, err)

	// Check if revoked
	revoked, err := bl.IsRevoked(ctx, jti)
	assert.NoError(t, err)
	assert.True(t, revoked)

	assert.NoError(t, mock.ExpectationsWereMet())
}
