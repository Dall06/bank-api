package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"bank-api/pkg/sigil"

	"github.com/labstack/echo/v4"
)

func TestProxyToUser(t *testing.T) {
	e := echo.New()

	// Mock server that acts as the Users microservice
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/login" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer target.Close()

	signer := sigil.NewSigner(sigil.DefaultConfig("secret", "gateway"))
	h := NewProxyHandler(target.URL, "http://localhost:8082", signer)

	tests := []struct {
		name         string
		method       string
		path         string
		body         string
		expectedCode int
	}{
		{
			name:         "Success proxy login",
			method:       http.MethodPost,
			path:         "/auth/login",
			body:         `{"username":"test"}`,
			expectedCode: http.StatusOK,
		},
		{
			name:         "Not found path",
			method:       http.MethodPost,
			path:         "/invalid",
			body:         "",
			expectedCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewReader([]byte(tt.body)))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)

			err := h.ProxyToUser(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rec.Code != tt.expectedCode {
				t.Errorf("got code %d, want %d", rec.Code, tt.expectedCode)
			}
		})
	}
}

func TestProxyToTransactions(t *testing.T) {
	e := echo.New()

	// Mock server that acts as the Transactions microservice
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/transactions" && r.Method == http.MethodGet {
			if r.URL.RawQuery == "limit=10" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"query_ok"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer target.Close()

	signer := sigil.NewSigner(sigil.DefaultConfig("secret", "gateway"))
	h := NewProxyHandler("http://localhost:8081", target.URL, signer)

	tests := []struct {
		name         string
		method       string
		path         string
		query        string
		expectedCode int
	}{
		{
			name:         "Success proxy transactions GET",
			method:       http.MethodGet,
			path:         "/transactions",
			expectedCode: http.StatusOK,
		},
		{
			name:         "Success proxy transactions GET with Query",
			method:       http.MethodGet,
			path:         "/transactions",
			query:        "limit=10",
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetURL := tt.path
			if tt.query != "" {
				targetURL += "?" + tt.query
			}
			req := httptest.NewRequest(tt.method, targetURL, nil)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)

			err := h.ProxyToTransactions(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rec.Code != tt.expectedCode {
				t.Errorf("got code %d, want %d", rec.Code, tt.expectedCode)
			}
		})
	}
}

func TestProxyError(t *testing.T) {
	e := echo.New()
	signer := sigil.NewSigner(sigil.DefaultConfig("secret", "gateway"))
	// Use an invalid URL to force transport error
	h := NewProxyHandler("http://invalid.local:12345", "http://invalid.local:12345", signer)

	req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	err := h.ProxyToUser(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
