// Package guardian provides robust JWT authentication with refresh tokens,
// token revocation, and role-based access control (RBAC).
package guardian

import (
	"context"
	"fmt"
	"time"

	"bank-api/pkg/jwt"

	"github.com/google/uuid"
)

// Permission constants are defined in opt/guardian/rbac.go
// e.g., PermPlansCreate, PermMembersRead, etc.

// Config holds guardian configuration
type Config struct {
	JWTSecret         string
	AccessExpiration  time.Duration
	RefreshExpiration time.Duration
}

// TokenPair represents an access token and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	AccessJTI    string    `json:"-"`
	RefreshJTI   string    `json:"-"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Guardian handles token generation, validation, and session management
type Guardian struct {
	config       Config
	jwtGenerator *jwt.Generator
	sessionStore SessionStore
}

// New creates a new Guardian instance
func New(config Config, sessionStore SessionStore) *Guardian {
	jwtGen := jwt.NewGenerator(jwt.Config{
		Secret:     config.JWTSecret,
		Expiration: config.AccessExpiration,
	})

	return &Guardian{
		config:       config,
		jwtGenerator: jwtGen,
		sessionStore: sessionStore,
	}
}

// GenerateTokenPair creates a new access/refresh token pair
// This registers a new session (multiple sessions per user are allowed)
func (g *Guardian) GenerateTokenPair(ctx context.Context, input jwt.GenerateInput) (*TokenPair, error) {
	// Generate access token
	accessOutput, err := g.jwtGenerator.Generate(input)
	if err != nil {
		return nil, err
	}

	// Register the new session (allows multiple sessions per user).
	// Sessions are keyed by user_id because auth identity is per-user, not
	// per-staff (one user can have staff entries in multiple companies).
	if g.sessionStore != nil {
		err = g.sessionStore.AddSession(ctx, input.UserID, accessOutput.JTI, accessOutput.ExpiresAt)
		if err != nil {
			return nil, fmt.Errorf("guardian: failed to create session: %w", err)
		}
	}

	// Generate refresh token (longer expiration)
	refreshJTI := uuid.New().String()
	refreshGen := jwt.NewGenerator(jwt.Config{
		Secret:     g.config.JWTSecret,
		Expiration: g.config.RefreshExpiration,
	})

	refreshOutput, err := refreshGen.Generate(jwt.GenerateInput{
		UserID:    input.UserID,
		StaffID:   input.StaffID,
		CompanyID: input.CompanyID,
		Slug:      input.Slug,
		Email:     input.Email,
		Role:      "refresh", // Special role to identify refresh tokens
	})
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessOutput.Token,
		RefreshToken: refreshOutput.Token,
		AccessJTI:    accessOutput.JTI,
		RefreshJTI:   refreshJTI,
		ExpiresAt:    accessOutput.ExpiresAt,
	}, nil
}

// ValidateAccessToken validates an access token and returns claims
func (g *Guardian) ValidateAccessToken(ctx context.Context, token string) (*jwt.Claims, error) {
	claims, err := g.jwtGenerator.Validate(token)
	if err != nil {
		return nil, err
	}

	// Check if session exists and is not expired
	if g.sessionStore != nil {
		active, err := g.sessionStore.IsActive(ctx, claims.ID)
		if err != nil {
			return nil, err
		}
		if !active {
			return nil, ErrTokenRevoked // Session invalidated or expired
		}
	}

	return claims, nil
}

// RefreshAccessToken creates a new access token from a valid refresh token
func (g *Guardian) RefreshAccessToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Validate refresh token
	refreshGen := jwt.NewGenerator(jwt.Config{
		Secret:     g.config.JWTSecret,
		Expiration: g.config.RefreshExpiration,
	})

	claims, err := refreshGen.Validate(refreshToken)
	if err != nil {
		return nil, err
	}

	// Verify it's a refresh token
	if claims.Role != "refresh" {
		return nil, ErrInvalidRefreshToken
	}

	// Generate new token pair (this will update the active session)
	return g.GenerateTokenPair(ctx, jwt.GenerateInput{
		UserID:    claims.UserID,
		StaffID:   claims.StaffID,
		CompanyID: claims.CompanyID,
		Slug:      claims.Slug,
		Email:     claims.Email,
		Role:      claims.Role,
	})
}

// TempTokenOutput represents the result of generating a temporary token
type TempTokenOutput struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// GenerateTempToken creates a short-lived token for company selection
func (g *Guardian) GenerateTempToken(userID, email string) (*TempTokenOutput, error) {
	// Temp token expires in 5 minutes (just for company selection)
	tempExpiration := 5 * time.Minute

	output, err := g.jwtGenerator.GenerateTempToken(userID, email, tempExpiration)
	if err != nil {
		return nil, err
	}

	return &TempTokenOutput{
		Token:     output.Token,
		ExpiresAt: output.ExpiresAt,
	}, nil
}

// ValidateTempToken validates a temporary token for company selection
func (g *Guardian) ValidateTempToken(token string) (*jwt.TempClaims, error) {
	return g.jwtGenerator.ValidateTempToken(token)
}

// Logout invalidates a specific session by JTI (current device only)
func (g *Guardian) Logout(ctx context.Context, jti string) error {
	if g.sessionStore == nil {
		return nil
	}
	return g.sessionStore.InvalidateSession(ctx, jti)
}

// LogoutAll invalidates all sessions for a user (logout everywhere)
func (g *Guardian) LogoutAll(ctx context.Context, userID string) error {
	if g.sessionStore == nil {
		return nil
	}
	return g.sessionStore.InvalidateAll(ctx, userID)
}

// LogoutAllByCompany invalidates all sessions for every active staff member of a company.
// Called when a company is suspended due to non-payment.
func (g *Guardian) LogoutAllByCompany(ctx context.Context, companyID string) error {
	if g.sessionStore == nil {
		return nil
	}
	return g.sessionStore.InvalidateAllByCompany(ctx, companyID)
}

// LogoutByToken extracts JTI from token and invalidates only that session
func (g *Guardian) LogoutByToken(ctx context.Context, accessToken string) error {
	claims, err := g.jwtGenerator.Validate(accessToken)
	if err != nil {
		return err
	}
	return g.Logout(ctx, claims.ID) // claims.ID is the JTI
}

// Cleanup removes expired sessions
func (g *Guardian) Cleanup(ctx context.Context) error {
	if g.sessionStore == nil {
		return nil
	}
	return g.sessionStore.Cleanup(ctx)
}
