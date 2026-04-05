package models

import (
	"time"

	"github.com/matthewhartstonge/argon2"
	"github.com/rs/xid"
)

// Role represents user roles for auth
type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// User represents a store user.
type User struct {
	BaseModel    `bson:",inline"`
	Email        string  `json:"email" bson:"email" validate:"required,email,lowercase"`
	Name         string  `json:"name" bson:"name" validate:"required,min=2,max=100"`
	PasswordHash string  `json:"-" bson:"password_hash"`
	Address      Address `json:"address" bson:"address"`
	Active       bool    `json:"active" bson:"active"`
	Role         Role    `json:"role" bson:"role"`
}

// UserParams contains parameters for creating a new user.
type UserParams struct {
	Email    string
	Name     string
	Password string
	Role     Role
	Address  Address
}

// NewUser creates a new user with generated ID, hashed password, and timestamps.
func NewUser(p UserParams) (*User, error) {
	argon := argon2.DefaultConfig()
	hash, err := argon.HashEncoded([]byte(p.Password))
	if err != nil {
		return nil, err
	}

	role := p.Role
	if role == "" {
		role = RoleUser
	}

	return &User{
		BaseModel: BaseModel{
			ID:        xid.New().String(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		Email:        p.Email,
		Name:         p.Name,
		PasswordHash: string(hash),
		Address:      p.Address,
		Active:       true,
		Role:         role,
	}, nil
}

// VerifyPassword checks if the provided password matches the stored hash.
func (u *User) VerifyPassword(password string) bool {
	ok, _ := argon2.VerifyEncoded([]byte(password), []byte(u.PasswordHash))
	return ok
}

// Update updates the user's profile information.
func (u *User) Update(name string, address Address) {
	if name != "" {
		u.Name = name
	}
	u.Address = address
	u.UpdatedAt = time.Now()
}

// Deactivate deactivates the user account.
func (u *User) Deactivate() {
	u.Active = false
	u.UpdatedAt = time.Now()
}
