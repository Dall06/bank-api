package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"bank-api/env/config"
	"bank-api/pkg/errs"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// TenantPoolManager manages database connection pools for multiple tenants
type TenantPoolManager struct {
	cfg   *config.Config
	pools map[string]*bun.DB
	mu    sync.RWMutex
}

// NewTenantPoolManager creates a new tenant pool manager
func NewTenantPoolManager(cfg *config.Config) *TenantPoolManager {
	return &TenantPoolManager{
		cfg:   cfg,
		pools: make(map[string]*bun.DB),
	}
}

// GetOrCreate returns an existing connection or creates a new one for the tenant
func (m *TenantPoolManager) GetOrCreate(ctx context.Context, slug string) (*bun.DB, error) {
	// Fast path: read with RLock
	m.mu.RLock()
	if pool, exists := m.pools[slug]; exists {
		m.mu.RUnlock()
		// Verify connection is still alive
		if err := pool.PingContext(ctx); err == nil {
			return pool, nil
		}
		// Dead connection, need to recreate
	}
	if _, exists := m.pools[slug]; !exists {
		m.mu.RUnlock()
	}

	// Slow path: create with Lock
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring lock
	if pool, exists := m.pools[slug]; exists {
		if err := pool.PingContext(ctx); err == nil {
			return pool, nil
		}
		// Close dead connection
		if closeErr := pool.Close(); closeErr != nil {
			slog.Warn("failed to close dead pool connection", "slug", slug, "error", closeErr)
		}
		delete(m.pools, slug)
	}

	// Create new connection
	uri := m.buildTenantURI(slug)
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(uri)))

	// Configure pool
	sqldb.SetMaxOpenConns(m.cfg.MaxPoolConnections)
	sqldb.SetMaxIdleConns(m.cfg.MaxIdleConnections)
	sqldb.SetConnMaxLifetime(time.Duration(m.cfg.ConnectionMaxLifetime) * time.Second)

	// Verify connection
	if err := sqldb.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("%w: %s", errs.InternalError("connection failed"), err.Error())
	}

	db := bun.NewDB(sqldb, pgdialect.New())
	m.pools[slug] = db

	return db, nil
}

// Close closes the connection for the specified tenant
func (m *TenantPoolManager) Close(slug string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pool, exists := m.pools[slug]; exists {
		err := pool.Close()
		delete(m.pools, slug)
		return err
	}
	return nil
}

// CloseAll closes all tenant connections
func (m *TenantPoolManager) CloseAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for slug, pool := range m.pools {
		if err := pool.Close(); err != nil {
			lastErr = err
		}
		delete(m.pools, slug)
	}
	return lastErr
}

// Health returns the health status of all tenant connections
func (m *TenantPoolManager) Health(ctx context.Context) map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := make(map[string]bool)
	for slug, pool := range m.pools {
		health[slug] = pool.PingContext(ctx) == nil
	}
	return health
}

// Count returns the number of active connections
func (m *TenantPoolManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.pools)
}

// buildTenantURI builds the database URI for a tenant
func (m *TenantPoolManager) buildTenantURI(slug string) string {
	dbName := fmt.Sprintf("gym_%s", strings.ReplaceAll(slug, "-", "_"))
	return strings.Replace(m.cfg.TenantDatabaseURITemplate, "{slug}", dbName, 1)
}
