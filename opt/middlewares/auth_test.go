package middlewares_test

import (
	"bank-api/opt/middlewares"
	"bank-api/pkg/jwt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestNewJWTAuth(t *testing.T) {
	secret := "test-secret-key-32-chars-long!!"

	// Generate a valid token for testing
	gen := jwt.NewGenerator(jwt.Config{
		Secret:     secret,
		Expiration: time.Hour,
	})
	validOutput, _ := gen.Generate(jwt.GenerateInput{
		StaffID:   "staff-123",
		CompanyID: "company-456",
		Slug:      "my-gym",
		Email:     "test@example.com",
		Role:      "admin",
	})

	// Generate an expired token
	expiredGen := jwt.NewGenerator(jwt.Config{
		Secret:     secret,
		Expiration: -time.Hour,
	})
	expiredOutput, _ := expiredGen.Generate(jwt.GenerateInput{
		StaffID:   "staff-999",
		CompanyID: "company-888",
		Slug:      "old-gym",
		Email:     "expired@example.com",
		Role:      "user",
	})

	// Generate a token with wrong secret
	wrongSecretGen := jwt.NewGenerator(jwt.Config{
		Secret:     "wrong-secret-key-32-chars-long!",
		Expiration: time.Hour,
	})
	wrongSecretOutput, _ := wrongSecretGen.Generate(jwt.GenerateInput{
		StaffID:   "staff-111",
		CompanyID: "company-222",
		Slug:      "wrong-gym",
		Email:     "wrong@example.com",
		Role:      "user",
	})

	tests := []struct {
		name          string
		authHeader    string
		wantStatus    int
		wantStaffID   string
		wantCompanyID string
		wantEmail     string
		wantRole      string
	}{
		{
			name:       "error - missing Authorization header returns 401",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "error - invalid Bearer format returns 401",
			authHeader: "InvalidFormat token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "error - missing Bearer prefix returns 401",
			authHeader: validOutput.Token,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "error - only Bearer without token returns 401",
			authHeader: "Bearer",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "error - Bearer with empty token returns 401",
			authHeader: "Bearer ",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "error - invalid token returns 401",
			authHeader: "Bearer invalid.token.here",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "error - expired token returns 401",
			authHeader: "Bearer " + expiredOutput.Token,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "error - token with wrong secret returns 401",
			authHeader: "Bearer " + wrongSecretOutput.Token,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "error - malformed token returns 401",
			authHeader: "Bearer not-a-valid-token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:          "success - valid token passes through",
			authHeader:    "Bearer " + validOutput.Token,
			wantStatus:    http.StatusOK,
			wantStaffID:   "staff-123",
			wantCompanyID: "company-456",
			wantEmail:     "test@example.com",
			wantRole:      "admin",
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

			middleware := middlewares.NewJWTAuth(secret)

			// Track whether next handler was called
			nextCalled := false
			handler := middleware(func(c echo.Context) error {
				nextCalled = true

				// Verify context values for successful cases
				if tt.wantStatus == http.StatusOK {
					if staffID := c.Get("staff_id"); staffID != tt.wantStaffID {
						t.Errorf("staff_id = %v, want %v", staffID, tt.wantStaffID)
					}
					if companyID := c.Get("company_id"); companyID != tt.wantCompanyID {
						t.Errorf("company_id = %v, want %v", companyID, tt.wantCompanyID)
					}
					if email := c.Get("email"); email != tt.wantEmail {
						t.Errorf("email = %v, want %v", email, tt.wantEmail)
					}
					if role := c.Get("role"); role != tt.wantRole {
						t.Errorf("role = %v, want %v", role, tt.wantRole)
					}
				}

				return c.String(http.StatusOK, "OK")
			})

			err := handler(c)

			// Check if we got an HTTP error for unauthorized cases
			if tt.wantStatus == http.StatusUnauthorized {
				if err == nil {
					t.Error("expected error for unauthorized case, got nil")
				}
				if httpErr, ok := err.(*echo.HTTPError); ok {
					if httpErr.Code != http.StatusUnauthorized {
						t.Errorf("HTTP error code = %v, want %v", httpErr.Code, http.StatusUnauthorized)
					}
				}
				if nextCalled {
					t.Error("next handler should not be called for unauthorized case")
				}
			}

			// Check successful case
			if tt.wantStatus == http.StatusOK {
				if err != nil {
					t.Errorf("unexpected error for authorized case: %v", err)
				}
				if !nextCalled {
					t.Error("next handler should be called for authorized case")
				}
				if rec.Code != http.StatusOK {
					t.Errorf("HTTP status = %v, want %v", rec.Code, http.StatusOK)
				}
			}
		})
	}
}

func TestNewJWTAuth_ContextValues(t *testing.T) {
	secret := "test-secret-key-32-chars-long!!"

	tests := []struct {
		name      string
		staffID   string
		companyID string
		slug      string
		email     string
		role      string
	}{
		{
			name:      "sets correct context values for admin",
			staffID:   "staff-admin-1",
			companyID: "company-1",
			slug:      "gym-one",
			email:     "admin@example.com",
			role:      "admin",
		},
		{
			name:      "sets correct context values for user",
			staffID:   "staff-user-2",
			companyID: "company-2",
			slug:      "gym-two",
			email:     "user@example.com",
			role:      "user",
		},
		{
			name:      "sets correct context values for manager",
			staffID:   "staff-mgr-3",
			companyID: "company-3",
			slug:      "gym-three",
			email:     "manager@example.com",
			role:      "manager",
		},
		{
			name:      "handles empty role",
			staffID:   "staff-4",
			companyID: "company-4",
			slug:      "gym-four",
			email:     "test@example.com",
			role:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := jwt.NewGenerator(jwt.Config{
				Secret:     secret,
				Expiration: time.Hour,
			})
			output, err := gen.Generate(jwt.GenerateInput{
				StaffID:   tt.staffID,
				CompanyID: tt.companyID,
				Slug:      tt.slug,
				Email:     tt.email,
				Role:      tt.role,
			})
			if err != nil {
				t.Fatalf("failed to generate token: %v", err)
			}

			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+output.Token)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			middleware := middlewares.NewJWTAuth(secret)
			handler := middleware(func(c echo.Context) error {
				// Verify all context values
				staffID := c.Get("staff_id")
				if staffID != tt.staffID {
					t.Errorf("staff_id = %v, want %v", staffID, tt.staffID)
				}

				companyID := c.Get("company_id")
				if companyID != tt.companyID {
					t.Errorf("company_id = %v, want %v", companyID, tt.companyID)
				}

				email := c.Get("email")
				if email != tt.email {
					t.Errorf("email = %v, want %v", email, tt.email)
				}

				role := c.Get("role")
				if role != tt.role {
					t.Errorf("role = %v, want %v", role, tt.role)
				}

				return c.String(http.StatusOK, "OK")
			})

			err = handler(c)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewJWTAuth_MultipleRequests(t *testing.T) {
	// Test that the middleware handles multiple requests correctly
	secret := "test-secret-key-32-chars-long!!"
	gen := jwt.NewGenerator(jwt.Config{
		Secret:     secret,
		Expiration: time.Hour,
	})

	// Create tokens for different users
	output1, _ := gen.Generate(jwt.GenerateInput{
		StaffID:   "staff-1",
		CompanyID: "company-1",
		Slug:      "gym-1",
		Email:     "user1@example.com",
		Role:      "admin",
	})
	output2, _ := gen.Generate(jwt.GenerateInput{
		StaffID:   "staff-2",
		CompanyID: "company-2",
		Slug:      "gym-2",
		Email:     "user2@example.com",
		Role:      "user",
	})

	middleware := middlewares.NewJWTAuth(secret)

	// First request
	e1 := echo.New()
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set("Authorization", "Bearer "+output1.Token)
	rec1 := httptest.NewRecorder()
	c1 := e1.NewContext(req1, rec1)

	handler1 := middleware(func(c echo.Context) error {
		if c.Get("staff_id") != "staff-1" {
			t.Errorf("request 1: staff_id = %v, want staff-1", c.Get("staff_id"))
		}
		return c.String(http.StatusOK, "OK")
	})

	if err := handler1(c1); err != nil {
		t.Errorf("request 1 failed: %v", err)
	}

	// Second request
	e2 := echo.New()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("Authorization", "Bearer "+output2.Token)
	rec2 := httptest.NewRecorder()
	c2 := e2.NewContext(req2, rec2)

	handler2 := middleware(func(c echo.Context) error {
		if c.Get("staff_id") != "staff-2" {
			t.Errorf("request 2: staff_id = %v, want staff-2", c.Get("staff_id"))
		}
		return c.String(http.StatusOK, "OK")
	})

	if err := handler2(c2); err != nil {
		t.Errorf("request 2 failed: %v", err)
	}
}

func TestNewJWTAuth_BearerCaseSensitivity(t *testing.T) {
	secret := "test-secret-key-32-chars-long!!"
	gen := jwt.NewGenerator(jwt.Config{
		Secret:     secret,
		Expiration: time.Hour,
	})
	validOutput, _ := gen.Generate(jwt.GenerateInput{
		StaffID:   "staff-123",
		CompanyID: "company-456",
		Slug:      "my-gym",
		Email:     "test@example.com",
		Role:      "admin",
	})

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "Bearer with capital B works",
			authHeader: "Bearer " + validOutput.Token,
			wantStatus: http.StatusOK,
		},
		{
			name:       "bearer with lowercase b works (case-insensitive)",
			authHeader: "bearer " + validOutput.Token,
			wantStatus: http.StatusOK,
		},
		{
			name:       "BEARER with all caps works (case-insensitive)",
			authHeader: "BEARER " + validOutput.Token,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tt.authHeader)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			middleware := middlewares.NewJWTAuth(secret)
			handler := middleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "OK")
			})

			err := handler(c)

			if tt.wantStatus == http.StatusUnauthorized {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestNewJWTAuth_ExtraSpaces(t *testing.T) {
	secret := "test-secret-key-32-chars-long!!"
	gen := jwt.NewGenerator(jwt.Config{
		Secret:     secret,
		Expiration: time.Hour,
	})
	validOutput, _ := gen.Generate(jwt.GenerateInput{
		StaffID:   "staff-123",
		CompanyID: "company-456",
		Slug:      "my-gym",
		Email:     "test@example.com",
		Role:      "admin",
	})

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "normal Bearer token works",
			authHeader: "Bearer " + validOutput.Token,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Bearer with multiple spaces fails",
			authHeader: "Bearer  " + validOutput.Token,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "Bearer with tab instead of space fails",
			authHeader: "Bearer\t" + validOutput.Token,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", tt.authHeader)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			middleware := middlewares.NewJWTAuth(secret)
			handler := middleware(func(c echo.Context) error {
				return c.String(http.StatusOK, "OK")
			})

			err := handler(c)

			if tt.wantStatus == http.StatusUnauthorized {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
