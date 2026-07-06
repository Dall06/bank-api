package usecases

import (
	"context"
	"testing"

	"bank-api/pkg/errs"
	"bank-api/srv/users/domain"
	"bank-api/srv/users/ports"

	"github.com/stretchr/testify/assert"
)

func TestUserUsecase_GetByID(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		expectedErr error
		setupUser   *domain.User
	}{
		{
			name:        "Obtener por ID exitoso",
			id:          "123",
			expectedErr: nil,
			setupUser: &domain.User{
				ID:    "123",
				Email: "test@test.com",
				Name:  "Test",
			},
		},
		{
			name:        "Falla ID vacio",
			id:          "",
			expectedErr: errs.ValueError("ID requerido"),
		},
		{
			name:        "Falla no encontrado",
			id:          "not-found",
			expectedErr: errs.NotFoundError("usuario no encontrado"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &ports.MockUserRepository{
				GetByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
					if tt.setupUser != nil && tt.setupUser.ID == id {
						return tt.setupUser, nil
					}
					return nil, errs.NotFoundError("not found")
				},
			}
			uc := NewUserUseCase(repo, nil)
			ctx := context.Background()

			res, err := uc.GetByID(ctx, tt.id)

			if tt.expectedErr != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErr.Error())
				assert.Nil(t, res)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, res)
			assert.Equal(t, tt.setupUser.ID, res.ID)
		})
	}
}
