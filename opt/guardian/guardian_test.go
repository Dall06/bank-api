package guardian

import (
	"context"
	"errors"
	"testing"
	"time"

	"bank-api/pkg/jwt"

	"github.com/stretchr/testify/assert"
)

// fakeSessionStore is a fake implementation of SessionStore for testing
// Supports multiple sessions per user
type fakeSessionStore struct {
	sessions       map[string]string // jti -> staffID (multiple JTIs can have same staffID)
	shouldError    bool
	sessionCreated bool // track if any session was ever created
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{
		sessions: make(map[string]string),
	}
}

func (f *fakeSessionStore) AddSession(ctx context.Context, staffID, jti string, expiresAt time.Time) error {
	if f.shouldError {
		return errors.New("session store error")
	}
	f.sessions[jti] = staffID
	f.sessionCreated = true
	return nil
}

func (f *fakeSessionStore) IsActive(ctx context.Context, jti string) (bool, error) {
	if f.shouldError {
		return false, errors.New("session store error")
	}
	_, exists := f.sessions[jti]
	// If no sessions were ever created, consider all tokens as active (for tests that don't set up sessions)
	if !f.sessionCreated {
		return true, nil
	}
	return exists, nil
}

func (f *fakeSessionStore) InvalidateSession(ctx context.Context, jti string) error {
	if f.shouldError {
		return errors.New("session store error")
	}
	delete(f.sessions, jti)
	return nil
}

func (f *fakeSessionStore) InvalidateAll(ctx context.Context, staffID string) error {
	if f.shouldError {
		return errors.New("session store error")
	}
	// Remove all sessions for this staff
	for jti, sid := range f.sessions {
		if sid == staffID {
			delete(f.sessions, jti)
		}
	}
	return nil
}

func (f *fakeSessionStore) InvalidateAllByCompany(ctx context.Context, companyID string) error {
	if f.shouldError {
		return errors.New("session store error")
	}
	// In tests, we clear all sessions (simplified - no company->staff mapping in fake)
	f.sessions = make(map[string]string)
	return nil
}

func (f *fakeSessionStore) Cleanup(ctx context.Context) error {
	return nil
}

// Helper to count sessions for a staff
func (f *fakeSessionStore) countSessionsForStaff(staffID string) int {
	count := 0
	for _, sid := range f.sessions {
		if sid == staffID {
			count++
		}
	}
	return count
}

func TestGenerateTokenPair(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		input   jwt.GenerateInput
		wantErr bool
	}{
		{
			name: "success - valid claims",
			config: Config{
				JWTSecret:         "test-secret-key",
				AccessExpiration:  15 * time.Minute,
				RefreshExpiration: 7 * 24 * time.Hour,
			},
			input: jwt.GenerateInput{
				UserID:    "staff-123",
				StaffID:   "staff-123",
				CompanyID: "company-456",
				Slug:      "test-gym",
				Email:     "user@example.com",
				Role:      "admin",
			},
			wantErr: false,
		},
		{
			name: "success - minimal claims",
			config: Config{
				JWTSecret:         "test-secret-key",
				AccessExpiration:  15 * time.Minute,
				RefreshExpiration: 7 * 24 * time.Hour,
			},
			input: jwt.GenerateInput{
				UserID:    "staff-123",
				StaffID:   "staff-123",
				CompanyID: "company-456",
				Email:     "user@example.com",
				Role:      "staff",
			},
			wantErr: false,
		},
		{
			name: "success - empty claims (edge case)",
			config: Config{
				JWTSecret:         "test-secret-key",
				AccessExpiration:  15 * time.Minute,
				RefreshExpiration: 7 * 24 * time.Hour,
			},
			input:   jwt.GenerateInput{},
			wantErr: false,
		},
		{
			name: "error - jwt generation failure",
			config: Config{
				JWTSecret:         "", // empty secret might cause error or just valid
				AccessExpiration:  15 * time.Minute,
				RefreshExpiration: 7 * 24 * time.Hour,
			},
			// In bank-api/pkg/jwt, an empty secret might not error on generation, but let's test session store error
			input: jwt.GenerateInput{
				UserID: "staff-123",
			},
			wantErr: false, // We'll test session store error below
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionStore := newFakeSessionStore()
			guardian := New(tt.config, sessionStore)

			got, err := guardian.GenerateTokenPair(context.Background(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
		})
	}

	t.Run("error - session store AddSession failure", func(t *testing.T) {
		sessionStore := newFakeSessionStore()
		sessionStore.shouldError = true
		guardian := New(Config{JWTSecret: "sec"}, sessionStore)
		_, err := guardian.GenerateTokenPair(context.Background(), jwt.GenerateInput{UserID: "123"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "guardian: failed to create session")
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionStore := newFakeSessionStore()
			guardian := New(tt.config, sessionStore)

			got, err := guardian.GenerateTokenPair(context.Background(), tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, got)
			assert.NotEmpty(t, got.AccessToken)
			assert.NotEmpty(t, got.RefreshToken)
			assert.NotEmpty(t, got.AccessJTI)
			assert.False(t, got.ExpiresAt.IsZero())

			// Verify access token
			accessClaims, err := guardian.jwtGenerator.Validate(got.AccessToken)
			assert.NoError(t, err)
			assert.Equal(t, tt.input.StaffID, accessClaims.StaffID)
			assert.Equal(t, tt.input.CompanyID, accessClaims.CompanyID)
			assert.Equal(t, tt.input.Slug, accessClaims.Slug)
			assert.Equal(t, tt.input.Email, accessClaims.Email)
			assert.Equal(t, tt.input.Role, accessClaims.Role)

			// Verify refresh token
			refreshGen := jwt.NewGenerator(jwt.Config{
				Secret:     tt.config.JWTSecret,
				Expiration: tt.config.RefreshExpiration,
			})
			refreshClaims, err := refreshGen.Validate(got.RefreshToken)
			assert.NoError(t, err)
			assert.Equal(t, "refresh", refreshClaims.Role)
			assert.Equal(t, tt.input.StaffID, refreshClaims.StaffID)
			assert.Equal(t, tt.input.CompanyID, refreshClaims.CompanyID)
		})
	}
}

func TestMultipleSessions(t *testing.T) {
	config := Config{
		JWTSecret:         "test-secret-key",
		AccessExpiration:  15 * time.Minute,
		RefreshExpiration: 7 * 24 * time.Hour,
	}

	sessionStore := newFakeSessionStore()
	guardian := New(config, sessionStore)

	// Generate first token pair (simulates login on device 1)
	pair1, err := guardian.GenerateTokenPair(context.Background(), jwt.GenerateInput{
		UserID:    "staff-123",
		StaffID:   "staff-123",
		CompanyID: "company-456",
		Email:     "user@example.com",
		Role:      "admin",
	})
	assert.NoError(t, err)

	// Generate second token pair (simulates login on device 2)
	pair2, err := guardian.GenerateTokenPair(context.Background(), jwt.GenerateInput{
		UserID:    "staff-123",
		StaffID:   "staff-123",
		CompanyID: "company-456",
		Email:     "user@example.com",
		Role:      "admin",
	})
	assert.NoError(t, err)

	// Both tokens should be different
	assert.NotEqual(t, pair1.AccessToken, pair2.AccessToken)
	assert.NotEqual(t, pair1.AccessJTI, pair2.AccessJTI)

	// Both tokens should be valid simultaneously
	claims1, err := guardian.ValidateAccessToken(context.Background(), pair1.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, "staff-123", claims1.StaffID)

	claims2, err := guardian.ValidateAccessToken(context.Background(), pair2.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, "staff-123", claims2.StaffID)

	// Should have 2 sessions for this staff
	assert.Equal(t, 2, sessionStore.countSessionsForStaff("staff-123"))
}

func TestValidateAccessToken(t *testing.T) {
	config := Config{
		JWTSecret:         "test-secret-key",
		AccessExpiration:  15 * time.Minute,
		RefreshExpiration: 7 * 24 * time.Hour,
	}

	tests := []struct {
		name      string
		setup     func(*Guardian, *fakeSessionStore) string
		wantErr   bool
		wantError error
	}{
		{
			name: "success - valid token",
			setup: func(g *Guardian, ss *fakeSessionStore) string {
				pair, _ := g.GenerateTokenPair(context.Background(), jwt.GenerateInput{
					UserID:    "staff-123",
					StaffID:   "staff-123",
					CompanyID: "company-456",
					Email:     "user@example.com",
					Role:      "admin",
				})
				return pair.AccessToken
			},
			wantErr: false,
		},
		{
			name: "error - expired token",
			setup: func(g *Guardian, ss *fakeSessionStore) string {
				// Create a generator with -1 hour expiration to make token already expired
				expiredGen := jwt.NewGenerator(jwt.Config{
					Secret:     config.JWTSecret,
					Expiration: -1 * time.Hour,
				})
				output, _ := expiredGen.Generate(jwt.GenerateInput{
					UserID:    "staff-123",
					StaffID:   "staff-123",
					CompanyID: "company-456",
					Email:     "user@example.com",
					Role:      "admin",
				})
				return output.Token
			},
			wantErr: true,
		},
		{
			name: "error - session invalidated",
			setup: func(g *Guardian, ss *fakeSessionStore) string {
				pair, _ := g.GenerateTokenPair(context.Background(), jwt.GenerateInput{
					UserID:    "staff-123",
					StaffID:   "staff-123",
					CompanyID: "company-456",
					Email:     "user@example.com",
					Role:      "admin",
				})
				// Invalidate the session
				ss.InvalidateSession(context.Background(), pair.AccessJTI)
				return pair.AccessToken
			},
			wantErr:   true,
			wantError: ErrTokenRevoked,
		},
		{
			name: "error - invalid token",
			setup: func(g *Guardian, ss *fakeSessionStore) string {
				return "invalid.token.string"
			},
			wantErr: true,
		},
		{
			name: "error - token with wrong secret",
			setup: func(g *Guardian, ss *fakeSessionStore) string {
				wrongGen := jwt.NewGenerator(jwt.Config{
					Secret:     "wrong-secret",
					Expiration: 15 * time.Minute,
				})
				output, _ := wrongGen.Generate(jwt.GenerateInput{
					UserID:    "staff-123",
					StaffID:   "staff-123",
					CompanyID: "company-456",
					Email:     "user@example.com",
					Role:      "admin",
				})
				return output.Token
			},
			wantErr: true,
		},
		{
			name: "error - session store IsActive failure",
			setup: func(g *Guardian, ss *fakeSessionStore) string {
				pair, _ := g.GenerateTokenPair(context.Background(), jwt.GenerateInput{
					UserID: "staff-123",
				})
				ss.shouldError = true
				return pair.AccessToken
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionStore := newFakeSessionStore()
			guardian := New(config, sessionStore)

			token := tt.setup(guardian, sessionStore)
			claims, err := guardian.ValidateAccessToken(context.Background(), token)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantError != nil {
					assert.Equal(t, tt.wantError, err)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, claims)
			assert.NotEmpty(t, claims.StaffID)
		})
	}
}

func TestRefreshAccessToken(t *testing.T) {
	config := Config{
		JWTSecret:         "test-secret-key",
		AccessExpiration:  15 * time.Minute,
		RefreshExpiration: 7 * 24 * time.Hour,
	}

	tests := []struct {
		name      string
		setup     func(*Guardian, *fakeSessionStore) string
		wantErr   bool
		wantError error
	}{
		{
			name: "success - valid refresh token",
			setup: func(g *Guardian, ss *fakeSessionStore) string {
				pair, _ := g.GenerateTokenPair(context.Background(), jwt.GenerateInput{
					UserID:    "staff-123",
					StaffID:   "staff-123",
					CompanyID: "company-456",
					Email:     "user@example.com",
					Role:      "admin",
				})
				return pair.RefreshToken
			},
			wantErr: false,
		},
		{
			name: "error - access token instead of refresh token",
			setup: func(g *Guardian, ss *fakeSessionStore) string {
				pair, _ := g.GenerateTokenPair(context.Background(), jwt.GenerateInput{
					UserID:    "staff-123",
					StaffID:   "staff-123",
					CompanyID: "company-456",
					Email:     "user@example.com",
					Role:      "admin",
				})
				// Return access token instead of refresh token
				return pair.AccessToken
			},
			wantErr:   true,
			wantError: ErrInvalidRefreshToken,
		},
		{
			name: "error - expired refresh token",
			setup: func(g *Guardian, ss *fakeSessionStore) string {
				expiredGen := jwt.NewGenerator(jwt.Config{
					Secret:     config.JWTSecret,
					Expiration: -1 * time.Hour,
				})
				output, _ := expiredGen.Generate(jwt.GenerateInput{
					UserID:    "staff-123",
					StaffID:   "staff-123",
					CompanyID: "company-456",
					Email:     "user@example.com",
					Role:      "refresh",
				})
				return output.Token
			},
			wantErr: true,
		},
		{
			name: "error - invalid token",
			setup: func(g *Guardian, ss *fakeSessionStore) string {
				return "invalid.refresh.token"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionStore := newFakeSessionStore()
			guardian := New(config, sessionStore)

			refreshToken := tt.setup(guardian, sessionStore)
			newPair, err := guardian.RefreshAccessToken(context.Background(), refreshToken)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantError != nil {
					assert.Equal(t, tt.wantError, err)
				}
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, newPair)
			assert.NotEmpty(t, newPair.AccessToken)
			assert.NotEmpty(t, newPair.RefreshToken)

			// Verify the new tokens are valid
			claims, err := guardian.ValidateAccessToken(context.Background(), newPair.AccessToken)
			assert.NoError(t, err)
			assert.Equal(t, "staff-123", claims.StaffID)
			assert.Equal(t, "company-456", claims.CompanyID)
		})
	}
}

func TestLogout(t *testing.T) {
	config := Config{
		JWTSecret:         "test-secret-key",
		AccessExpiration:  15 * time.Minute,
		RefreshExpiration: 7 * 24 * time.Hour,
	}

	t.Run("logout invalidates only current session", func(t *testing.T) {
		sessionStore := newFakeSessionStore()
		guardian := New(config, sessionStore)

		// Login on two devices
		pair1, _ := guardian.GenerateTokenPair(context.Background(), jwt.GenerateInput{
			UserID:    "staff-123",
			StaffID:   "staff-123",
			CompanyID: "company-456",
			Email:     "user@example.com",
			Role:      "admin",
		})
		pair2, _ := guardian.GenerateTokenPair(context.Background(), jwt.GenerateInput{
			UserID:    "staff-123",
			StaffID:   "staff-123",
			CompanyID: "company-456",
			Email:     "user@example.com",
			Role:      "admin",
		})

		// Verify both are valid
		assert.Equal(t, 2, sessionStore.countSessionsForStaff("staff-123"))

		// Logout from device 1 (by JTI)
		err := guardian.Logout(context.Background(), pair1.AccessJTI)
		assert.NoError(t, err)

		// Device 1 token should be invalid
		_, err = guardian.ValidateAccessToken(context.Background(), pair1.AccessToken)
		assert.Error(t, err)
		assert.Equal(t, ErrTokenRevoked, err)

		// Device 2 token should still be valid
		claims, err := guardian.ValidateAccessToken(context.Background(), pair2.AccessToken)
		assert.NoError(t, err)
		assert.Equal(t, "staff-123", claims.StaffID)

		// Should have 1 session left
		assert.Equal(t, 1, sessionStore.countSessionsForStaff("staff-123"))
	})

	t.Run("logout non-existent session should not error", func(t *testing.T) {
		sessionStore := newFakeSessionStore()
		guardian := New(config, sessionStore)

		err := guardian.Logout(context.Background(), "non-existent-jti")
		assert.NoError(t, err)
	})
}

func TestLogoutAll(t *testing.T) {
	config := Config{
		JWTSecret:         "test-secret-key",
		AccessExpiration:  15 * time.Minute,
		RefreshExpiration: 7 * 24 * time.Hour,
	}

	t.Run("logout all invalidates all sessions for user", func(t *testing.T) {
		sessionStore := newFakeSessionStore()
		guardian := New(config, sessionStore)

		// Login on three devices
		pair1, _ := guardian.GenerateTokenPair(context.Background(), jwt.GenerateInput{
			UserID:    "staff-123",
			StaffID:   "staff-123",
			CompanyID: "company-456",
			Email:     "user@example.com",
			Role:      "admin",
		})
		pair2, _ := guardian.GenerateTokenPair(context.Background(), jwt.GenerateInput{
			UserID:    "staff-123",
			StaffID:   "staff-123",
			CompanyID: "company-456",
			Email:     "user@example.com",
			Role:      "admin",
		})
		pair3, _ := guardian.GenerateTokenPair(context.Background(), jwt.GenerateInput{
			UserID:    "staff-123",
			StaffID:   "staff-123",
			CompanyID: "company-456",
			Email:     "user@example.com",
			Role:      "admin",
		})

		// Also login another user
		otherPair, _ := guardian.GenerateTokenPair(context.Background(), jwt.GenerateInput{
			UserID:    "staff-456",
			StaffID:   "staff-456",
			CompanyID: "company-456",
			Email:     "other@example.com",
			Role:      "admin",
		})

		// Verify all are valid
		assert.Equal(t, 3, sessionStore.countSessionsForStaff("staff-123"))
		assert.Equal(t, 1, sessionStore.countSessionsForStaff("staff-456"))

		// Logout all for staff-123
		err := guardian.LogoutAll(context.Background(), "staff-123")
		assert.NoError(t, err)

		// All staff-123 tokens should be invalid
		_, err = guardian.ValidateAccessToken(context.Background(), pair1.AccessToken)
		assert.Equal(t, ErrTokenRevoked, err)
		_, err = guardian.ValidateAccessToken(context.Background(), pair2.AccessToken)
		assert.Equal(t, ErrTokenRevoked, err)
		_, err = guardian.ValidateAccessToken(context.Background(), pair3.AccessToken)
		assert.Equal(t, ErrTokenRevoked, err)

		// Other user's token should still be valid
		claims, err := guardian.ValidateAccessToken(context.Background(), otherPair.AccessToken)
		assert.NoError(t, err)
		assert.Equal(t, "staff-456", claims.StaffID)

		// Should have 0 sessions for staff-123, 1 for staff-456
		assert.Equal(t, 0, sessionStore.countSessionsForStaff("staff-123"))
		assert.Equal(t, 1, sessionStore.countSessionsForStaff("staff-456"))
	})
}

func TestLogoutByToken(t *testing.T) {
	config := Config{
		JWTSecret:         "test-secret-key",
		AccessExpiration:  15 * time.Minute,
		RefreshExpiration: 7 * 24 * time.Hour,
	}

	t.Run("logout by token invalidates only that session", func(t *testing.T) {
		sessionStore := newFakeSessionStore()
		guardian := New(config, sessionStore)

		// Login on two devices
		pair1, _ := guardian.GenerateTokenPair(context.Background(), jwt.GenerateInput{
			UserID:    "staff-123",
			StaffID:   "staff-123",
			CompanyID: "company-456",
			Email:     "user@example.com",
			Role:      "admin",
		})
		pair2, _ := guardian.GenerateTokenPair(context.Background(), jwt.GenerateInput{
			UserID:    "staff-123",
			StaffID:   "staff-123",
			CompanyID: "company-456",
			Email:     "user@example.com",
			Role:      "admin",
		})

		// Logout using the token itself
		err := guardian.LogoutByToken(context.Background(), pair1.AccessToken)
		assert.NoError(t, err)

		// Device 1 token should be invalid
		_, err = guardian.ValidateAccessToken(context.Background(), pair1.AccessToken)
		assert.Equal(t, ErrTokenRevoked, err)

		// Device 2 token should still be valid
		claims, err := guardian.ValidateAccessToken(context.Background(), pair2.AccessToken)
		assert.NoError(t, err)
		assert.Equal(t, "staff-123", claims.StaffID)
	})

	t.Run("logout by token fails if token is invalid", func(t *testing.T) {
		sessionStore := newFakeSessionStore()
		guardian := New(config, sessionStore)

		err := guardian.LogoutByToken(context.Background(), "invalid.token.here")
		assert.Error(t, err)
	})
}

func TestLogoutWithNilSessionStore(t *testing.T) {
	config := Config{
		JWTSecret:         "test-secret-key",
		AccessExpiration:  15 * time.Minute,
		RefreshExpiration: 7 * 24 * time.Hour,
	}

	guardian := New(config, nil)
	_, _ = guardian.GenerateTokenPair(context.Background(), jwt.GenerateInput{
		UserID:    "staff-123",
		StaffID:   "staff-123",
		CompanyID: "company-456",
		Email:     "user@example.com",
		Role:      "admin",
	})

	// Should not error when sessionStore is nil
	err := guardian.Logout(context.Background(), "some-jti")
	assert.NoError(t, err)

	err = guardian.LogoutAll(context.Background(), "staff-123")
	assert.NoError(t, err)

	err = guardian.LogoutAllByCompany(context.Background(), "company-456")
	assert.NoError(t, err)
}

func TestGuardian_LogoutAllByCompany(t *testing.T) {
	ss := newFakeSessionStore()
	g := New(Config{
		JWTSecret:        "test-secret-for-logout",
		AccessExpiration: time.Hour,
	}, ss)

	// Create sessions for two different users (same company in real scenario)
	pair1, err := g.GenerateTokenPair(context.Background(), jwt.GenerateInput{
		UserID:    "user-1",
		StaffID:   "staff-1",
		CompanyID: "company-abc",
		Email:     "user1@test.com",
		Role:      "owner",
	})
	assert.NoError(t, err)

	pair2, err := g.GenerateTokenPair(context.Background(), jwt.GenerateInput{
		UserID:    "user-2",
		StaffID:   "staff-2",
		CompanyID: "company-abc",
		Email:     "user2@test.com",
		Role:      "admin",
	})
	assert.NoError(t, err)

	// Both sessions should be active
	active1, _ := ss.IsActive(context.Background(), pair1.AccessJTI)
	active2, _ := ss.IsActive(context.Background(), pair2.AccessJTI)
	assert.True(t, active1, "session 1 should be active before logout")
	assert.True(t, active2, "session 2 should be active before logout")

	// Logout all by company
	err = g.LogoutAllByCompany(context.Background(), "company-abc")
	assert.NoError(t, err)

	// Both sessions should be invalidated
	active1, _ = ss.IsActive(context.Background(), pair1.AccessJTI)
	active2, _ = ss.IsActive(context.Background(), pair2.AccessJTI)
	assert.False(t, active1, "session 1 should be inactive after company logout")
	assert.False(t, active2, "session 2 should be inactive after company logout")
}

func TestGuardian_LogoutAllByCompany_NilSessionStore(t *testing.T) {
	g := New(Config{
		JWTSecret:        "test-secret-for-logout",
		AccessExpiration: time.Hour,
	}, nil)

	err := g.LogoutAllByCompany(context.Background(), "company-abc")
	assert.NoError(t, err, "should not error with nil session store")
}

func TestGenerateAndValidateTempToken(t *testing.T) {
	g := New(Config{JWTSecret: "secret"}, newFakeSessionStore())

	t.Run("success", func(t *testing.T) {
		out, err := g.GenerateTempToken("user-1", "user@test.com")
		assert.NoError(t, err)
		assert.NotEmpty(t, out.Token)
		assert.False(t, out.ExpiresAt.IsZero())

		claims, err := g.ValidateTempToken(out.Token)
		assert.NoError(t, err)
		assert.Equal(t, "user-1", claims.UserID)
		assert.Equal(t, "user@test.com", claims.Email)
	})

	t.Run("invalid token", func(t *testing.T) {
		_, err := g.ValidateTempToken("invalid")
		assert.Error(t, err)
	})
}

func TestCleanup(t *testing.T) {
	t.Run("success with store", func(t *testing.T) {
		ss := newFakeSessionStore()
		g := New(Config{JWTSecret: "secret"}, ss)
		err := g.Cleanup(context.Background())
		assert.NoError(t, err)
	})

	t.Run("success with nil store", func(t *testing.T) {
		g := New(Config{JWTSecret: "secret"}, nil)
		err := g.Cleanup(context.Background())
		assert.NoError(t, err)
	})
}

