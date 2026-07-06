package redis_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	myredis "bank-api/opt/redis"
)

func TestRedisClient(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("unexpected miniredis err: %v", err)
	}
	defer s.Close()

	tests := []struct {
		name       string
		addr       string
		timeoutSec int
		setup      func(*myredis.RedisClient)
		run        func(*testing.T, *myredis.RedisClient)
	}{
		{
			name:       "Connect successfully",
			addr:       s.Addr(),
			timeoutSec: 1,
			setup:      func(c *myredis.RedisClient) {},
			run: func(t *testing.T, c *myredis.RedisClient) {
				err := c.Connect()
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				c.Disconnect()
			},
		},
		{
			name:       "Connect failure with timeout",
			addr:       "invalid_addr:1234",
			timeoutSec: 1,
			setup:      func(c *myredis.RedisClient) {},
			run: func(t *testing.T, c *myredis.RedisClient) {
				err := c.Connect()
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				c.Disconnect()
			},
		},
		{
			name:       "Connect with retry success",
			addr:       s.Addr(),
			timeoutSec: 1,
			setup:      func(c *myredis.RedisClient) {
				// We pause miniredis to make the first ping fail
				s.Close()
			},
			run: func(t *testing.T, c *myredis.RedisClient) {
				// Restart miniredis after 1.5 seconds so retry succeeds
				go func() {
					time.Sleep(1500 * time.Millisecond)
					s.Restart()
				}()
				err := c.Connect()
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				c.Disconnect()
			},
		},
		{
			name:       "Set and Get",
			addr:       s.Addr(),
			timeoutSec: 1,
			setup: func(c *myredis.RedisClient) {
				c.Connect()
			},
			run: func(t *testing.T, c *myredis.RedisClient) {
				ctx := context.Background()
				ok, err := c.Set(ctx, "key1", "value1", time.Minute, false)
				if err != nil || !ok {
					t.Errorf("expected success, got err=%v ok=%v", err, ok)
				}

				val, err := c.Get(ctx, "key1")
				if err != nil || val != "value1" {
					t.Errorf("expected value1, got %v (err: %v)", val, err)
				}
				
				err = c.Del(ctx, "key1")
				if err != nil {
					t.Errorf("expected no error on del, got %v", err)
				}
				c.Disconnect()
			},
		},
		{
			name:       "Set NX",
			addr:       s.Addr(),
			timeoutSec: 1,
			setup: func(c *myredis.RedisClient) {
				c.Connect()
			},
			run: func(t *testing.T, c *myredis.RedisClient) {
				ctx := context.Background()
				ok, err := c.Set(ctx, "key2", "value2", time.Minute, true)
				if err != nil || !ok {
					t.Errorf("expected success on Set NX, got err=%v ok=%v", err, ok)
				}

				ok, err = c.Set(ctx, "key2", "value2", time.Minute, true)
				if err != nil || ok {
					t.Errorf("expected ok=false, err=nil on Set NX existing key, got err=%v ok=%v", err, ok)
				}

				c.Disconnect()
			},
		},
		{
			name:       "Set error",
			addr:       s.Addr(),
			timeoutSec: 1,
			setup: func(c *myredis.RedisClient) {
				c.Connect()
				c.Disconnect() // close to force error
			},
			run: func(t *testing.T, c *myredis.RedisClient) {
				ctx := context.Background()
				ok, err := c.Set(ctx, "key3", "value3", time.Minute, false)
				if err == nil {
					t.Errorf("expected error on closed client, got ok=%v", ok)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := myredis.NewRedisClient(tt.addr, tt.timeoutSec)
			tt.setup(client)
			tt.run(t, client)
		})
	}
}
