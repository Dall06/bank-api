package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client define la interfaz/puerto con las operaciones requeridas de Redis.
type Client interface {
	Connect() error
	Disconnect() error
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error)
	Del(ctx context.Context, keys ...string) error
}

// RedisClient es el adaptador concreto que implementa Client utilizando go-redis.
type RedisClient struct {
	addr       string
	timeoutSec int
	rdb        *redis.Client
}

// NewRedisClient crea una nueva instancia de RedisClient con sus propiedades configuradas.
func NewRedisClient(addr string, timeoutSec int) *RedisClient {
	return &RedisClient{
		addr:       addr,
		timeoutSec: timeoutSec,
	}
}

// Connect conecta a Redis con verificación por Ping y reintentos.
func (c *RedisClient) Connect() error {
	rdb := redis.NewClient(&redis.Options{
		Addr: c.addr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.timeoutSec)*time.Second)
	defer cancel()

	// Ping inicial
	if err := rdb.Ping(ctx).Err(); err == nil {
		c.rdb = rdb
		return nil
	}

	// Reintentos si no está disponible de inmediato
	start := time.Now()
	timeout := time.Duration(c.timeoutSec) * time.Second
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 1*time.Second)
		err := rdb.Ping(pingCtx).Err()
		pingCancel()
		
		if err == nil {
			c.rdb = rdb
			return nil
		}

		if time.Since(start) > timeout {
			rdb.Close()
			return fmt.Errorf("failed to connect to redis after %d seconds: %w", c.timeoutSec, err)
		}
	}
}

// Disconnect cierra la conexión del cliente con Redis.
func (c *RedisClient) Disconnect() error {
	if c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}

// Get obtiene el valor de una llave.
func (c *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

// Set guarda una llave. Si isNX es true, actúa como SetNX (solo escribe si no existe).
// Retorna true si la llave fue escrita, y false si no se escribió o hubo error.
func (c *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error) {
	if isNX {
		return c.rdb.SetNX(ctx, key, value, expiration).Result()
	}
	
	err := c.rdb.Set(ctx, key, value, expiration).Err()
	if err != nil {
		return false, err
	}
	return true, nil
}

// Del elimina una o más llaves.
func (c *RedisClient) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}
