package ports

import (
	"context"
	"testing"
	"bank-api/srv/users/domain"
	"github.com/stretchr/testify/assert"
	"github.com/labstack/echo/v4"
)

func TestMocks(t *testing.T) {
	ctx := context.Background()

	emptyRepo := &MockUserRepository{}
	emptyRepo.Create(ctx, nil)
	emptyRepo.GetByID(ctx, "")
	emptyRepo.GetByEmail(ctx, "")

	repo := &MockUserRepository{
		CreateFunc: func(ctx context.Context, user *domain.User) (*domain.User, error) { return nil, nil },
		GetByIDFunc: func(ctx context.Context, id string) (*domain.User, error) { return nil, nil },
		GetByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) { return nil, nil },
	}
	u, err := repo.Create(ctx, nil)
	assert.Nil(t, u)
	assert.NoError(t, err)
	u1, err1 := repo.GetByID(ctx, "")
	assert.Nil(t, u1)
	assert.NoError(t, err1)
	u2, err2 := repo.GetByEmail(ctx, "")
	assert.Nil(t, u2)
	assert.NoError(t, err2)

	emptyUsecase := &MockUserUsecase{}
	emptyUsecase.Signup(ctx, domain.SignupRequest{})
	emptyUsecase.Login(ctx, domain.LoginRequest{})
	emptyUsecase.GetByID(ctx, "")

	usecase := &MockUserUsecase{
		SignupFunc: func(ctx context.Context, req domain.SignupRequest) (*domain.AuthResponse, error) { return nil, nil },
		LoginFunc: func(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) { return nil, nil },
		GetByIDFunc: func(ctx context.Context, id string) (*domain.User, error) { return nil, nil },
	}
	resp, err := usecase.Signup(ctx, domain.SignupRequest{})
	assert.Nil(t, resp)
	assert.NoError(t, err)
	resp2, err := usecase.Login(ctx, domain.LoginRequest{})
	assert.Nil(t, resp2)
	assert.NoError(t, err)
	resp3, err := usecase.GetByID(ctx, "")
	assert.Nil(t, resp3)
	assert.NoError(t, err)

	handler := &MockUserHandler{
		SignupFunc: func(c echo.Context) error { return nil },
		LoginFunc: func(c echo.Context) error { return nil },
		GetMeFunc: func(c echo.Context) error { return nil },
		GetInternalFunc: func(c echo.Context) error { return nil },
	}
	assert.NoError(t, handler.Signup(nil))
	assert.NoError(t, handler.Login(nil))
	assert.NoError(t, handler.GetMe(nil))
	assert.NoError(t, handler.GetInternal(nil))
}
