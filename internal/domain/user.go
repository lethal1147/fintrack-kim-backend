package domain

import "time"

type AuthProvider string

const (
	ProviderLocal    AuthProvider = "local"
	ProviderGoogle   AuthProvider = "google"
	ProviderFacebook AuthProvider = "facebook"
)

type User struct {
	ID           string
	Email        string
	Name         string
	AvatarURL    string
	PasswordHash string
	Provider     AuthProvider
	ProviderID   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserRepository interface {
	FindByID(id string) (*User, error)
	FindByEmail(email string) (*User, error)
	FindByProviderID(provider AuthProvider, providerID string) (*User, error)
	Create(user *User) error
	Update(user *User) error
}
