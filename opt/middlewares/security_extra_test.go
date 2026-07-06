package middlewares_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"bank-api/opt/middlewares"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestScanRequestMiddleware_SkipPathsAndMethods(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		path        string
		shouldBlock bool
	}{
		{
			name:        "skips OPTIONS requests",
			method:      http.MethodOptions,
			path:        "/anything?q=UNION%20SELECT",
			shouldBlock: false,
		},
		{
			name:        "skips health check",
			method:      http.MethodGet,
			path:        "/health",
			shouldBlock: false,
		},
		{
			name:        "blocks malicious path",
			method:      http.MethodGet,
			path:        "/api/UNION%20SELECT/test",
			shouldBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			mw := middlewares.ScanRequestMiddleware()
			handler := mw(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler(c)

			if tt.shouldBlock {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestScanRequestMiddleware_CustomHeaders(t *testing.T) {
	tests := []struct {
		name        string
		headerKey   string
		headerVal   string
		shouldBlock bool
	}{
		{
			name:        "blocks malicious custom header",
			headerKey:   "X-Custom-Data",
			headerVal:   "<script>alert(1)</script>",
			shouldBlock: true,
		},
		{
			name:        "skips safe headers",
			headerKey:   "User-Agent",
			headerVal:   "Mozilla/5.0 UNION SELECT",
			shouldBlock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			mw := middlewares.ScanRequestMiddleware()
			handler := mw(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(tt.headerKey, tt.headerVal)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler(c)

			if tt.shouldBlock {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestScanRequestMiddleware_SkipBodyScan(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		body        string
		shouldBlock bool
	}{
		{
			name:        "skips auth login",
			path:        "/api/v1/auth/login",
			body:        `{"password": "' OR 1=1"}`,
			shouldBlock: false,
		},
		{
			name:        "skips auth register",
			path:        "/api/v1/auth/register",
			body:        `{"password": "' OR 1=1"}`,
			shouldBlock: false,
		},
		{
			name:        "skips auth reset",
			path:        "/api/v1/auth/reset-password",
			body:        `{"password": "' OR 1=1"}`,
			shouldBlock: false,
		},
		{
			name:        "skips instructors (base64 images)",
			path:        "/api/v1/instructors",
			body:        `{"image": "UNION SELECT"}`,
			shouldBlock: false,
		},
		{
			name:        "skips settings logo",
			path:        "/api/v1/settings/logo",
			body:        `{"image": "UNION SELECT"}`,
			shouldBlock: false,
		},
		{
			name:        "blocks other paths",
			path:        "/api/v1/users",
			body:        `{"name": "UNION SELECT"}`,
			shouldBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			mw := middlewares.ScanRequestMiddleware()
			handler := mw(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler(c)

			if tt.shouldBlock {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type errReader struct{}

func (errReader) Read(p []byte) (n int, err error) {
	return 0, bytes.ErrTooLarge
}

func TestScanRequestMiddleware_BodyReadError(t *testing.T) {
	e := echo.New()
	mw := middlewares.ScanRequestMiddleware()
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/test", new(errReader))
	req.ContentLength = 100 // fake content length
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	assert.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}
