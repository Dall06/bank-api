package middlewares

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// RateLimiter provides in-memory rate limiting by IP and email.
type RateLimiter struct {
	mu       sync.RWMutex
	requests map[string][]time.Time

	// Configuration
	ipLimit       int           // Max requests per IP within ipWindow
	ipWindow      time.Duration // Time window for IP limit
	emailLimit    int           // Max requests per email within emailWindow
	emailWindow   time.Duration // Time window for email limit
	cleanupTicker *time.Ticker
}

// NewRateLimiter creates a new rate limiter.
//
//	ipLimit:    max requests per IP within a 1-minute window
//	emailLimit: max requests per email within a 1-hour window
//
// Zero or negative values fall back to sane defaults (10 IP/min, 25 email/hour)
// so callers that forget to configure it still get working protection.
// The windows are fixed on purpose — tuning only the counts covers ~every
// real ops need. If you ever need variable windows, promote them to params.
func NewRateLimiter(ipLimit, emailLimit int) *RateLimiter {
	if ipLimit <= 0 {
		ipLimit = 10
	}
	if emailLimit <= 0 {
		emailLimit = 25
	}
	rl := &RateLimiter{
		requests:    make(map[string][]time.Time),
		ipLimit:     ipLimit,
		ipWindow:    time.Minute,
		emailLimit:  emailLimit,
		emailWindow: time.Hour,
	}

	// Cleanup old entries every 5 minutes
	rl.cleanupTicker = time.NewTicker(5 * time.Minute)
	go rl.cleanup()

	return rl
}

// cleanup removes old entries periodically
func (rl *RateLimiter) cleanup() {
	for range rl.cleanupTicker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, times := range rl.requests {
			// Keep only entries from the last hour
			var valid []time.Time
			for _, t := range times {
				if now.Sub(t) < time.Hour {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, key)
			}
			if len(valid) > 0 {
				rl.requests[key] = valid
			}
		}
		rl.mu.Unlock()
	}
}

// isLimited checks if the key has exceeded the rate limit
func (rl *RateLimiter) isLimited(key string, limit int, window time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-window)

	// Get existing requests and filter to window
	var validTimes []time.Time
	for _, t := range rl.requests[key] {
		if t.After(windowStart) {
			validTimes = append(validTimes, t)
		}
	}

	// Check if over limit
	if len(validTimes) >= limit {
		return true
	}

	// Add current request
	validTimes = append(validTimes, now)
	rl.requests[key] = validTimes

	return false
}

// CheckIP checks if an IP has exceeded the rate limit
func (rl *RateLimiter) CheckIP(ip string) bool {
	return rl.isLimited("ip:"+ip, rl.ipLimit, rl.ipWindow)
}

// CheckEmail checks if an email has exceeded the rate limit
func (rl *RateLimiter) CheckEmail(email string) bool {
	return rl.isLimited("email:"+email, rl.emailLimit, rl.emailWindow)
}

// RateLimitMiddleware creates Echo middleware for IP-based rate limiting.
// The path parameter restricts the check to a specific route.
func RateLimitMiddleware(rl *RateLimiter, path string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Method != http.MethodPost || c.Path() != path {
				return next(c)
			}

			ip := c.RealIP()
			if rl.CheckIP(ip) {
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"error":   "rate_limit_exceeded",
					"message": "Too many requests. Please wait a minute before trying again.",
				})
			}

			return next(c)
		}
	}
}

// CheckEmailLimit checks if an email has exceeded the rate limit and returns an error if so.
func (rl *RateLimiter) CheckEmailLimit(email string) error {
	if rl.CheckEmail(email) {
		return echo.NewHTTPError(http.StatusTooManyRequests,
			"You have submitted too many requests. Please wait an hour before trying again.")
	}
	return nil
}
