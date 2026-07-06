package middlewares_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bank-api/opt/middlewares"

	"github.com/labstack/echo/v4"
)

func TestGatewayRequestLogger(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	slog.SetDefault(logger)

	tests := []struct {
		name          string
		method        string
		status        int
		handlerErr    error
		contextValues map[string]interface{}
		wantLogMsg    string
		wantLogFields []string
	}{
		{
			name:       "successful request logs info",
			method:     http.MethodGet,
			status:     http.StatusOK,
			wantLogMsg: "request completed",
			wantLogFields: []string{
				"level=INFO",
				"method=GET",
				"status=200",
			},
		},
		{
			name:       "client error logs warn",
			method:     http.MethodPost,
			status:     http.StatusBadRequest,
			wantLogMsg: "client error",
			wantLogFields: []string{
				"level=WARN",
				"method=POST",
				"status=400",
			},
		},
		{
			name:       "server error logs error",
			method:     http.MethodPut,
			status:     http.StatusInternalServerError,
			wantLogMsg: "server error",
			wantLogFields: []string{
				"level=ERROR",
				"method=PUT",
				"status=500",
			},
		},
		{
			name:       "handler error logs as failed request",
			method:     http.MethodDelete,
			status:     http.StatusOK, // Status doesn't matter if error is returned
			handlerErr: errors.New("something went wrong"),
			wantLogMsg: "request failed",
			wantLogFields: []string{
				"level=ERROR",
				"method=DELETE",
				"error=\"something went wrong\"",
			},
		},
		{
			name:   "includes tenant and staff id if present",
			method: http.MethodGet,
			status: http.StatusOK,
			contextValues: map[string]interface{}{
				"tenant_slug": "tenant-1",
				"staff_id":    "staff-1",
			},
			wantLogMsg: "request completed",
			wantLogFields: []string{
				"level=INFO",
				"tenant=tenant-1",
				"staff_id=staff-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuf.Reset()

			e := echo.New()
			req := httptest.NewRequest(tt.method, "/test", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			for k, v := range tt.contextValues {
				c.Set(k, v)
			}

			handler := middlewares.GatewayRequestLogger()(func(c echo.Context) error {
				if tt.handlerErr != nil {
					return tt.handlerErr
				}
				return c.NoContent(tt.status)
			})

			_ = handler(c)

			logOutput := logBuf.String()

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
