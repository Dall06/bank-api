package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bank-api/env/consts"
	"bank-api/opt/middlewares"

	"github.com/labstack/echo/v4"
)

func TestMockHeaderMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		mockID     string
		wantMockID string
		wantCtx    bool
	}{
		{
			name:       "with mock id header",
			mockID:     "mock-123",
			wantMockID: "mock-123",
			wantCtx:    true,
		},
		{
			name:       "without mock id header",
			mockID:     "",
			wantMockID: "",
			wantCtx:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.mockID != "" {
				req.Header.Set(consts.MockIDHeaderKey, tt.mockID)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := middlewares.MockHeaderMiddleware(func(c echo.Context) error {
				ctx := c.Request().Context()
				val, ok := ctx.Value(consts.MockIDContextKey).(string)
				if tt.wantCtx {
					if !ok {
						t.Errorf("expected context value for mock ID, but got none")
					}
					if val != tt.wantMockID {
						t.Errorf("expected mock ID %q, got %q", tt.wantMockID, val)
					}
				} else {
					if ok || val != "" {
						t.Errorf("expected no context value for mock ID, got %q", val)
					}
				}
				return c.String(http.StatusOK, "OK")
			})

			_ = handler(c)
		})
	}
}
