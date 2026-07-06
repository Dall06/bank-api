package middlewares_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bank-api/opt/middlewares"
	"bank-api/pkg/jwt"

	"github.com/labstack/echo/v4"
)

func TestNewOptionalJWTAuth(t *testing.T) {
	secret := "test-secret-key-32-chars-long!!"
	gen := jwt.NewGenerator(jwt.Config{
		Secret:     secret,
		Expiration: time.Hour,
	})
	validOutput, _ := gen.Generate(jwt.GenerateInput{
		StaffID:   "staff-123",
		CompanyID: "company-456",
		Role:      "admin",
	})

	tests := []struct {
		name        string
		authHeader  string
		wantStaffID string
		wantRole    string
	}{
		{
			name:       "no token proceeds without error",
			authHeader: "",
		},
		{
			name:       "invalid token proceeds without error",
			authHeader: "Bearer invalid",
		},
		{
			name:        "valid token sets claims",
			authHeader:  "Bearer " + validOutput.Token,
			wantStaffID: "staff-123",
			wantRole:    "admin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			handler := middlewares.NewOptionalJWTAuth(secret)(func(c echo.Context) error {
				if tt.wantStaffID != "" {
					if c.Get("staff_id") != tt.wantStaffID {
						t.Errorf("expected staff_id %q, got %v", tt.wantStaffID, c.Get("staff_id"))
					}
				} else {
					if c.Get("staff_id") != nil {
						t.Errorf("expected no staff_id, got %v", c.Get("staff_id"))
					}
				}
				return c.NoContent(http.StatusOK)
			})

			err := handler(c)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestRoleMiddlewares(t *testing.T) {
	tests := []struct {
		name       string
		middleware echo.MiddlewareFunc
		role       interface{}
		wantStatus int
	}{
		// RequireGod
		{"RequireGod with god role", middlewares.RequireGod(), "god", http.StatusOK},
		{"RequireGod with admin role", middlewares.RequireGod(), "admin", http.StatusForbidden},
		{"RequireGod with no role", middlewares.RequireGod(), nil, http.StatusForbidden},

		// RequireAdminOrGod
		{"RequireAdminOrGod with god role", middlewares.RequireAdminOrGod(), "god", http.StatusOK},
		{"RequireAdminOrGod with admin role", middlewares.RequireAdminOrGod(), "admin", http.StatusOK},
		{"RequireAdminOrGod with owner role", middlewares.RequireAdminOrGod(), "owner", http.StatusForbidden},
		{"RequireAdminOrGod with no role", middlewares.RequireAdminOrGod(), nil, http.StatusForbidden},
		{"RequireAdminOrGod with wrong type", middlewares.RequireAdminOrGod(), 123, http.StatusForbidden},

		// RequireGodOrOwner
		{"RequireGodOrOwner with god role", middlewares.RequireGodOrOwner(), "god", http.StatusOK},
		{"RequireGodOrOwner with owner role", middlewares.RequireGodOrOwner(), "owner", http.StatusOK},
		{"RequireGodOrOwner with admin role", middlewares.RequireGodOrOwner(), "admin", http.StatusForbidden},
		{"RequireGodOrOwner with no role", middlewares.RequireGodOrOwner(), nil, http.StatusForbidden},
		{"RequireGodOrOwner with wrong type", middlewares.RequireGodOrOwner(), 123, http.StatusForbidden},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.role != nil {
				c.Set("role", tt.role)
			}

			handler := tt.middleware(func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			err := handler(c)

			if tt.wantStatus == http.StatusOK {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if he, ok := err.(*echo.HTTPError); ok {
					if he.Code != tt.wantStatus {
						t.Errorf("expected status %d, got %d", tt.wantStatus, he.Code)
					}
				} else {
					t.Errorf("expected HTTP error, got %v", err)
				}
			}
		})
	}
}

type mockCompanyStatusChecker struct {
	suspended bool
	err       error
}

func (m *mockCompanyStatusChecker) IsCompanySuspended(ctx echo.Context, companyID string) (bool, error) {
	return m.suspended, m.err
}

func TestNewCompanyStatusCheck(t *testing.T) {
	tests := []struct {
		name       string
		companyID  interface{}
		suspended  bool
		err        error
		wantStatus int
	}{
		{"no company id skips check", nil, false, nil, http.StatusOK},
		{"empty company id skips check", "", false, nil, http.StatusOK},
		{"invalid company id type skips check", 123, false, nil, http.StatusOK},
		{"valid active company passes", "company-1", false, nil, http.StatusOK},
		{"valid suspended company fails", "company-2", true, nil, http.StatusPaymentRequired},
		{"error during check returns 503", "company-3", false, errors.New("db error"), http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.companyID != nil {
				c.Set("company_id", tt.companyID)
			}

			checker := &mockCompanyStatusChecker{
				suspended: tt.suspended,
				err:       tt.err,
			}

			handler := middlewares.NewCompanyStatusCheck(checker)(func(c echo.Context) error {
				return c.NoContent(http.StatusOK)
			})

			err := handler(c)

			// Wait, the middleware returns c.JSON() for errors instead of return error.
			// So err will be nil, but the response code will be set.
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if rec.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rec.Code)
			}
		})
	}
}
