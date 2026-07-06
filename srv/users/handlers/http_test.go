package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bank-api/pkg/errs"
	"bank-api/srv/users/domain"
	"bank-api/srv/users/ports"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestUserHandler_Signup(t *testing.T) {
	tests := []struct {
		name         string
		reqBody      interface{}
		mockSignup   func(ctx context.Context, req domain.SignupRequest) (*domain.AuthResponse, error)
		wantStatus   int
		wantErrBody  string
	}{
		{
			name: "éxito",
			reqBody: domain.SignupRequest{
				Email:    "test@test.com",
				Password: "password",
				Name:     "Test",
			},
			mockSignup: func(ctx context.Context, req domain.SignupRequest) (*domain.AuthResponse, error) {
				u, _ := domain.NewUser("1", "test@test.com", "hash", "Test")
				return &domain.AuthResponse{User: u, Token: "t"}, nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "body inválido",
			reqBody: "not-json",
			mockSignup: nil,
			wantStatus: http.StatusBadRequest,
			wantErrBody: "cuerpo de petición inválido",
		},
		{
			name: "error del usecase",
			reqBody: domain.SignupRequest{
				Email: "test@test.com",
			},
			mockSignup: func(ctx context.Context, req domain.SignupRequest) (*domain.AuthResponse, error) {
				return nil, errs.ValueError("el correo ya está registrado")
			},
			wantStatus: http.StatusBadRequest,
			wantErrBody: "el correo ya está registrado",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUC := &ports.MockUserUsecase{
				SignupFunc: tt.mockSignup,
			}
			h := NewUserHandler(mockUC, mockUC, mockUC)

			err := h.Signup(c)
			
			if tt.wantErrBody != "" {
				if err != nil {
					assert.Contains(t, err.Error(), tt.wantErrBody)
				} else {
					// si el error subyacente lo manejó echo y no retornó err
					assert.Equal(t, tt.wantStatus, rec.Code)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestUserHandler_Login(t *testing.T) {
	tests := []struct {
		name         string
		reqBody      interface{}
		mockLogin    func(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error)
		wantStatus   int
		wantErrBody  string
	}{
		{
			name: "éxito",
			reqBody: domain.LoginRequest{
				Email:    "test@test.com",
				Password: "password",
			},
			mockLogin: func(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
				u, _ := domain.NewUser("1", "test@test.com", "hash", "Test")
				return &domain.AuthResponse{User: u, Token: "t"}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "credenciales inválidas",
			reqBody: domain.LoginRequest{
				Email: "test@test.com",
			},
			mockLogin: func(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
				return nil, errs.UnauthorizedError("credenciales inválidas")
			},
			wantStatus: http.StatusUnauthorized,
			wantErrBody: "credenciales inválidas",
		},
		{
			name: "body inválido",
			reqBody: "not-json",
			mockLogin: nil,
			wantStatus: http.StatusBadRequest,
			wantErrBody: "cuerpo de petición inválido",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			body, _ := json.Marshal(tt.reqBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			mockUC := &ports.MockUserUsecase{
				LoginFunc: tt.mockLogin,
			}
			h := NewUserHandler(mockUC, mockUC, mockUC)

			err := h.Login(c)
			
			if tt.wantErrBody != "" {
				if err != nil {
					assert.Contains(t, err.Error(), tt.wantErrBody)
				} else {
					assert.Equal(t, tt.wantStatus, rec.Code)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantStatus, rec.Code)
			}
		})
	}
}

func TestUserHandler_GetMe(t *testing.T) {
	tests := []struct {
		name        string
		paramID     string
		ctxUserID   interface{}
		mockGet     func(ctx context.Context, id string) (*domain.User, error)
		wantStatus  int
		wantErrBody string
	}{
		{
			name:      "éxito con param id",
			paramID:   "1",
			ctxUserID: nil,
			mockGet: func(ctx context.Context, id string) (*domain.User, error) {
				u, _ := domain.NewUser("1", "test@test.com", "hash", "Test")
				return u, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:      "éxito con ctx user_id",
			paramID:   "",
			ctxUserID: "2",
			mockGet: func(ctx context.Context, id string) (*domain.User, error) {
				u, _ := domain.NewUser("2", "test2@test.com", "hash", "Test 2")
				return u, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name:        "sin id param ni ctx",
			paramID:     "",
			ctxUserID:   nil,
			mockGet:     nil,
			wantStatus:  http.StatusUnauthorized,
			wantErrBody: "no se pudo identificar el usuario",
		},
		{
			name:      "error de usecase",
			paramID:   "3",
			ctxUserID: nil,
			mockGet: func(ctx context.Context, id string) (*domain.User, error) {
				return nil, errs.NotFoundError("usuario no encontrado")
			},
			wantStatus:  http.StatusNotFound,
			wantErrBody: "usuario no encontrado",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			if tt.paramID != "" {
				c.SetParamNames("id")
				c.SetParamValues(tt.paramID)
			}
			if tt.ctxUserID != nil {
				c.Set("user_id", tt.ctxUserID)
			}

			mockUC := &ports.MockUserUsecase{
				GetByIDFunc: tt.mockGet,
			}
			h := NewUserHandler(mockUC, mockUC, mockUC)

			err := h.GetMe(c)

			if tt.wantErrBody != "" {
				if err != nil {
					assert.Contains(t, err.Error(), tt.wantErrBody)
				} else {
					assert.Equal(t, tt.wantStatus, rec.Code)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantStatus, rec.Code)
			}
		})
	}
}

