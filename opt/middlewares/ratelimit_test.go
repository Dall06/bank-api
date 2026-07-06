package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bank-api/opt/middlewares"

	"github.com/labstack/echo/v4"
)

func TestRateLimiter(t *testing.T) {
	// Simple test for IP limiting
	t.Run("IP Rate Limiting", func(t *testing.T) {
		rl := middlewares.NewRateLimiter(2, 5) // 2 requests per IP, 5 per email
		ip := "192.168.1.1"

		if rl.CheckIP(ip) {
			t.Errorf("expected false on first request, got true")
		}
		if rl.CheckIP(ip) {
			t.Errorf("expected false on second request, got true")
		}
		if !rl.CheckIP(ip) {
			t.Errorf("expected true on third request (over limit), got false")
		}
	})

	// Simple test for Email limiting
	t.Run("Email Rate Limiting", func(t *testing.T) {
		rl := middlewares.NewRateLimiter(5, 2)
		email := "test@example.com"

		if rl.CheckEmail(email) {
			t.Errorf("expected false on first request, got true")
		}
		if rl.CheckEmail(email) {
			t.Errorf("expected false on second request, got true")
		}
		if !rl.CheckEmail(email) {
			t.Errorf("expected true on third request (over limit), got false")
		}

		// Also test CheckEmailLimit error return
		err := rl.CheckEmailLimit(email)
		if err == nil {
			t.Errorf("expected error on fourth request, got nil")
		}
	})

	// Test default limits when <= 0
	t.Run("Default Limits", func(t *testing.T) {
		rl := middlewares.NewRateLimiter(0, -1) // defaults to 10 and 25
		ip := "10.0.0.1"
		for i := 0; i < 10; i++ {
			if rl.CheckIP(ip) {
				t.Errorf("expected false on request %d, got true", i)
			}
		}
		if !rl.CheckIP(ip) {
			t.Errorf("expected true on 11th request (over limit), got false")
		}
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	rl := middlewares.NewRateLimiter(1, 5) // 1 request per IP

	tests := []struct {
		name       string
		method     string
		path       string
		reqPath    string
		ip         string
		wantStatus int
	}{
		{
			name:       "not POST method bypasses limit",
			method:     http.MethodGet,
			path:       "/login",
			reqPath:    "/login",
			ip:         "127.0.0.1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "different path bypasses limit",
			method:     http.MethodPost,
			path:       "/login",
			reqPath:    "/register",
			ip:         "127.0.0.1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "first POST to path allowed",
			method:     http.MethodPost,
			path:       "/login",
			reqPath:    "/login",
			ip:         "192.168.0.2",
			wantStatus: http.StatusOK,
		},
		{
			name:       "second POST to same path blocked",
			method:     http.MethodPost,
			path:       "/login",
			reqPath:    "/login",
			ip:         "192.168.0.2",
			wantStatus: http.StatusTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(tt.method, tt.reqPath, nil)
			req.Header.Set(echo.HeaderXRealIP, tt.ip)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath(tt.reqPath)

			handler := middlewares.RateLimitMiddleware(rl, tt.path)(func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			_ = handler(c)

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}
