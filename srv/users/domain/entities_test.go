package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_NewUser(t *testing.T) {
	u, err := NewUser("user-1", "john@example.com", "password123", "John Doe")
	assert.NoError(t, err)
	assert.Equal(t, "user-1", u.ID)
	assert.Equal(t, "John Doe", u.Name)
	assert.Equal(t, "john@example.com", u.Email)
	assert.NotEmpty(t, u.PasswordHash)
	assert.False(t, u.CreatedAt.IsZero())
	assert.False(t, u.UpdatedAt.IsZero())
}

func TestUser_PasswordHashing(t *testing.T) {
	u, err := NewUser("user-2", "jane@example.com", "securepass", "Jane Doe")
	assert.NoError(t, err)
	assert.NotEqual(t, "securepass", u.PasswordHash)
	assert.NotEmpty(t, u.PasswordHash)

	// Check correct password
	assert.True(t, u.CheckPassword("securepass"))

	// Check incorrect password
	assert.False(t, u.CheckPassword("wrongpass"))
}
