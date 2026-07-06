package middlewares_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bank-api/opt/middlewares"
	"bank-api/pkg/sigil"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSigilVerifier(t *testing.T) {
	secret := "test-secret-key"
	config := sigil.DefaultConfig(secret, "gateway")
	signer := sigil.NewSigner(config)
	verifier := sigil.NewVerifier(config, []string{"gateway"})

	tests := []struct {
		name           string
		setupRequest   func(req *http.Request, body []byte)
		body           string
		expectedStatus int
		expectError    bool
		errorContains  string
	}{
		{
			name: "success - valid signature",
			setupRequest: func(req *http.Request, body []byte) {
				headers := signer.SignRequest(body)
				for k, v := range headers {
					req.Header.Set(k, v)
				}
			},
			body:           `{"test":"data"}`,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "error - missing headers",
			setupRequest: func(req *http.Request, body []byte) {
				// No headers set
			},
			body:           `{"test":"data"}`,
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
			errorContains:  "missing authentication headers",
		},
		{
			name: "error - missing service ID",
			setupRequest: func(req *http.Request, body []byte) {
				headers := signer.SignRequest(body)
				req.Header.Set(sigil.HeaderTimestamp, headers[sigil.HeaderTimestamp])
				req.Header.Set(sigil.HeaderSignature, headers[sigil.HeaderSignature])
				// Missing X-Service-ID
			},
			body:           `{"test":"data"}`,
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
			errorContains:  "missing authentication headers",
		},
		{
			name: "error - unknown service",
			setupRequest: func(req *http.Request, body []byte) {
				// Sign with an unknown service ID
				unknownConfig := sigil.DefaultConfig(secret, "unknown-service")
				unknownSigner := sigil.NewSigner(unknownConfig)
				headers := unknownSigner.SignRequest(body)
				for k, v := range headers {
					req.Header.Set(k, v)
				}
			},
			body:           `{"test":"data"}`,
			expectedStatus: http.StatusForbidden,
			expectError:    true,
			errorContains:  "unknown service",
		},
		{
			name: "error - invalid signature",
			setupRequest: func(req *http.Request, body []byte) {
				headers := signer.SignRequest(body)
				for k, v := range headers {
					req.Header.Set(k, v)
				}
				// Corrupt the signature
				req.Header.Set(sigil.HeaderSignature, "invalid-signature")
			},
			body:           `{"test":"data"}`,
			expectedStatus: http.StatusUnauthorized,
			expectError:    true,
			errorContains:  "invalid signature",
		},
		{
			name: "success - empty body",
			setupRequest: func(req *http.Request, body []byte) {
				headers := signer.SignRequest(body)
				for k, v := range headers {
					req.Header.Set(k, v)
				}
			},
			body:           "",
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "success - large body",
			setupRequest: func(req *http.Request, body []byte) {
				headers := signer.SignRequest(body)
				for k, v := range headers {
					req.Header.Set(k, v)
				}
			},
			body:           string(make([]byte, 10000)),
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			body := []byte(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.setupRequest(req, body)

			middleware := middlewares.NewSigilVerifier(verifier)
			handler := middleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "OK")
			})

			err := handler(c)

			if tt.expectError {
				require.Error(t, err)
				httpErr, ok := err.(*echo.HTTPError)
				require.True(t, ok)
				assert.Equal(t, tt.expectedStatus, httpErr.Code)
				if tt.errorContains != "" {
					assert.Contains(t, httpErr.Message, tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestNewSigilVerifier_ExpiredTimestamp(t *testing.T) {
	secret := "test-secret-key"

	// Create config with very short tolerance
	config := sigil.Config{
		Secret:             secret,
		ServiceID:          "gateway",
		TimestampTolerance: 1 * time.Millisecond,
	}

	signer := sigil.NewSigner(config)
	verifier := sigil.NewVerifier(config, []string{"gateway"})

	e := echo.New()
	body := []byte(`{"test":"data"}`)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Sign the request
	headers := signer.SignRequest(body)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Wait for timestamp to expire
	time.Sleep(10 * time.Millisecond)

	middleware := middlewares.NewSigilVerifier(verifier)
	handler := middleware(func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	require.Error(t, err)
	httpErr, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
	assert.Contains(t, httpErr.Message, "request expired")
}

func TestNewSigilVerifier_BodyPreserved(t *testing.T) {
	secret := "test-secret-key"
	config := sigil.DefaultConfig(secret, "gateway")
	signer := sigil.NewSigner(config)
	verifier := sigil.NewVerifier(config, []string{"gateway"})

	e := echo.New()
	originalBody := `{"test":"data","nested":{"key":"value"}}`
	body := []byte(originalBody)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	headers := signer.SignRequest(body)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	var receivedBody []byte
	middleware := middlewares.NewSigilVerifier(verifier)
	handler := middleware(func(c echo.Context) error {
		// Read the body in the handler to verify it's preserved
		buf := new(bytes.Buffer)
		buf.ReadFrom(c.Request().Body)
		receivedBody = buf.Bytes()
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, originalBody, string(receivedBody))
}

func TestNewSigilVerifier_ServiceIDInContext(t *testing.T) {
	secret := "test-secret-key"
	config := sigil.DefaultConfig(secret, "gateway")
	signer := sigil.NewSigner(config)
	verifier := sigil.NewVerifier(config, []string{"gateway"})

	e := echo.New()
	body := []byte(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	headers := signer.SignRequest(body)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	var contextServiceID string
	middleware := middlewares.NewSigilVerifier(verifier)
	handler := middleware(func(c echo.Context) error {
		contextServiceID = c.Get("service_id").(string)
		return c.String(http.StatusOK, "OK")
	})

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, "gateway", contextServiceID)
}

func TestNewSigilHeaders(t *testing.T) {
	secret := "test-secret-key"
	config := sigil.DefaultConfig(secret, "gateway")
	signer := sigil.NewSigner(config)

	tests := []struct {
		name string
		body []byte
	}{
		{
			name: "adds headers to request with body",
			body: []byte(`{"test":"data"}`),
		},
		{
			name: "adds headers to request with empty body",
			body: []byte{},
		},
		{
			name: "adds headers to request with nil body",
			body: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sigilHeaders := middlewares.NewSigilHeaders(signer)
			req := httptest.NewRequest(http.MethodPost, "/test", nil)

			sigilHeaders.AddHeaders(req, tt.body)

			assert.NotEmpty(t, req.Header.Get(sigil.HeaderServiceID))
			assert.NotEmpty(t, req.Header.Get(sigil.HeaderTimestamp))
			assert.NotEmpty(t, req.Header.Get(sigil.HeaderSignature))
			assert.Equal(t, "gateway", req.Header.Get(sigil.HeaderServiceID))
		})
	}
}

func TestSigilVerifier_MultipleServices(t *testing.T) {
	secret := "shared-secret"

	// Create verifier that accepts multiple services
	config := sigil.DefaultConfig(secret, "verifier")
	verifier := sigil.NewVerifier(config, []string{"gateway", "companies", "gym"})

	services := []string{"gateway", "companies", "gym"}

	for _, serviceID := range services {
		t.Run("accepts "+serviceID, func(t *testing.T) {
			signerConfig := sigil.DefaultConfig(secret, serviceID)
			signer := sigil.NewSigner(signerConfig)

			e := echo.New()
			body := []byte(`{}`)
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			headers := signer.SignRequest(body)
			for k, v := range headers {
				req.Header.Set(k, v)
			}

			middleware := middlewares.NewSigilVerifier(verifier)
			handler := middleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "OK")
			})

			err := handler(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestSigilVerifier_DifferentHTTPMethods(t *testing.T) {
	secret := "test-secret-key"
	config := sigil.DefaultConfig(secret, "gateway")
	signer := sigil.NewSigner(config)
	verifier := sigil.NewVerifier(config, []string{"gateway"})

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
	}

	for _, method := range methods {
		t.Run(method+" request", func(t *testing.T) {
			e := echo.New()
			body := []byte(`{}`)
			req := httptest.NewRequest(method, "/test", bytes.NewReader(body))
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			headers := signer.SignRequest(body)
			for k, v := range headers {
				req.Header.Set(k, v)
			}

			middleware := middlewares.NewSigilVerifier(verifier)
			handler := middleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "OK")
			})

			err := handler(c)
			require.NoError(t, err)
		})
	}
}
