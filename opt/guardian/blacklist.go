package guardian

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

// Blacklist defines the interface for token revocation
type Blacklist interface {
	// Revoke marks a token as revoked
	Revoke(ctx context.Context, jti string, expiresAt time.Time) error

	// IsRevoked checks if a token has been revoked
	IsRevoked(ctx context.Context, jti string) (bool, error)

	// Cleanup removes expired revoked tokens
	Cleanup(ctx context.Context) error
}

// revokedTokenDB is the database model for revoked tokens
type revokedTokenDB struct {
	bun.BaseModel `bun:"table:revoked_tokens"`

	JTI       string    `bun:"jti,pk,type:uuid"`
	RevokedAt time.Time `bun:"revoked_at,notnull,default:current_timestamp"`
	ExpiresAt time.Time `bun:"expires_at,notnull"`
}

// PostgresBlacklist implements Blacklist using PostgreSQL
type PostgresBlacklist struct {
	db bun.IDB
}

// NewPostgresBlacklist creates a new PostgreSQL-backed blacklist
func NewPostgresBlacklist(db bun.IDB) *PostgresBlacklist {
	return &PostgresBlacklist{db: db}
}

// Revoke marks a token as revoked
func (b *PostgresBlacklist) Revoke(ctx context.Context, jti string, expiresAt time.Time) error {
	token := &revokedTokenDB{
		JTI:       jti,
		RevokedAt: time.Now(),
		ExpiresAt: expiresAt,
	}

	_, err := b.db.NewInsert().
		Model(token).
		On("CONFLICT (jti) DO NOTHING").
		Exec(ctx)

	return err
}

// IsRevoked checks if a token has been revoked
func (b *PostgresBlacklist) IsRevoked(ctx context.Context, jti string) (bool, error) {
	exists, err := b.db.NewSelect().
		Model((*revokedTokenDB)(nil)).
		Where("jti = ?", jti).
		Exists(ctx)

	return exists, err
}

// Cleanup removes expired revoked tokens
func (b *PostgresBlacklist) Cleanup(ctx context.Context) error {
	_, err := b.db.NewDelete().
		Model((*revokedTokenDB)(nil)).
		Where("expires_at < ?", time.Now()).
		Exec(ctx)

	return err
}

// NoopBlacklist is a no-op implementation of Blacklist (for testing or when revocation is not needed)
type NoopBlacklist struct{}

// Revoke does nothing
func (b *NoopBlacklist) Revoke(ctx context.Context, jti string, expiresAt time.Time) error {
	return nil
}

// IsRevoked always returns false
func (b *NoopBlacklist) IsRevoked(ctx context.Context, jti string) (bool, error) {
	return false, nil
}

// Cleanup does nothing
func (b *NoopBlacklist) Cleanup(ctx context.Context) error {
	return nil
}
