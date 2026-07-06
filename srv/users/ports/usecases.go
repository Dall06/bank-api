package ports

import (
	"bank-api/srv/users/domain"
	"context"
)

type SignupUseCase interface {
	Signup(ctx context.Context, req domain.SignupRequest) (*domain.AuthResponse, error)
}

type GetUseCase interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

type LoginUseCase interface {
	Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error)
}
