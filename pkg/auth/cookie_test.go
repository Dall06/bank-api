package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestCookieTokens(t *testing.T) {
	e := echo.New()

	tests := []struct {
		name       string
		action     func(ctx echo.Context)
		cookieName string
		wantValue  string
		wantMaxAge int
	}{
		{
			name: "SetTokenCookie",
			action: func(ctx echo.Context) {
				SetTokenCookie(ctx, "my-token", time.Now().Add(time.Hour), true)
			},
			cookieName: TokenCookieName,
			wantValue:  "my-token",
			wantMaxAge: 0,
		},
		{
			name: "ClearTokenCookie",
			action: func(ctx echo.Context) {
				ClearTokenCookie(ctx, true)
			},
			cookieName: TokenCookieName,
			wantValue:  "",
			wantMaxAge: -1,
		},
		{
			name: "SetMemberTokenCookie",
			action: func(ctx echo.Context) {
				SetMemberTokenCookie(ctx, "member-token", time.Now().Add(time.Hour), true)
			},
			cookieName: MemberTokenCookieName,
			wantValue:  "member-token",
			wantMaxAge: 0,
		},
		{
			name: "ClearMemberTokenCookie",
			action: func(ctx echo.Context) {
				ClearMemberTokenCookie(ctx, true)
			},
			cookieName: MemberTokenCookieName,
			wantValue:  "",
			wantMaxAge: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)

			tt.action(ctx)

			var found *http.Cookie
			for _, c := range rec.Result().Cookies() {
				if c.Name == tt.cookieName {
					found = c
					break
				}
			}

			if found == nil {
				t.Fatalf("cookie %s not set", tt.cookieName)
			}
			if found.Value != tt.wantValue {
				t.Errorf("got value %q, want %q", found.Value, tt.wantValue)
			}
			if found.MaxAge != tt.wantMaxAge {
				t.Errorf("got maxage %d, want %d", found.MaxAge, tt.wantMaxAge)
			}
		})
	}
}

func TestGetTokens(t *testing.T) {
	e := echo.New()

	tests := []struct {
		name       string
		setup      func(req *http.Request)
		action     func(ctx echo.Context) string
		wantResult string
	}{
		{
			name: "GetTokenFromCookie success",
			setup: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: TokenCookieName, Value: "token-123"})
			},
			action:     GetTokenFromCookie,
			wantResult: "token-123",
		},
		{
			name:       "GetTokenFromCookie not found",
			setup:      func(req *http.Request) {},
			action:     GetTokenFromCookie,
			wantResult: "",
		},
		{
			name: "GetMemberTokenFromCookie success",
			setup: func(req *http.Request) {
				req.AddCookie(&http.Cookie{Name: MemberTokenCookieName, Value: "mem-token-123"})
			},
			action:     GetMemberTokenFromCookie,
			wantResult: "mem-token-123",
		},
		{
			name:       "GetMemberTokenFromCookie not found",
			setup:      func(req *http.Request) {},
			action:     GetMemberTokenFromCookie,
			wantResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setup(req)
			rec := httptest.NewRecorder()
			ctx := e.NewContext(req, rec)

			got := tt.action(ctx)
			if got != tt.wantResult {
				t.Errorf("got %q, want %q", got, tt.wantResult)
			}
		})
	}
}
