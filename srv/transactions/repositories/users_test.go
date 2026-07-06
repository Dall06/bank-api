package repositories_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"bank-api/pkg/sigil"
	"bank-api/srv/transactions/repositories"
)

func TestUsersRepository_ValidateUser(t *testing.T) {
	tests := []struct {
		name       string
		accountID  string
		mockStatus int
		mockBody   string
		wantErr    bool
	}{
		{
			name:       "valid user",
			accountID:  "acc-123",
			mockStatus: http.StatusOK,
			mockBody:   `{"id":"acc-123"}`,
			wantErr:    false,
		},
		{
			name:       "user not found",
			accountID:  "acc-not-found",
			mockStatus: http.StatusNotFound,
			mockBody:   `{"error":"not found"}`,
			wantErr:    true,
		},
		{
			name:       "internal server error",
			accountID:  "acc-error",
			mockStatus: http.StatusInternalServerError,
			mockBody:   `{"error":"server error"}`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatus)
				_, _ = w.Write([]byte(tt.mockBody))
			}))
			defer server.Close()

			config := sigil.DefaultConfig("test-secret", "test")
			signer := sigil.NewSigner(config)

			repo := repositories.NewUsersRepository(server.URL, signer)

			err := repo.ValidateUser(context.Background(), tt.accountID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUsersRepository_ValidateUser_InvalidURL(t *testing.T) {
	// Request creation should fail if URL contains invalid control characters
	config := sigil.DefaultConfig("test-secret", "test")
	signer := sigil.NewSigner(config)

	repo := repositories.NewUsersRepository("http://\x00invalid-url", signer)
	err := repo.ValidateUser(context.Background(), "123")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}
