package usecases

import (
	"bank-api/srv/users/ports"
	"context"
	"errors"

	"bank-api/pkg/errs"
	"bank-api/pkg/jwt"
	"bank-api/srv/users/domain"

	"github.com/google/uuid"
)

type SignupUseCase struct {
	repo   ports.UserRepository
	jwtGen *jwt.Generator
}

// NewSignupUseCase inicializa los casos de uso
func NewSignupUseCase(repo ports.UserRepository, jwtGen *jwt.Generator) ports.SignupUseCase {
	return &SignupUseCase{
		repo:   repo,
		jwtGen: jwtGen,
	}
}

func (uc *SignupUseCase) Signup(ctx context.Context, req domain.SignupRequest) (*domain.AuthResponse, error) {
	// Verificar si el usuario ya existe
	existing, err := uc.repo.GetByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, errs.ErrNotFound) {
		return nil, errs.InternalError("error al verificar el correo: %v", err)
	}
	if existing != nil {
		return nil, errs.ValueError("el correo ya está registrado")
	}

	// Crear el usuario con un nuevo UUID
	id := uuid.New().String()
	user, err := domain.NewUser(id, req.Email, req.Password, req.Name)
	if err != nil {
		return nil, errs.InternalError("error al procesar la contraseña: %v", err)
	}

	// Guardar en repositorio
	createdUser, err := uc.repo.Create(ctx, user)
	if err != nil {
		return nil, errs.InternalError("error al guardar usuario: %v", err)
	}

	// Generar JWT
	tokenOut, err := uc.jwtGen.Generate(jwt.GenerateInput{
		UserID: createdUser.ID,
		Email:  createdUser.Email,
		Role:   "merchant",
	})
	if err != nil {
		return nil, errs.InternalError("error al generar token: %v", err)
	}

	return &domain.AuthResponse{
		User:  createdUser,
		Token: tokenOut.Token,
	}, nil
}
