package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"bank-api/opt/middlewares"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultThrottleConfig(t *testing.T) {
	config := middlewares.DefaultThrottleConfig()

	assert.Equal(t, 5, config.RequestsPerMinute)
	assert.Equal(t, 10, config.BurstSize)
	assert.Equal(t, 5*time.Minute, config.CleanupInterval)
}

func TestNewThrottle(t *testing.T) {
	config := middlewares.ThrottleConfig{
		RequestsPerMinute: 10,
		BurstSize:         5,
		CleanupInterval:   1 * time.Second,
	}

	throttle := middlewares.NewThrottle(config)
	defer throttle.Stop()

	assert.NotNil(t, throttle)
}

func TestThrottle_Allow(t *testing.T) {
	tests := []struct {
		name              string
		requestsPerMinute int
		burstSize         int
		numRequests       int
		expectedAllowed   int
	}{
		{
			name:              "allows requests under limit",
			requestsPerMinute: 5,
			burstSize:         0,
			numRequests:       3,
			expectedAllowed:   3,
		},
		{
			name:              "allows burst requests",
			requestsPerMinute: 5,
			burstSize:         5,
			numRequests:       10,
			expectedAllowed:   10,
		},
		{
			name:              "blocks requests over limit",
			requestsPerMinute: 3,
			burstSize:         2,
			numRequests:       10,
			expectedAllowed:   5, // 3 + 2 burst
		},
		{
			name:              "single request always allowed",
			requestsPerMinute: 1,
			burstSize:         0,
			numRequests:       1,
			expectedAllowed:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := middlewares.ThrottleConfig{
				RequestsPerMinute: tt.requestsPerMinute,
				BurstSize:         tt.burstSize,
				CleanupInterval:   1 * time.Hour,
			}

			throttle := middlewares.NewThrottle(config)
			defer throttle.Stop()

			allowed := 0
			for i := 0; i < tt.numRequests; i++ {
				if throttle.Allow("test-key") {
					allowed++
				}
			}

			assert.Equal(t, tt.expectedAllowed, allowed)
		})
	}
}

func TestThrottle_Remaining(t *testing.T) {
	config := middlewares.ThrottleConfig{
		RequestsPerMinute: 10,
		BurstSize:         0,
		CleanupInterval:   1 * time.Hour,
	}

	throttle := middlewares.NewThrottle(config)
	defer throttle.Stop()

	// Initial state - no requests made
	assert.Equal(t, 10, throttle.Remaining("test-key"))

	// After some requests
	throttle.Allow("test-key")
	throttle.Allow("test-key")
	throttle.Allow("test-key")

	assert.Equal(t, 7, throttle.Remaining("test-key"))

	// Different key should have full remaining
	assert.Equal(t, 10, throttle.Remaining("other-key"))
}

func TestThrottle_DifferentKeys(t *testing.T) {
	config := middlewares.ThrottleConfig{
		RequestsPerMinute: 2,
		BurstSize:         0,
		CleanupInterval:   1 * time.Hour,
	}

	throttle := middlewares.NewThrottle(config)
	defer throttle.Stop()

	// Key 1 uses all requests
	assert.True(t, throttle.Allow("key1"))
	assert.True(t, throttle.Allow("key1"))
	assert.False(t, throttle.Allow("key1"))

	// Key 2 should still have full quota
	assert.True(t, throttle.Allow("key2"))
	assert.True(t, throttle.Allow("key2"))
	assert.False(t, throttle.Allow("key2"))
}

func TestThrottle_ConcurrentAccess(t *testing.T) {
	config := middlewares.ThrottleConfig{
		RequestsPerMinute: 100,
		BurstSize:         50,
		CleanupInterval:   1 * time.Hour,
	}

	throttle := middlewares.NewThrottle(config)
	defer throttle.Stop()

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if throttle.Allow("concurrent-key") {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Should have allowed exactly 150 (100 + 50 burst)
	assert.Equal(t, 150, allowed)
}

func TestNewThrottleMiddleware(t *testing.T) {
	config := middlewares.ThrottleConfig{
		RequestsPerMinute: 3,
		BurstSize:         0,
		CleanupInterval:   1 * time.Hour,
	}

	keyFunc := func(c echo.Context) string {
		return c.RealIP()
	}

	middleware := middlewares.NewThrottleMiddleware(config, keyFunc)

	// Make requests
	for i := 0; i < 5; i++ {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:1234"
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler := middleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "OK")
		})

		err := handler(c)

		if i < 3 {
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
		} else {
			require.NoError(t, err)
			assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		}
	}
}

func TestNewThrottleMiddleware_DifferentIPs(t *testing.T) {
	config := middlewares.ThrottleConfig{
		RequestsPerMinute: 2,
		BurstSize:         0,
		CleanupInterval:   1 * time.Hour,
	}

	middleware := middlewares.NewThrottleMiddleware(config, func(c echo.Context) string {
		return c.RealIP()
	})

	ips := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}

	for _, ip := range ips {
		for i := 0; i < 3; i++ {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = ip + ":1234"
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := middleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "OK")
			})

			err := handler(c)

			if i < 2 {
				require.NoError(t, err, "IP %s request %d should succeed", ip, i)
				assert.Equal(t, http.StatusOK, rec.Code)
			} else {
				require.NoError(t, err)
				assert.Equal(t, http.StatusTooManyRequests, rec.Code, "IP %s request %d should be blocked", ip, i)
			}
		}
	}
}

func TestIPKeyFunc(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		expected   string
	}{
		{
			name:       "extracts IP from remote addr",
			remoteAddr: "192.168.1.1:1234",
			expected:   "192.168.1.1",
		},
		{
			name:       "handles IPv6",
			remoteAddr: "[::1]:1234",
			expected:   "::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			key := middlewares.IPKeyFunc(c)
			assert.Equal(t, tt.expected, key)
		})
	}
}

func TestNewIPThrottle(t *testing.T) {
	config := middlewares.ThrottleConfig{
		RequestsPerMinute: 5,
		BurstSize:         2,
		CleanupInterval:   1 * time.Minute,
	}

	middleware := middlewares.NewIPThrottle(config)
	require.NotNil(t, middleware)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}


func TestThrottle_Stop(t *testing.T) {
	config := middlewares.ThrottleConfig{
		RequestsPerMinute: 10,
		BurstSize:         5,
		CleanupInterval:   10 * time.Millisecond,
	}

	throttle := middlewares.NewThrottle(config)

	// Use the throttle
	throttle.Allow("test-key")

	// Stop should not panic
	require.NotPanics(t, func() {
		throttle.Stop()
	})

	// Wait a bit to ensure cleanup goroutine has stopped
	time.Sleep(20 * time.Millisecond)
}

func TestThrottle_RateLimitHeaders(t *testing.T) {
	config := middlewares.ThrottleConfig{
		RequestsPerMinute: 10,
		BurstSize:         5,
		CleanupInterval:   1 * time.Hour,
	}

	middleware := middlewares.NewThrottleMiddleware(config, func(c echo.Context) string {
		return "test-key"
	})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	require.NoError(t, err)

	// Headers should be set
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Remaining"))
}
