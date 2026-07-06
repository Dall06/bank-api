package guardian

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func setupSessionMockDB(t *testing.T) (*bun.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	db := bun.NewDB(mockDB, pgdialect.New())
	return db, mock
}

func TestNewPostgresSessionStore(t *testing.T) {
	store := NewPostgresSessionStore(nil)
	assert.NotNil(t, store)
}

func TestPostgresSessionStore_AddSession(t *testing.T) {
	db, mock := setupSessionMockDB(t)
	defer db.Close()
	store := NewPostgresSessionStore(db)

	expiresAt := time.Now().Add(time.Hour)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
		err := store.AddSession(ctx, "user-1", "jti-1", expiresAt)
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectExec("INSERT").WillReturnError(errors.New("db error"))
		err := store.AddSession(ctx, "user-1", "jti-1", expiresAt)
		assert.Error(t, err)
	})
}

func TestPostgresSessionStore_IsActive(t *testing.T) {
	db, mock := setupSessionMockDB(t)
	defer db.Close()
	store := NewPostgresSessionStore(db)
	ctx := context.Background()

	t.Run("active", func(t *testing.T) {
		mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
		active, err := store.IsActive(ctx, "jti-1")
		assert.NoError(t, err)
		assert.True(t, active)
	})

	t.Run("error", func(t *testing.T) {
		mock.ExpectQuery("SELECT").WillReturnError(errors.New("db error"))
		_, err := store.IsActive(ctx, "jti-1")
		assert.Error(t, err)
	})
}

func TestPostgresSessionStore_InvalidateSession(t *testing.T) {
	db, mock := setupSessionMockDB(t)
	defer db.Close()
	store := NewPostgresSessionStore(db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(1, 1))
		err := store.InvalidateSession(ctx, "jti-1")
		assert.NoError(t, err)
	})
}

func TestPostgresSessionStore_InvalidateAll(t *testing.T) {
	db, mock := setupSessionMockDB(t)
	defer db.Close()
	store := NewPostgresSessionStore(db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(1, 1))
		err := store.InvalidateAll(ctx, "user-1")
		assert.NoError(t, err)
	})
}

func TestPostgresSessionStore_InvalidateAllByCompany(t *testing.T) {
	db, mock := setupSessionMockDB(t)
	defer db.Close()
	store := NewPostgresSessionStore(db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(1, 1))
		err := store.InvalidateAllByCompany(ctx, "company-1")
		assert.NoError(t, err)
	})
}

func TestPostgresSessionStore_Cleanup(t *testing.T) {
	db, mock := setupSessionMockDB(t)
	defer db.Close()
	store := NewPostgresSessionStore(db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("DELETE").WillReturnResult(sqlmock.NewResult(1, 1))
		err := store.Cleanup(ctx)
		assert.NoError(t, err)
	})
}

func TestNoopSessionStore(t *testing.T) {
	store := &NoopSessionStore{}
	ctx := context.Background()

	err := store.AddSession(ctx, "u", "j", time.Now())
	assert.NoError(t, err)

	active, err := store.IsActive(ctx, "j")
	assert.NoError(t, err)
	assert.True(t, active)

	err = store.InvalidateSession(ctx, "j")
	assert.NoError(t, err)

	err = store.InvalidateAll(ctx, "u")
	assert.NoError(t, err)

	err = store.InvalidateAllByCompany(ctx, "c")
	assert.NoError(t, err)

	err = store.Cleanup(ctx)
	assert.NoError(t, err)
}
