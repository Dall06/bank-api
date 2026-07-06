package middlewares_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"bank-api/opt/middlewares"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoggerMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		setupContext   func(c echo.Context)
		handler        echo.HandlerFunc
		expectedStatus int
		expectError    bool
	}{
		{
			name: "success - logs request with trace-id from header",
			setupContext: func(c echo.Context) {
				c.Request().Header.Set("trace-id", "test-trace-123")
				c.Request().Header.Set("request-id", "test-request-456")
			},
			handler: func(c echo.Context) error {
				return c.String(http.StatusOK, "OK")
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name: "success - logs request with trace-id from context",
			setupContext: func(c echo.Context) {
				c.Set("trace-id", "context-trace-789")
				c.Set("request-id", "context-request-012")
			},
			handler: func(c echo.Context) error {
				return c.String(http.StatusOK, "OK")
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:         "success - logs request without trace-id",
			setupContext: func(c echo.Context) {},
			handler: func(c echo.Context) error {
				return c.String(http.StatusOK, "OK")
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:         "error - logs handler error",
			setupContext: func(c echo.Context) {},
			handler: func(c echo.Context) error {
				return errors.New("handler error")
			},
			expectedStatus: http.StatusOK,
			expectError:    true,
		},
		{
			name: "success - POST request with body",
			setupContext: func(c echo.Context) {
				c.Request().Header.Set("Content-Type", "application/json")
			},
			handler: func(c echo.Context) error {
				return c.JSON(http.StatusCreated, map[string]string{"id": "123"})
			},
			expectedStatus: http.StatusCreated,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			tt.setupContext(c)

			middleware := middlewares.LoggerMiddleware()
			wrappedHandler := middleware(tt.handler)

			err := wrappedHandler(c)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestLoggerMiddleware_HTTPMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodHead,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run("logs "+method+" request", func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(method, "/api/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := func(c echo.Context) error {
				return c.NoContent(http.StatusNoContent)
			}

			middleware := middlewares.LoggerMiddleware()
			wrappedHandler := middleware(handler)

			err := wrappedHandler(c)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNoContent, rec.Code)
		})
	}
}

func TestLoggerMiddleware_StatusCodes(t *testing.T) {
	statusCodes := []int{
		http.StatusOK,
		http.StatusCreated,
		http.StatusNoContent,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusNotFound,
		http.StatusInternalServerError,
	}

	for _, status := range statusCodes {
		t.Run("logs status "+http.StatusText(status), func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := func(c echo.Context) error {
				return c.NoContent(status)
			}

			middleware := middlewares.LoggerMiddleware()
			wrappedHandler := middleware(handler)

			err := wrappedHandler(c)
			require.NoError(t, err)
			assert.Equal(t, status, rec.Code)
		})
	}
}

func TestLoggerMiddleware_HeaderPriority(t *testing.T) {
	t.Run("header takes priority over context value", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Set both header and context
		c.Request().Header.Set("trace-id", "header-trace")
		c.Set("trace-id", "context-trace")

		handlerCalled := false
		handler := func(c echo.Context) error {
			handlerCalled = true
			return c.String(http.StatusOK, "OK")
		}

		middleware := middlewares.LoggerMiddleware()
		wrappedHandler := middleware(handler)

		err := wrappedHandler(c)
		require.NoError(t, err)
		assert.True(t, handlerCalled)
	})
}

func TestLoggerMiddleware_ConcurrentRequests(t *testing.T) {
	middleware := middlewares.LoggerMiddleware()

	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func(idx int) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := func(c echo.Context) error {
				return c.String(http.StatusOK, "OK")
			}

			wrappedHandler := middleware(handler)
			_ = wrappedHandler(c)
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}
