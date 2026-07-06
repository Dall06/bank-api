package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bank-api/mocks/providers/domain"
	"bank-api/mocks/providers/handlers"
	"github.com/labstack/echo/v4"
)

type mockUsecase struct {
	ExecuteFunc func(ctx context.Context, req domain.ExecuteRequest) (domain.ExecuteResponse, error)
}

func (m *mockUsecase) Execute(ctx context.Context, req domain.ExecuteRequest) (domain.ExecuteResponse, error) {
	return m.ExecuteFunc(ctx, req)
}

func TestExecuteHandler(t *testing.T) {
	tests := []struct {
		name         string
		reqBody      interface{}
		mockFunc     func(ctx context.Context, req domain.ExecuteRequest) (domain.ExecuteResponse, error)
		expectedCode int
	}{
		{
			name: "Success",
			reqBody: domain.ExecuteRequest{
				AccountID: "acc-123456",
				Type:      "CREDIT",
				Amount:    1500.00,
				Currency:  "MXN",
			},
			mockFunc: func(ctx context.Context, req domain.ExecuteRequest) (domain.ExecuteResponse, error) {
				return domain.ExecuteResponse{
					TransactionID: "txn-789",
					Status:        "APPROVED",
					Balance:       5500.00,
					ExecutedAt:    "2025-03-15T10:30:00Z",
				}, nil
			},
			expectedCode: http.StatusOK,
		},
		{
			name: "Failure",
			reqBody: domain.ExecuteRequest{
				AccountID: "acc-123456",
				Type:      "CREDIT",
				Amount:    15000.00,
				Currency:  "MXN",
			},
			mockFunc: func(ctx context.Context, req domain.ExecuteRequest) (domain.ExecuteResponse, error) {
				return domain.ExecuteResponse{}, &domain.ProviderError{
					Code:    "INSUFFICIENT_FUNDS",
					Message: "The account does not have enough balance to complete the transaction",
				}
			},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/provider/v1/execute", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mock := &mockUsecase{ExecuteFunc: tt.mockFunc}
			h := handlers.NewProviderHandler(mock)

			err := h.Execute(c)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if rec.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, rec.Code)
			}
		})
	}
}

func TestMockInterceptorMiddleware(t *testing.T) {
	e := echo.New()
	mock := &mockUsecase{}
	h := handlers.NewProviderHandler(mock)

	tests := []struct {
		name       string
		mockID     string
		wantStatus int
	}{
		{"insufficient funds", "id-insufficient-funds", http.StatusBadRequest},
		{"error 500", "id-error-500", http.StatusInternalServerError},
		{"no header", "", http.StatusOK},
		{"invalid header", "unknown", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.mockID != "" {
				req.Header.Set("X-Mock-Id", tt.mockID)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := h.MockInterceptorMiddleware(func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			_ = handler(c)
			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}
