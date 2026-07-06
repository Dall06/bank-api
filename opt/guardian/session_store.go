package guardian

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// SessionStore defines the interface for managing user sessions
// Multiple sessions per user are allowed
type SessionStore interface {
	// AddSession registers a new session (allows multiple per user)
	AddSession(ctx context.Context, userID, jti string, expiresAt time.Time) error

	// IsActive checks if a token JTI exists and is not expired
	IsActive(ctx context.Context, jti string) (bool, error)

	// InvalidateSession removes a specific session by JTI (single logout)
	InvalidateSession(ctx context.Context, jti string) error

	// InvalidateAll removes all sessions for a user (logout everywhere)
	InvalidateAll(ctx context.Context, userID string) error

	// InvalidateAllByCompany removes all sessions for all active staff of a company
	InvalidateAllByCompany(ctx context.Context, companyID string) error

	// Cleanup removes expired sessions
	Cleanup(ctx context.Context) error
}

// userSessionDB is the database model for user sessions.
// Matches scripts/init-db.sql: CREATE TABLE user_sessions (id, user_id, token_jti, expires_at, created_at)
type userSessionDB struct {
	bun.BaseModel `bun:"table:user_sessions"`

	ID        string    `bun:"id,pk,type:uuid"`
	UserID    string    `bun:"user_id,type:uuid,notnull"`
	TokenJTI  string    `bun:"token_jti,type:uuid,notnull"`
	ExpiresAt time.Time `bun:"expires_at,notnull"`
	CreatedAt time.Time `bun:"created_at,notnull,default:current_timestamp"`
}

// PostgresSessionStore implements SessionStore using PostgreSQL
type PostgresSessionStore struct {
	db bun.IDB
}

// NewPostgresSessionStore creates a new PostgreSQL-backed session store
func NewPostgresSessionStore(db bun.IDB) *PostgresSessionStore {
	return &PostgresSessionStore{db: db}
}

// AddSession registers a new session for a user
func (s *PostgresSessionStore) AddSession(ctx context.Context, userID, jti string, expiresAt time.Time) error {
	session := &userSessionDB{
		ID:        uuid.New().String(),
		UserID:    userID,
		TokenJTI:  jti,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	_, err := s.db.NewInsert().
		Model(session).
		Exec(ctx)

	return err
}

// IsActive checks if a token JTI exists and is not expired
func (s *PostgresSessionStore) IsActive(ctx context.Context, jti string) (bool, error) {
	exists, err := s.db.NewSelect().
		Model((*userSessionDB)(nil)).
		Where("token_jti = ?", jti).
		Where("expires_at > ?", time.Now()).
		Exists(ctx)

	return exists, err
}

// InvalidateSession removes a specific session by JTI
func (s *PostgresSessionStore) InvalidateSession(ctx context.Context, jti string) error {
	_, err := s.db.NewDelete().
		Model((*userSessionDB)(nil)).
		Where("token_jti = ?", jti).
		Exec(ctx)

	return err
}

// InvalidateAll removes all sessions for a user
func (s *PostgresSessionStore) InvalidateAll(ctx context.Context, userID string) error {
	_, err := s.db.NewDelete().
		Model((*userSessionDB)(nil)).
		Where("user_id = ?", userID).
		Exec(ctx)

	return err
}

// InvalidateAllByCompany removes all sessions for all active staff of a company.
// Joins user_sessions with staff table to find all users belonging to the company.
func (s *PostgresSessionStore) InvalidateAllByCompany(ctx context.Context, companyID string) error {
	subq := s.db.NewSelect().
		TableExpr("staff").
		Column("user_id").
		Where("company_id = ?", companyID).
		Where("is_active = ?", true)

	_, err := s.db.NewDelete().
		Model((*userSessionDB)(nil)).
		Where("user_id IN (?)", subq).
		Exec(ctx)
	return err
}

// Cleanup removes expired sessions
func (s *PostgresSessionStore) Cleanup(ctx context.Context) error {
	_, err := s.db.NewDelete().
		Model((*userSessionDB)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)

	return err
}

// NoopSessionStore is a no-op implementation of SessionStore (for testing)
type NoopSessionStore struct{}

// AddSession does nothing
func (s *NoopSessionStore) AddSession(ctx context.Context, userID, jti string, expiresAt time.Time) error {
	return nil
}

// IsActive always returns true (no session validation)
func (s *NoopSessionStore) IsActive(ctx context.Context, jti string) (bool, error) {
	return true, nil
}

// InvalidateSession does nothing
func (s *NoopSessionStore) InvalidateSession(ctx context.Context, jti string) error {
	return nil
}

// InvalidateAll does nothing
func (s *NoopSessionStore) InvalidateAll(ctx context.Context, userID string) error {
	return nil
}

// InvalidateAllByCompany does nothing
func (s *NoopSessionStore) InvalidateAllByCompany(ctx context.Context, companyID string) error {
	return nil
}

// Cleanup does nothing
func (s *NoopSessionStore) Cleanup(ctx context.Context) error {
	return nil
}
