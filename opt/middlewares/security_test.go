package middlewares

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestScanRequestMiddleware_SQLInjection(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldBlock bool
	}{
		// Should be blocked
		{"union select", "1 UNION SELECT * FROM users", true},
		{"drop table", "'; DROP TABLE users;--", true},
		{"or 1=1", "' OR 1=1", true},
		{"comment", "admin'--", true},
		{"sleep", "1; SLEEP(5)", true},
		{"information schema", "SELECT * FROM INFORMATION_SCHEMA", true},

		// Should pass
		{"normal text", "hello world", false},
		{"email", "user@example.com", false},
		{"uuid", "550e8400-e29b-41d4-a716-446655440000", false},
		{"number", "12345", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			mw := ScanRequestMiddleware()
			handler := mw(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/?q="+url.QueryEscape(tt.input), nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			err := handler(c)

			if tt.shouldBlock {
				assert.Error(t, err)
				he, ok := err.(*echo.HTTPError)
				assert.True(t, ok)
				assert.Equal(t, http.StatusForbidden, he.Code)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestScanRequestMiddleware_XSS(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldBlock bool
	}{
		{"script tag", "<script>alert(1)</script>", true},
		{"onclick", `<div onclick="alert(1)">`, true},
		{"javascript", `javascript:alert(1)`, true},
		{"iframe", `<iframe src="x">`, true},
		{"normal text", "Hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			mw := ScanRequestMiddleware()
			handler := mw(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/?q="+url.QueryEscape(tt.input), nil)
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

func TestScanRequestMiddleware_PathTraversal(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldBlock bool
	}{
		{"dot dot slash", "../../../etc/passwd", true},
		{"etc passwd", "/etc/passwd", true},
		{"encoded", "%2e%2e/", true},
		{"normal path", "/api/v1/users", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			mw := ScanRequestMiddleware()
			handler := mw(func(c echo.Context) error {
				return c.String(http.StatusOK, "ok")
			})

			req := httptest.NewRequest(http.MethodGet, "/?path="+url.QueryEscape(tt.input), nil)
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

func TestScanRequestMiddleware_Body(t *testing.T) {
	e := echo.New()
	mw := ScanRequestMiddleware()
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// Malicious body
	body := `{"name": "test' UNION SELECT * FROM users--"}`
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	assert.Error(t, err)

	// Safe body
	safeBody := `{"name": "John Doe", "email": "john@example.com"}`
	req2 := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(safeBody))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)

	err2 := handler(c2)
	assert.NoError(t, err2)
}

func TestScanRequestMiddleware_SkipsAuthHeader(t *testing.T) {
	e := echo.New()
	mw := ScanRequestMiddleware()
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	// JWT in Authorization header should not be scanned
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiJ9...")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	assert.NoError(t, err)
}

func TestScanForThreats(t *testing.T) {
	assert.Equal(t, "sql_injection", scanForThreats("' OR 1=1"))
	assert.Equal(t, "xss", scanForThreats("<script>"))
	assert.Equal(t, "path_traversal", scanForThreats("../../../"))
	assert.Equal(t, "", scanForThreats("normal text"))
}
