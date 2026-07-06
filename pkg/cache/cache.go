package cache

import (
	"context"
	"encoding/json"
	"time"

	"bank-api/opt/redis"
	"bank-api/pkg/errs"
)

type Manager struct {
	rdb redis.Client
}

func NewManager(rdb redis.Client) *Manager {
	return &Manager{rdb: rdb}
}

// GetIdempotency extrae los bytes crudos directamente de Redis de forma segura.
// Retorna (bytes, encontrado, error)
func (m *Manager) GetIdempotency(ctx context.Context, fullKey string) ([]byte, bool, error) {
	if fullKey == "" {
		return nil, false, nil
	}
	cachedVal, err := m.rdb.Get(ctx, fullKey)
	if err != nil {
		if err.Error() == "redis: nil" {
			return nil, false, nil
		}
		return nil, false, errs.InternalError("failed to query idempotency cache: %v", err)
	}
	if cachedVal == "" {
		return nil, false, nil
	}

	return []byte(cachedVal), true, nil
}

// Lock maneja los errores transaccionales de Redis devolviendo un InternalError descriptivo.
func (m *Manager) Lock(ctx context.Context, fullLockKey string) (func(), error) {
	if fullLockKey == "" {
		return func() {}, nil
	}

	locked, err := m.rdb.Set(ctx, fullLockKey, "processing", 1*time.Minute, true)
	if err != nil {
		return nil, errs.InternalError("failed to acquire distributed lock: %v", err)
	}
	if !locked {
		return nil, errs.ConflictError("transaction is already in progress")
	}

	cleanup := func() {
		_ = m.rdb.Del(context.Background(), fullLockKey)
	}
	return cleanup, nil
}

// Save propaga el error si la escritura en Redis o si la serialización fallan.
func (m *Manager) Save(ctx context.Context, fullKey string, data any, ttl time.Duration) error {
	if fullKey == "" || data == nil {
		return nil
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return errs.InternalError("failed to marshal data for cache: %v", err)
	}

	_, err = m.rdb.Set(ctx, fullKey, string(bytes), ttl, false)
	if err != nil {
		return errs.InternalError("failed to write data to cache: %v", err)
	}
	return nil
}

// GetRaw lee directamente de Redis y maneja la desconexión o fallos del nodo.
func (m *Manager) GetRaw(ctx context.Context, fullKey string) (string, error) {
	val, err := m.rdb.Get(ctx, fullKey)
	if err != nil {
		return "", errs.InternalError("failed to read raw value from cache: %v", err)
	}
	return val, nil
}
