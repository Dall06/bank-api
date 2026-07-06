package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestSetup(t *testing.T) {
	e := echo.New()
	cfg := Config{
		JWTSecret:      "secret",
		UserURL:        "http://localhost:8081",
		TransactionsURL: "http://localhost:8082",
		AllowedOrigins: "http://localhost:3000",
		SigilSecret:    "sigil",
	}

	Setup(e, cfg)

	tests := []struct {
		name         string
		method       string
		path         string
		expectedCode int
	}{
		{
			name:         "Health check /health",
			method:       http.MethodGet,
			path:         "/health",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Health check /api/health",
			method:       http.MethodGet,
			path:         "/api/health",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Swagger UI",
			method:       http.MethodGet,
			path:         "/swagger/index.html",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.expectedCode && tt.path == "/swagger/index.html" && rec.Code == http.StatusMovedPermanently {
			    // Swagger UI redirects are fine
			} else if rec.Code != tt.expectedCode {
				t.Errorf("got code %d, want %d", rec.Code, tt.expectedCode)
			}
		})
	}
}
