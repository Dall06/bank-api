package usecases

import (
	"context"
	"testing"
	"time"

	"bank-api/pkg/errs"
	"bank-api/pkg/jwt"
	"bank-api/srv/users/domain"
	"bank-api/srv/users/ports"

	"github.com/stretchr/testify/assert"
)

func TestUserUsecase_Signup(t *testing.T) {
	jwtGen := jwt.NewGenerator(jwt.Config{
		Secret:     "test-secret",
		Expiration: 1 * time.Hour,
	})

	tests := []struct {
		name        string
		email       string
		password    string
		nameVal     string
		expectedErr error
	}{
		{
			name:        "Signup exitoso",
			email:       "juan@example.com",
			password:    "password123",
			nameVal:     "Juan Perez",
			expectedErr: nil,
		},
		{
			name:        "Signup fallido - usuario duplicado",
			email:       "juan@example.com",
			password:    "otrapassword",
			nameVal:     "Juan Perez",
			expectedErr: errs.ValueError("el correo ya está registrado"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &ports.MockUserRepository{
				CreateFunc: func(ctx context.Context, user *domain.User) (*domain.User, error) {
					return user, nil
				},
				GetByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
					if tt.expectedErr != nil && tt.expectedErr.Error() == "el correo ya está registrado" {
						return &domain.User{}, nil // Simula que el usuario ya existe
					}
					return nil, errs.NotFoundError("not found")
				},
			}
			uc := NewSignupUseCase(repo, jwtGen)
			ctx := context.Background()

			res, err := uc.Signup(ctx, domain.SignupRequest{
				Email:    tt.email,
				Password: tt.password,
				Name:     tt.nameVal,
			})

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
				assert.Nil(t, res)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, res)
			assert.NotEmpty(t, res.Token)
			assert.Equal(t, tt.email, res.User.Email)
		})
	}
}
