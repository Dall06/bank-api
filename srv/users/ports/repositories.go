package ports

import (
	"bank-api/srv/users/domain"
	"context"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}
