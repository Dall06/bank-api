package middlewares_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bank-api/opt/middlewares"

	"github.com/labstack/echo/v4"
)

func TestAuditMiddleware(t *testing.T) {
	// Setup custom slog to capture output
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	slog.SetDefault(logger)

	tests := []struct {
		name          string
		method        string
		status        int
		contextValues map[string]interface{}
		wantLogMsg    string
		wantLogFields []string
		wantNoLog     bool
	}{
		{
			name:      "skip GET request",
			method:    http.MethodGet,
			status:    http.StatusOK,
			wantNoLog: true,
		},
		{
			name:      "skip HEAD request",
			method:    http.MethodHead,
			status:    http.StatusOK,
			wantNoLog: true,
		},
		{
			name:      "skip OPTIONS request",
			method:    http.MethodOptions,
			status:    http.StatusOK,
			wantNoLog: true,
		},
		{
			name:       "audit POST request - success",
			method:     http.MethodPost,
			status:     http.StatusOK,
			wantLogMsg: "audit",
			wantLogFields: []string{
				"level=INFO",
				"method=POST",
				"status=200",
			},
		},
		{
			name:       "audit PUT request - user error",
			method:     http.MethodPut,
			status:     http.StatusBadRequest,
			wantLogMsg: "audit",
			wantLogFields: []string{
				"level=WARN",
				"method=PUT",
				"status=400",
			},
		},
		{
			name:       "audit PATCH request - server error",
			method:     http.MethodPatch,
			status:     http.StatusInternalServerError,
			wantLogMsg: "audit",
			wantLogFields: []string{
				"level=ERROR",
				"method=PATCH",
				"status=500",
			},
		},
		{
			name:   "audit DELETE request with context fields",
			method: http.MethodDelete,
			status: http.StatusOK,
			contextValues: map[string]interface{}{
				"user_id":    "u-123",
				"staff_id":   "s-123",
				"company_id": "c-123",
				"role":       "admin",
			},
			wantLogMsg: "audit",
			wantLogFields: []string{
				"level=INFO",
				"method=DELETE",
				"status=200",
				"user_id=u-123",
				"staff_id=s-123",
				"company_id=c-123",
				"role=admin",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuf.Reset()

			e := echo.New()
			req := httptest.NewRequest(tt.method, "/test-path", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			for k, v := range tt.contextValues {
				c.Set(k, v)
			}

			handler := middlewares.AuditMiddleware()(func(c echo.Context) error {
				return c.NoContent(tt.status)
			})

			_ = handler(c)

			logOutput := logBuf.String()

			if tt.wantNoLog {
				if logOutput != "" {
					t.Errorf("expected no log output, got: %s", logOutput)
				}
				return
			}

			if !strings.Contains(logOutput, tt.wantLogMsg) {
				t.Errorf("expected log to contain message %q, got: %s", tt.wantLogMsg, logOutput)
			}

			for _, field := range tt.wantLogFields {
				if !strings.Contains(logOutput, field) {
					t.Errorf("expected log to contain field %q, got: %s", field, logOutput)
				}
			}
		})
	}
}
