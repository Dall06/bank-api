package middlewares

import (
	"strings"

	"bank-api/opt/guardian"
	"bank-api/pkg/auth"
	"bank-api/pkg/errs"
	pkgjwt "bank-api/pkg/jwt"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// extractTokenString extracts JWT token string from cookie first, then Bearer header.
// Cookie takes priority because it's the secure path (HttpOnly).
func extractTokenString(ctx echo.Context) string {
	// 1. Try HttpOnly cookie first (secure path)
	if token := auth.GetTokenFromCookie(ctx); token != "" {
		return token
	}

	// 2. Fallback to Authorization: Bearer header (backward compat + API clients)
	authHeader := ctx.Request().Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return parts[1]
}

// extractAndValidateToken extracts and validates a JWT token from cookie or Authorization header
func extractAndValidateToken(ctx echo.Context, secret string) (*pkgjwt.Claims, error) {
	tokenString := extractTokenString(ctx)
	if tokenString == "" {
		return nil, errs.ErrUnauthorized
	}

	// Parse and validate token
	token, err := jwt.ParseWithClaims(tokenString, &pkgjwt.Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing algorithm to prevent algorithm confusion attacks
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})

	if err != nil || !token.Valid {
		return nil, errs.ErrUnauthorized
	}

	claims, ok := token.Claims.(*pkgjwt.Claims)
	if !ok {
		return nil, errs.ErrUnauthorized
	}

	return claims, nil
}

// setClaimsInContext stores JWT claims in the Echo context
func setClaimsInContext(ctx echo.Context, claims *pkgjwt.Claims) {
	ctx.Set("user_id", claims.UserID)
	ctx.Set("staff_id", claims.StaffID)
	ctx.Set("company_id", claims.CompanyID)
	ctx.Set("slug", claims.Slug)
	ctx.Set("email", claims.Email)
	ctx.Set("role", claims.Role)
	ctx.Set("jti", claims.ID) // JWT ID for logout
}

// NewJWTAuth creates a JWT authentication middleware
func NewJWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			claims, err := extractAndValidateToken(ctx, secret)
			if err != nil {
				return echo.NewHTTPError(401, err.Error())
			}

			setClaimsInContext(ctx, claims)
			return next(ctx)
		}
	}
}

// NewOptionalJWTAuth creates a middleware that validates JWT if present but doesn't require it
// If no token is provided, the request continues without error (anonymous access)
// If token is provided but invalid, the request continues without error
// If token is valid, claims are stored in context
func NewOptionalJWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			claims, err := extractAndValidateToken(ctx, secret)
			if err == nil && claims != nil {
				setClaimsInContext(ctx, claims)
			}
			return next(ctx)
		}
	}
}

// RequireGod creates a middleware that requires the user to have "god" role
// God role is the first staff member created with full permissions
func RequireGod() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			role := ctx.Get("role")
			if role == nil || role != "god" {
				return echo.NewHTTPError(403, "forbidden: god role required")
			}
			return next(ctx)
		}
	}
}

// RequireAdminOrGod creates a middleware that requires the user to have "admin" or "god" role
func RequireAdminOrGod() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			role := ctx.Get("role")
			if role == nil {
				return echo.NewHTTPError(403, "forbidden: admin or god role required")
			}

			roleStr, ok := role.(string)
			if !ok {
				return echo.NewHTTPError(403, "forbidden: admin or god role required")
			}

			if roleStr != "admin" && roleStr != "god" {
				return echo.NewHTTPError(403, "forbidden: admin or god role required")
			}

			return next(ctx)
		}
	}
}

// RequireGodOrOwner creates a middleware that requires the user to have "god" or "owner" role
// Used for sensitive operations like report exports that only business owners should access
func RequireGodOrOwner() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			role := ctx.Get("role")
			if role == nil {
				return echo.NewHTTPError(403, "forbidden: god or owner role required")
			}

			roleStr, ok := role.(string)
			if !ok {
				return echo.NewHTTPError(403, "forbidden: god or owner role required")
			}

			if roleStr != "god" && roleStr != "owner" {
				return echo.NewHTTPError(403, "forbidden: god or owner role required")
			}

			return next(ctx)
		}
	}
}

// NewGuardianAuth creates a JWT authentication middleware that validates sessions.
// Reads token from HttpOnly cookie first, then Bearer header (backward compat).
// Validates JWT signature AND checks if the token is still active in session store.
func NewGuardianAuth(g *guardian.Guardian) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			tokenString := extractTokenString(ctx)
			if tokenString == "" {
				return echo.NewHTTPError(401, "unauthorized")
			}

			// Validate token using Guardian (checks JWT + session store)
			claims, err := g.ValidateAccessToken(ctx.Request().Context(), tokenString)
			if err != nil {
				return echo.NewHTTPError(401, "unauthorized")
			}

			// Reject member tokens on staff routes
			if claims.Role == guardian.RoleMember {
				return echo.NewHTTPError(403, "forbidden: staff access only")
			}

			setClaimsInContext(ctx, claims)
			return next(ctx)
		}
	}
}

// CompanyStatusChecker is an interface for checking company status
// This is used to decouple the middleware from the repository implementation
type CompanyStatusChecker interface {
	IsCompanySuspended(ctx echo.Context, companyID string) (bool, error)
}

// NewCompanyStatusCheck creates a middleware that verifies the company is not suspended
// This middleware should be applied after JWT authentication middleware
// If the company is suspended, returns 403 Forbidden with a descriptive message
func NewCompanyStatusCheck(checker CompanyStatusChecker) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			companyID := ctx.Get("company_id")
			if companyID == nil {
				// No company ID in context, skip check (might be a public route)
				return next(ctx)
			}

			companyIDStr, ok := companyID.(string)
			if !ok || companyIDStr == "" {
				return next(ctx)
			}

			suspended, err := checker.IsCompanySuspended(ctx, companyIDStr)
			if err != nil {
				// Fail-closed: if we can't verify status, block access
				return ctx.JSON(503, map[string]interface{}{
					"error":   "service_unavailable",
					"message": "Unable to verify account status. Please try again.",
				})
			}

			if suspended {
				return ctx.JSON(402, map[string]interface{}{
					"error":   "subscription_expired",
					"message": "Your subscription has expired or your account has been suspended. Please renew to continue.",
					"code":    "SUBSCRIPTION_EXPIRED",
				})
			}

			return next(ctx)
		}
	}
}
