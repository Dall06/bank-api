package ports

import (
	"context"

	"bank-api/srv/users/domain"

	"github.com/labstack/echo/v4"
)

type MockUserRepository struct {
	CreateFunc     func(ctx context.Context, user *domain.User) (*domain.User, error)
	GetByIDFunc    func(ctx context.Context, id string) (*domain.User, error)
	GetByEmailFunc func(ctx context.Context, email string) (*domain.User, error)
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, user)
	}
	return user, nil
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(ctx, email)
	}
	return nil, nil
}

type MockUserUsecase struct {
	SignupFunc  func(ctx context.Context, req domain.SignupRequest) (*domain.AuthResponse, error)
	LoginFunc   func(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error)
	GetByIDFunc func(ctx context.Context, id string) (*domain.User, error)
}

func (m *MockUserUsecase) Signup(ctx context.Context, req domain.SignupRequest) (*domain.AuthResponse, error) {
	if m.SignupFunc != nil {
		return m.SignupFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockUserUsecase) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	if m.LoginFunc != nil {
		return m.LoginFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockUserUsecase) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

type MockUserHandler struct {
	SignupFunc func(c echo.Context) error
	LoginFunc  func(c echo.Context) error
	GetMeFunc  func(c echo.Context) error
	GetInternalFunc func(c echo.Context) error
}

func (m *MockUserHandler) Signup(c echo.Context) error {
	if m.SignupFunc != nil {
		return m.SignupFunc(c)
	}
	return nil
}

func (m *MockUserHandler) Login(c echo.Context) error {
	if m.LoginFunc != nil {
		return m.LoginFunc(c)
	}
	return nil
}

func (m *MockUserHandler) GetMe(c echo.Context) error {
	if m.GetMeFunc != nil {
		return m.GetMeFunc(c)
	}
	return nil
}

func (m *MockUserHandler) GetInternal(c echo.Context) error {
	if m.GetInternalFunc != nil {
		return m.GetInternalFunc(c)
	}
	return nil
}
