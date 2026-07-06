package middlewares

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// ThrottleConfig holds rate limiting configuration
type ThrottleConfig struct {
	// RequestsPerMinute is the maximum number of requests allowed per minute per key
	RequestsPerMinute int
	// BurstSize is the maximum burst size (allows some flexibility)
	BurstSize int
	// CleanupInterval is how often to clean up expired entries
	CleanupInterval time.Duration
}

// DefaultThrottleConfig returns sensible defaults for authentication endpoints
func DefaultThrottleConfig() ThrottleConfig {
	return ThrottleConfig{
		RequestsPerMinute: 5,
		BurstSize:         10,
		CleanupInterval:   5 * time.Minute,
	}
}

// VerifyThrottleConfig returns config for verification kiosk terminals.
// Higher limits because a single kiosk IP may serve 100+ members per hour.
func VerifyThrottleConfig() ThrottleConfig {
	return ThrottleConfig{
		RequestsPerMinute: 120,
		BurstSize:         30,
		CleanupInterval:   5 * time.Minute,
	}
}

// throttleEntry tracks request count for a single key
type throttleEntry struct {
	count   int
	resetAt time.Time
	mu      sync.Mutex
}

// Throttle implements a sliding window rate limiter
type Throttle struct {
	config  ThrottleConfig
	entries sync.Map // map[string]*throttleEntry
	stopCh  chan struct{}
}

// NewThrottle creates a new rate limiter
func NewThrottle(config ThrottleConfig) *Throttle {
	t := &Throttle{
		config: config,
		stopCh: make(chan struct{}),
	}

	// Start cleanup goroutine
	go t.cleanup()

	return t
}

// Stop stops the cleanup goroutine
func (t *Throttle) Stop() {
	close(t.stopCh)
}

// cleanup periodically removes expired entries
func (t *Throttle) cleanup() {
	ticker := time.NewTicker(t.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			t.entries.Range(func(key, value interface{}) bool {
				entry := value.(*throttleEntry)
				entry.mu.Lock()
				if now.After(entry.resetAt) {
					t.entries.Delete(key)
				}
				entry.mu.Unlock()
				return true
			})
		case <-t.stopCh:
			return
		}
	}
}

// Allow checks if a request is allowed for the given key
func (t *Throttle) Allow(key string) bool {
	now := time.Now()
	window := time.Minute

	// Get or create entry
	val, _ := t.entries.LoadOrStore(key, &throttleEntry{
		count:   0,
		resetAt: now.Add(window),
	})
	entry := val.(*throttleEntry)

	entry.mu.Lock()
	defer entry.mu.Unlock()

	// Reset if window has passed
	if now.After(entry.resetAt) {
		entry.count = 0
		entry.resetAt = now.Add(window)
	}

	// Check if under limit
	if entry.count < t.config.RequestsPerMinute+t.config.BurstSize {
		entry.count++
		return true
	}

	return false
}

// Remaining returns the number of remaining requests for a key
func (t *Throttle) Remaining(key string) int {
	val, ok := t.entries.Load(key)
	if !ok {
		return t.config.RequestsPerMinute
	}

	entry := val.(*throttleEntry)
	entry.mu.Lock()
	defer entry.mu.Unlock()

	remaining := (t.config.RequestsPerMinute + t.config.BurstSize) - entry.count
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

// NewThrottleMiddleware returns an Echo middleware that rate limits requests
// keyFunc extracts the rate limiting key from the request (e.g., IP address)
func NewThrottleMiddleware(config ThrottleConfig, keyFunc func(c echo.Context) string) echo.MiddlewareFunc {
	throttle := NewThrottle(config)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			key := keyFunc(c)

			if !throttle.Allow(key) {
				c.Response().Header().Set("Retry-After", "60")
				return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			}

			// Add rate limit headers
			effectiveLimit := config.RequestsPerMinute + config.BurstSize
			c.Response().Header().Set("X-RateLimit-Limit", strconv.Itoa(effectiveLimit))
			c.Response().Header().Set("X-RateLimit-Remaining", strconv.Itoa(throttle.Remaining(key)))

			return next(c)
		}
	}
}

// IPKeyFunc extracts the client IP address as the rate limit key
func IPKeyFunc(c echo.Context) string {
	return c.RealIP()
}

// NewIPThrottle creates a throttle middleware that limits by IP address
func NewIPThrottle(config ThrottleConfig) echo.MiddlewareFunc {
	return NewThrottleMiddleware(config, IPKeyFunc)
}

