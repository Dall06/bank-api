package usecases

import (
	"bank-api/srv/users/ports"
	"context"

	"bank-api/pkg/errs"
	"bank-api/pkg/jwt"
	"bank-api/srv/users/domain"
)

type LoginUseCase struct {
	repo   ports.UserRepository
	jwtGen *jwt.Generator
}

// NewLoginUseCase inicializa los casos de uso
func NewLoginUseCase(repo ports.UserRepository, jwtGen *jwt.Generator) ports.LoginUseCase {
	return &LoginUseCase{
		repo:   repo,
		jwtGen: jwtGen,
	}
}

func (uc *LoginUseCase) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	user, err := uc.repo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return nil, errs.UnauthorizedError("credenciales inválidas")
	}

	if !user.CheckPassword(req.Password) {
		return nil, errs.UnauthorizedError("credenciales inválidas")
	}

	tokenOut, err := uc.jwtGen.Generate(jwt.GenerateInput{
		UserID: user.ID,
		Email:  user.Email,
		Role:   "merchant",
	})
	if err != nil {
		return nil, errs.InternalError("error al generar token: %v", err)
	}

	return &domain.AuthResponse{
		User:  user,
		Token: tokenOut.Token,
	}, nil
}
