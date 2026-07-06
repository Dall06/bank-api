package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

// Connect establishes a connection to the database with retry logic
func Connect(databaseURI string, timeoutSec int) (*bun.DB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(databaseURI)))

	// Try initial connection
	if err := sqldb.Ping(); err == nil {
		db := bun.NewDB(sqldb, pgdialect.New())
		return db, nil
	}

	// Retry with timeout
	start := time.Now()
	timeout := time.Duration(timeoutSec) * time.Second
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := sqldb.Ping(); err == nil {
			db := bun.NewDB(sqldb, pgdialect.New())
			return db, nil
		}

		if time.Since(start) > timeout {
			break
		}
	}

	return nil, fmt.Errorf("failed to connect to database after %d seconds", timeoutSec)
}

// AutoMigrate crea la tabla en la base de datos para el tipo de modelo T dado si no existe.
func AutoMigrate[T any](db *bun.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var model T
	_, err := db.NewCreateTable().Model(&model).IfNotExists().Exec(ctx)
	return err
}
