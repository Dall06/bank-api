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

func TestUserUsecase_Login(t *testing.T) {
	jwtGen := jwt.NewGenerator(jwt.Config{
		Secret:     "test-secret",
		Expiration: 1 * time.Hour,
	})

	tests := []struct {
		name          string
		email         string
		password      string
		expectedErr   error
		setupRepoUser *domain.User
	}{
		{
			name:        "Login exitoso",
			email:       "pedro@example.com",
			password:    "pedropass",
			expectedErr: nil,
			setupRepoUser: func() *domain.User {
				u, _ := domain.NewUser("1", "pedro@example.com", "pedropass", "Pedro")
				return u
			}(),
		},
		{
			name:        "Login fallido - contraseña incorrecta",
			email:       "pedro@example.com",
			password:    "claveincorrecta",
			expectedErr: errs.UnauthorizedError("credenciales inválidas"),
			setupRepoUser: func() *domain.User {
				u, _ := domain.NewUser("1", "pedro@example.com", "pedropass", "Pedro")
				return u
			}(),
		},
		{
			name:        "Login fallido - usuario inexistente",
			email:       "inexistente@example.com",
			password:    "algunaclave",
			expectedErr: errs.UnauthorizedError("credenciales inválidas"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &ports.MockUserRepository{
				GetByEmailFunc: func(ctx context.Context, email string) (*domain.User, error) {
					if tt.setupRepoUser != nil && email == tt.setupRepoUser.Email {
						return tt.setupRepoUser, nil
					}
					return nil, errs.NotFoundError("not found")
				},
			}
			uc := NewLoginUseCase(repo, jwtGen)
			ctx := context.Background()

			res, err := uc.Login(ctx, domain.LoginRequest{
				Email:    tt.email,
				Password: tt.password,
			})

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, res)
			assert.NotEmpty(t, res.Token)
			assert.Equal(t, tt.email, res.User.Email)
		})
	}
}
