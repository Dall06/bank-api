package usecases_test

import (
	"context"
	"testing"
	
	"bank-api/mocks/providers/domain"
	"bank-api/mocks/providers/usecases"
)

func TestExecute(t *testing.T) {
	usecase := usecases.NewProviderUsecase()

	tests := []struct {
		name    string
		req     domain.ExecuteRequest
		wantErr bool
		errCode string
	}{
		{
			name: "Success - Under 10000",
			req: domain.ExecuteRequest{
				AccountID: "acc-123456",
				Type:      "CREDIT",
				Amount:    1500.0,
				Currency:  "MXN",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := usecase.Execute(context.Background(), tt.req)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else {
					if providerErr, ok := err.(*domain.ProviderError); ok {
						if providerErr.Code != tt.errCode {
							t.Errorf("expected error code %s, got %s", tt.errCode, providerErr.Code)
						}
					} else {
						t.Errorf("expected ProviderError type")
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if resp.Status != "APPROVED" {
					t.Errorf("expected status APPROVED, got %s", resp.Status)
				}
			}
		})
	}
}
