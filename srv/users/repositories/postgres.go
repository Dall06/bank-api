package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"bank-api/pkg/errs"
	"bank-api/srv/users/domain"
	"bank-api/srv/users/ports"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type UserRepository struct {
	db bun.IDB
}

func NewUserRepository(db bun.IDB) ports.UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}

	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	_, err := r.db.NewInsert().
		Model(user).
		Exec(ctx)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	err := r.db.NewSelect().
		Model(&user).
		Where("id = ?", id).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.NotFoundError("user not found")
		}
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.NewSelect().
		Model(&user).
		Where("email = ?", email).
		Scan(ctx)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errs.NotFoundError("user not found")
		}
		return nil, err
	}

	return &user, nil
}
