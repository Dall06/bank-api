package usecases

import (
	"context"

	"bank-api/pkg/errs"
	"bank-api/pkg/jwt"
	"bank-api/srv/users/domain"
	"bank-api/srv/users/ports"
)

// GetUseCase implementa la lógica de negocio
type GetUseCase struct {
	repo   ports.UserRepository
	jwtGen *jwt.Generator
}

// NewUserUseCase inicializa los casos de uso
func NewUserUseCase(repo ports.UserRepository, jwtGen *jwt.Generator) ports.GetUseCase {
	return &GetUseCase{
		repo:   repo,
		jwtGen: jwtGen,
	}
}

func (uc *GetUseCase) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if id == "" {
		return nil, errs.ValueError("ID requerido")
	}

	user, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, errs.NotFoundError("usuario no encontrado")
	}

	return user, nil
}
