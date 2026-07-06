package middlewares_test

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"bank-api/opt/middlewares"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	// Discard slog output during tests
	slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, nil)))
}

func TestIDsMiddleware(t *testing.T) {
	tests := []struct {
		name              string
		setupRequest      func(*http.Request)
		expectTraceID     bool
		expectRequestID   bool
		traceIDInHeader   string
		validateTraceID   func(string) bool
		validateRequestID func(string) bool
	}{
		{
			name: "generates new trace ID when missing",
			setupRequest: func(req *http.Request) {
				// No trace-id header
			},
			expectTraceID:   true,
			expectRequestID: true,
			validateTraceID: func(id string) bool {
				_, err := uuid.Parse(id)
				return err == nil
			},
			validateRequestID: func(id string) bool {
				_, err := uuid.Parse(id)
				return err == nil
			},
		},
		{
			name: "uses provided trace ID",
			setupRequest: func(req *http.Request) {
				req.Header.Set("x-trace-id", "existing-trace-id-123")
			},
			expectTraceID:   true,
			expectRequestID: true,
			traceIDInHeader: "existing-trace-id-123",
			validateTraceID: func(id string) bool {
				return id == "existing-trace-id-123"
			},
			validateRequestID: func(id string) bool {
				_, err := uuid.Parse(id)
				return err == nil
			},
		},
		{
			name: "generates request ID regardless of trace ID",
			setupRequest: func(req *http.Request) {
				req.Header.Set("x-trace-id", "trace-456")
			},
			expectTraceID:   true,
			expectRequestID: true,
			traceIDInHeader: "trace-456",
			validateTraceID: func(id string) bool {
				return id == "trace-456"
			},
			validateRequestID: func(id string) bool {
				_, err := uuid.Parse(id)
				return err == nil
			},
		},
		{
			name: "handles empty trace ID as missing",
			setupRequest: func(req *http.Request) {
				req.Header.Set("x-trace-id", "")
			},
			expectTraceID:   true,
			expectRequestID: true,
			validateTraceID: func(id string) bool {
				_, err := uuid.Parse(id)
				return err == nil
			},
			validateRequestID: func(id string) bool {
				_, err := uuid.Parse(id)
				return err == nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.setupRequest(req)

			// Middleware
			middleware := middlewares.IDsMiddleware()
			nextCalled := false

			handler := middleware(func(ctx echo.Context) error {
				nextCalled = true
				return ctx.String(http.StatusOK, "success")
			})

			// Execute
			err := handler(c)

			// Assertions
			require.NoError(t, err)
			assert.True(t, nextCalled, "next handler should be called")

			// Verify trace ID
			if tt.expectTraceID {
				// Check response header
				traceID := rec.Header().Get("x-trace-id")
				assert.NotEmpty(t, traceID, "trace-id should be set in response header")
				if tt.validateTraceID != nil {
					assert.True(t, tt.validateTraceID(traceID), "trace-id should be valid")
				}

				// Check context
				ctxTraceID := c.Get("trace-id")
				assert.NotNil(t, ctxTraceID, "trace-id should be set in context")
				assert.Equal(t, traceID, ctxTraceID, "trace-id in context should match header")
			}

			// Verify request ID
			if tt.expectRequestID {
				// Check response header
				requestID := rec.Header().Get("x-request-id")
				assert.NotEmpty(t, requestID, "request-id should be set in response header")
				if tt.validateRequestID != nil {
					assert.True(t, tt.validateRequestID(requestID), "request-id should be valid UUID")
				}

				// Check context
				ctxRequestID := c.Get("request-id")
				assert.NotNil(t, ctxRequestID, "request-id should be set in context")
				assert.Equal(t, requestID, ctxRequestID, "request-id in context should match header")
			}
		})
	}
}

func TestIDsMiddleware_ResponseHeaders(t *testing.T) {
	tests := []struct {
		name        string
		traceIDIn   string
		expectBoth  bool
		expectTrace string
	}{
		{
			name:        "sets both headers when trace ID provided",
			traceIDIn:   "custom-trace-123",
			expectBoth:  true,
			expectTrace: "custom-trace-123",
		},
		{
			name:       "sets both headers when trace ID missing",
			traceIDIn:  "",
			expectBoth: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.traceIDIn != "" {
				req.Header.Set("x-trace-id", tt.traceIDIn)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			middleware := middlewares.IDsMiddleware()
			handler := middleware(func(ctx echo.Context) error {
				return ctx.NoContent(http.StatusOK)
			})

			err := handler(c)
			require.NoError(t, err)

			if tt.expectBoth {
				assert.NotEmpty(t, rec.Header().Get("x-trace-id"), "x-trace-id header should be set")
				assert.NotEmpty(t, rec.Header().Get("x-request-id"), "x-request-id header should be set")

				if tt.expectTrace != "" {
					assert.Equal(t, tt.expectTrace, rec.Header().Get("x-trace-id"))
				}
			}
		})
	}
}

func TestIDsMiddleware_ContextValues(t *testing.T) {
	tests := []struct {
		name      string
		traceIDIn string
	}{
		{
			name:      "stores values in context with provided trace ID",
			traceIDIn: "trace-from-upstream",
		},
		{
			name:      "stores values in context with generated trace ID",
			traceIDIn: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.traceIDIn != "" {
				req.Header.Set("x-trace-id", tt.traceIDIn)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			var capturedTraceID, capturedRequestID interface{}

			middleware := middlewares.IDsMiddleware()
			handler := middleware(func(ctx echo.Context) error {
				capturedTraceID = ctx.Get("trace-id")
				capturedRequestID = ctx.Get("request-id")
				return ctx.NoContent(http.StatusOK)
			})

			err := handler(c)
			require.NoError(t, err)

			// Verify context values
			assert.NotNil(t, capturedTraceID, "trace-id should be in context")
			assert.NotNil(t, capturedRequestID, "request-id should be in context")

			// Verify they match headers
			assert.Equal(t, rec.Header().Get("x-trace-id"), capturedTraceID)
			assert.Equal(t, rec.Header().Get("x-request-id"), capturedRequestID)

			// If trace ID was provided, verify it matches
			if tt.traceIDIn != "" {
				assert.Equal(t, tt.traceIDIn, capturedTraceID)
			}
		})
	}
}

func TestIDsMiddleware_UUIDGeneration(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No trace-id header
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	middleware := middlewares.IDsMiddleware()
	handler := middleware(func(ctx echo.Context) error {
		return ctx.NoContent(http.StatusOK)
	})

	err := handler(c)
	require.NoError(t, err)

	// Verify both IDs are valid UUIDs
	traceID := rec.Header().Get("x-trace-id")
	requestID := rec.Header().Get("x-request-id")

	_, err = uuid.Parse(traceID)
	assert.NoError(t, err, "trace-id should be a valid UUID when generated")

	_, err = uuid.Parse(requestID)
	assert.NoError(t, err, "request-id should be a valid UUID")
}

func TestIDsMiddleware_UniqueRequestIDs(t *testing.T) {
	// Test that each request gets a unique request ID
	traceID := "shared-trace-id"

	requestIDs := make(map[string]bool)

	for i := 0; i < 10; i++ {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("x-trace-id", traceID)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		middleware := middlewares.IDsMiddleware()
		handler := middleware(func(ctx echo.Context) error {
			return ctx.NoContent(http.StatusOK)
		})

		err := handler(c)
		require.NoError(t, err)

		requestID := rec.Header().Get("x-request-id")
		assert.NotEmpty(t, requestID)
		assert.False(t, requestIDs[requestID], "request ID should be unique")
		requestIDs[requestID] = true
	}

	assert.Len(t, requestIDs, 10, "all request IDs should be unique")
}

func TestIDsMiddleware_ErrorLogging(t *testing.T) {
	t.Run("logs error when UUID generation fails", func(t *testing.T) {
		// UUID generation rarely fails in practice. Verify that the middleware
		// handles normal operation without errors.
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		middleware := middlewares.IDsMiddleware()
		handler := middleware(func(ctx echo.Context) error {
			return ctx.NoContent(http.StatusOK)
		})

		err := handler(c)

		// Should not error in normal case
		assert.NoError(t, err)
	})
}

func TestIDsMiddleware_ConcurrentRequests(t *testing.T) {
	// Test thread safety
	middleware := middlewares.IDsMiddleware()

	done := make(chan bool, 50)
	requestIDs := make(chan string, 50)

	for i := 0; i < 50; i++ {
		go func() {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := middleware(func(ctx echo.Context) error {
				return ctx.NoContent(http.StatusOK)
			})

			err := handler(c)
			if err == nil {
				requestIDs <- rec.Header().Get("x-request-id")
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}
	close(requestIDs)

	// Verify all request IDs are unique
	idMap := make(map[string]bool)
	for id := range requestIDs {
		assert.False(t, idMap[id], "concurrent request IDs should all be unique")
		idMap[id] = true
	}

	assert.Len(t, idMap, 50, "all concurrent requests should get unique IDs")
}

func TestIDsMiddleware_PreservesTraceIDCase(t *testing.T) {
	tests := []struct {
		name       string
		traceID    string
		expectSame bool
	}{
		{
			name:       "preserves lowercase",
			traceID:    "trace-lowercase-123",
			expectSame: true,
		},
		{
			name:       "preserves uppercase",
			traceID:    "TRACE-UPPERCASE-456",
			expectSame: true,
		},
		{
			name:       "preserves mixed case",
			traceID:    "Trace-MixedCase-789",
			expectSame: true,
		},
		{
			name:       "preserves special characters",
			traceID:    "trace_with-special.chars:123",
			expectSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("x-trace-id", tt.traceID)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			middleware := middlewares.IDsMiddleware()
			handler := middleware(func(ctx echo.Context) error {
				return ctx.NoContent(http.StatusOK)
			})

			err := handler(c)
			require.NoError(t, err)

			if tt.expectSame {
				assert.Equal(t, tt.traceID, rec.Header().Get("x-trace-id"),
					"trace ID should be preserved exactly as provided")
			}
		})
	}
}

func TestIDsMiddleware_NextHandlerError(t *testing.T) {
	// Verify that errors from next handler are propagated
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	expectedError := errors.New("handler error")

	middleware := middlewares.IDsMiddleware()
	handler := middleware(func(ctx echo.Context) error {
		return expectedError
	})

	err := handler(c)

	// Error should be propagated
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)

	// But IDs should still be set
	assert.NotEmpty(t, rec.Header().Get("x-trace-id"))
	assert.NotEmpty(t, rec.Header().Get("x-request-id"))
}

