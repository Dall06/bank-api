package domain

import (
	"time"

	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	bun.BaseModel `bun:"table:users"`

	ID           string    `json:"id" bun:"id,pk,type:uuid"`
	Email        string    `json:"email" bun:"email,notnull"`
	PasswordHash string    `json:"-" bun:"password_hash,notnull"`
	Name         string    `json:"name" bun:"name,notnull"`
	CreatedAt    time.Time `json:"created_at" bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt    time.Time `json:"updated_at" bun:"updated_at,notnull,default:current_timestamp"`
}

func NewUser(id, email, password, name string) (*User, error) {
	hashed, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	return &User{
		ID:           id,
		Email:        email,
		PasswordHash: hashed,
		Name:         name,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}
