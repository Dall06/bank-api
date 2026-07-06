package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockRedisClient struct {
	getFn func(ctx context.Context, key string) (string, error)
	setFn func(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error)
	delFn func(ctx context.Context, keys ...string) error
}

func (m *mockRedisClient) Connect() error    { return nil }
func (m *mockRedisClient) Disconnect() error { return nil }
func (m *mockRedisClient) Get(ctx context.Context, key string) (string, error) {
	if m.getFn != nil {
		return m.getFn(ctx, key)
	}
	return "", nil
}
func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error) {
	if m.setFn != nil {
		return m.setFn(ctx, key, value, expiration, isNX)
	}
	return false, nil
}
func (m *mockRedisClient) Del(ctx context.Context, keys ...string) error {
	if m.delFn != nil {
		return m.delFn(ctx, keys...)
	}
	return nil
}

func TestManager_GetIdempotency(t *testing.T) {
	tests := []struct {
		name      string
		fullKey   string
		mockGet   func(ctx context.Context, key string) (string, error)
		wantBytes []byte
		wantFound bool
		wantErr   bool
	}{
		{
			name:      "empty key",
			fullKey:   "",
			wantBytes: nil,
			wantFound: false,
			wantErr:   false,
		},
		{
			name:    "redis nil error (not found)",
			fullKey: "key1",
			mockGet: func(ctx context.Context, key string) (string, error) {
				return "", errors.New("redis: nil")
			},
			wantBytes: nil,
			wantFound: false,
			wantErr:   false,
		},
		{
			name:    "other redis error",
			fullKey: "key2",
			mockGet: func(ctx context.Context, key string) (string, error) {
				return "", errors.New("some error")
			},
			wantBytes: nil,
			wantFound: false,
			wantErr:   true,
		},
		{
			name:    "empty value in cache",
			fullKey: "key3",
			mockGet: func(ctx context.Context, key string) (string, error) {
				return "", nil
			},
			wantBytes: nil,
			wantFound: false,
			wantErr:   false,
		},
		{
			name:    "found valid value",
			fullKey: "key4",
			mockGet: func(ctx context.Context, key string) (string, error) {
				return "some-data", nil
			},
			wantBytes: []byte("some-data"),
			wantFound: true,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(&mockRedisClient{getFn: tt.mockGet})
			gotBytes, gotFound, err := m.GetIdempotency(context.Background(), tt.fullKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetIdempotency() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotFound != tt.wantFound {
				t.Errorf("GetIdempotency() gotFound = %v, want %v", gotFound, tt.wantFound)
			}
			if string(gotBytes) != string(tt.wantBytes) {
				t.Errorf("GetIdempotency() gotBytes = %v, want %v", string(gotBytes), string(tt.wantBytes))
			}
		})
	}
}

func TestManager_Lock(t *testing.T) {
	tests := []struct {
		name        string
		fullLockKey string
		mockSet     func(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error)
		wantLocked  bool
		wantErr     bool
	}{
		{
			name:        "empty lock key",
			fullLockKey: "",
			wantLocked:  true, // returns dummy func, no error
			wantErr:     false,
		},
		{
			name:        "redis set error",
			fullLockKey: "lock1",
			mockSet: func(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error) {
				return false, errors.New("set error")
			},
			wantLocked: false,
			wantErr:    true,
		},
		{
			name:        "lock not acquired (conflict)",
			fullLockKey: "lock2",
			mockSet: func(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error) {
				return false, nil
			},
			wantLocked: false,
			wantErr:    true,
		},
		{
			name:        "lock acquired successfully",
			fullLockKey: "lock3",
			mockSet: func(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error) {
				return true, nil
			},
			wantLocked: true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(&mockRedisClient{setFn: tt.mockSet})
			cleanup, err := m.Lock(context.Background(), tt.fullLockKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("Lock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantLocked && cleanup == nil {
				t.Errorf("Lock() expected cleanup func, got nil")
			}
			if !tt.wantLocked && cleanup != nil {
				t.Errorf("Lock() expected nil cleanup func, got one")
			}
			if cleanup != nil {
				cleanup() // test that it doesn't panic
			}
		})
	}
}

func TestManager_Save(t *testing.T) {
	tests := []struct {
		name    string
		fullKey string
		data    any
		ttl     time.Duration
		mockSet func(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error)
		wantErr bool
	}{
		{
			name:    "empty key",
			fullKey: "",
			data:    "data",
			wantErr: false,
		},
		{
			name:    "nil data",
			fullKey: "key1",
			data:    nil,
			wantErr: false,
		},
		{
			name:    "marshal error",
			fullKey: "key2",
			data:    make(chan int), // unmarshallable
			wantErr: true,
		},
		{
			name:    "redis set error",
			fullKey: "key3",
			data:    "data",
			mockSet: func(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error) {
				return false, errors.New("set error")
			},
			wantErr: true,
		},
		{
			name:    "success",
			fullKey: "key4",
			data:    "data",
			mockSet: func(ctx context.Context, key string, value interface{}, expiration time.Duration, isNX bool) (bool, error) {
				return true, nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(&mockRedisClient{setFn: tt.mockSet})
			err := m.Save(context.Background(), tt.fullKey, tt.data, tt.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_GetRaw(t *testing.T) {
	tests := []struct {
		name    string
		fullKey string
		mockGet func(ctx context.Context, key string) (string, error)
		wantVal string
		wantErr bool
	}{
		{
			name:    "redis get error",
			fullKey: "key1",
			mockGet: func(ctx context.Context, key string) (string, error) {
				return "", errors.New("get error")
			},
			wantVal: "",
			wantErr: true,
		},
		{
			name:    "success",
			fullKey: "key2",
			mockGet: func(ctx context.Context, key string) (string, error) {
				return "raw-data", nil
			},
			wantVal: "raw-data",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(&mockRedisClient{getFn: tt.mockGet})
			val, err := m.GetRaw(context.Background(), tt.fullKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRaw() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if val != tt.wantVal {
				t.Errorf("GetRaw() val = %v, want %v", val, tt.wantVal)
			}
		})
	}
}
