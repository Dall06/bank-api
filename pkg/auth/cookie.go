package auth

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	// TokenCookieName is the name of the HttpOnly cookie that stores the JWT
	TokenCookieName = "bro_token"
	// MemberTokenCookieName is the name of the HttpOnly cookie for member portal JWT
	MemberTokenCookieName = "bro_member_token"
)

// SetTokenCookie sets an HttpOnly, Secure, SameSite cookie with the JWT token.
// The cookie is scoped to Path=/ so it's sent on all requests.
func SetTokenCookie(ctx echo.Context, token string, expiresAt time.Time, isSecure bool) {
	cookie := &http.Cookie{ // #nosec G124 -- Secure is set via isSecure param
		Name:     TokenCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	}
	ctx.SetCookie(cookie)
}

// ClearTokenCookie removes the token cookie by setting it to expired.
func ClearTokenCookie(ctx echo.Context, isSecure bool) {
	cookie := &http.Cookie{ // #nosec G124 -- Secure is set via isSecure param
		Name:     TokenCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	}
	ctx.SetCookie(cookie)
}

// GetTokenFromCookie reads the JWT token from the cookie.
// Returns empty string if cookie is not present.
func GetTokenFromCookie(ctx echo.Context) string {
	cookie, err := ctx.Cookie(TokenCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// SetMemberTokenCookie sets an HttpOnly cookie with the member portal JWT.
func SetMemberTokenCookie(ctx echo.Context, token string, expiresAt time.Time, isSecure bool) {
	cookie := &http.Cookie{ // #nosec G124 -- Secure is set via isSecure param
		Name:     MemberTokenCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	}
	ctx.SetCookie(cookie)
}

// ClearMemberTokenCookie removes the member token cookie by setting it to expired.
func ClearMemberTokenCookie(ctx echo.Context, isSecure bool) {
	cookie := &http.Cookie{ // #nosec G124 -- Secure is set via isSecure param
		Name:     MemberTokenCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   isSecure,
		SameSite: http.SameSiteLaxMode,
	}
	ctx.SetCookie(cookie)
}

// GetMemberTokenFromCookie reads the member portal JWT from the cookie.
func GetMemberTokenFromCookie(ctx echo.Context) string {
	cookie, err := ctx.Cookie(MemberTokenCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
