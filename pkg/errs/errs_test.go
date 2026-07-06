package errs

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestCustomError(t *testing.T) {
	tests := []struct {
		name     string
		sentinel error
		msg      string
		wantMsg  string
	}{
		{
			name:     "custom error properties",
			sentinel: ErrNotFound,
			msg:      "custom message",
			wantMsg:  "custom message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &customError{sentinel: tt.sentinel, msg: tt.msg}
			if err.Error() != tt.wantMsg {
				t.Errorf("Error() = %v, want %v", err.Error(), tt.wantMsg)
			}
			if !errors.Is(err.Unwrap(), tt.sentinel) {
				t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), tt.sentinel)
			}
		})
	}
}

func TestErrorFactories(t *testing.T) {
	tests := []struct {
		name     string
		factory  func(string, ...any) error
		sentinel error
		msg      string
		args     []any
		wantMsg  string
	}{
		{"NotFoundError", NotFoundError, ErrNotFound, "not found %s", []any{"user"}, "not found user"},
		{"InternalError", InternalError, ErrInternal, "internal %s", []any{"error"}, "internal error"},
		{"ValueError", ValueError, ErrValue, "value %d", []any{1}, "value 1"},
		{"UnauthorizedError", UnauthorizedError, ErrUnauthorized, "unauthorized %s", []any{"action"}, "unauthorized action"},
		{"ForbiddenError", ForbiddenError, ErrForbidden, "forbidden %s", []any{"access"}, "forbidden access"},
		{"NotValidError", NotValidError, ErrNotValid, "not valid %s", []any{"input"}, "not valid input"},
		{"ConflictError", ConflictError, ErrConflict, "conflict %s", []any{"data"}, "conflict data"},
		{"ServiceUnavailableError", ServiceUnavailableError, ErrServiceUnavail, "unavailable %s", []any{"service"}, "unavailable service"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.factory(tt.msg, tt.args...)
			if !errors.Is(err, tt.sentinel) {
				t.Errorf("expected error to wrap %v, got %v", tt.sentinel, err)
			}
			if err.Error() != tt.wantMsg {
				t.Errorf("expected message %q, got %q", tt.wantMsg, err.Error())
			}
		})
	}
}

func TestRestCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"ErrNotFound", ErrNotFound, 404},
		{"ErrInternal", ErrInternal, 500},
		{"ErrValue", ErrValue, 400},
		{"ErrUnauthorized", ErrUnauthorized, 401},
		{"ErrForbidden", ErrForbidden, 403},
		{"ErrNotValid", ErrNotValid, 424},
		{"ErrConflict", ErrConflict, 409},
		{"ErrPlanLimitExceeded", ErrPlanLimitExceeded, 402},
		{"ErrServiceUnavail", ErrServiceUnavail, 503},
		{"Wrapped ErrNotFound", wrappedErr(ErrNotFound), 404},
		{"Unknown Error", errors.New("unknown"), 500},
		{"Nil Error", nil, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RestCode(tt.err); got != tt.want {
				t.Errorf("RestCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

// replacing fmtErrorWrap with a proper wrapper
func wrappedErr(err error) error {
	return &customError{sentinel: err, msg: "wrapped"}
}

func TestRestCodeWrapped(t *testing.T) {
	if got := RestCode(wrappedErr(ErrNotFound)); got != 404 {
		t.Errorf("RestCode() = %v, want 404", got)
	}
}

func TestErrorHandler(t *testing.T) {
	e := echo.New()

	tests := []struct {
		name       string
		err        error
		committed  bool
		wantStatus int
		wantBody   string
	}{
		{
			name:       "404 error",
			err:        NotFoundError("user not found"),
			wantStatus: 404,
			wantBody:   `{"error":"user not found"}` + "\n",
		},
		{
			name:       "500 internal error replaces message",
			err:        InternalError("db connection failed"),
			wantStatus: 500,
			wantBody:   `{"error":"internal error"}` + "\n",
		},
		{
			name:       "already committed",
			err:        NotFoundError("not found"),
			committed:  true,
			wantStatus: 200, // Status doesn't change since we return early
			wantBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			if tt.committed {
				c.Response().WriteHeader(http.StatusOK)
				c.Response().Committed = true
			}

			ErrorHandler(tt.err, c)

			if !tt.committed {
				if rec.Code != tt.wantStatus {
					t.Errorf("ErrorHandler status = %v, want %v", rec.Code, tt.wantStatus)
				}
				if rec.Body.String() != tt.wantBody {
					t.Errorf("ErrorHandler body = %v, want %v", rec.Body.String(), tt.wantBody)
				}
			}
		})
	}
}
