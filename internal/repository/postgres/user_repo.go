package postgres

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

type userModel struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Email        string `gorm:"uniqueIndex;not null"`
	Name         string `gorm:"not null"`
	AvatarURL    string
	PasswordHash string
	Provider     string `gorm:"not null;default:local"`
	ProviderID   string
	TOTPSecret   string
	TOTPEnabled  bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (userModel) TableName() string { return "users" }

func toUserDomain(m *userModel) *domain.User {
	return &domain.User{
		ID:           m.ID,
		Email:        m.Email,
		Name:         m.Name,
		AvatarURL:    m.AvatarURL,
		PasswordHash: m.PasswordHash,
		Provider:     domain.AuthProvider(m.Provider),
		ProviderID:   m.ProviderID,
		TOTPSecret:   m.TOTPSecret,
		TOTPEnabled:  m.TOTPEnabled,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

type UserRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) FindByID(id string) (*domain.User, error) {
	var m userModel
	if err := r.db.First(&m, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("user not found")
		}
		return nil, apperror.Internal(err.Error())
	}
	return toUserDomain(&m), nil
}

func (r *UserRepo) FindByEmail(email string) (*domain.User, error) {
	var m userModel
	if err := r.db.First(&m, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("user not found")
		}
		return nil, apperror.Internal(err.Error())
	}
	return toUserDomain(&m), nil
}

func (r *UserRepo) FindByProviderID(provider domain.AuthProvider, providerID string) (*domain.User, error) {
	var m userModel
	if err := r.db.First(&m, "provider = ? AND provider_id = ?", string(provider), providerID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("user not found")
		}
		return nil, apperror.Internal(err.Error())
	}
	return toUserDomain(&m), nil
}

func (r *UserRepo) Create(user *domain.User) error {
	m := &userModel{
		Email:        user.Email,
		Name:         user.Name,
		AvatarURL:    user.AvatarURL,
		PasswordHash: user.PasswordHash,
		Provider:     string(user.Provider),
		ProviderID:   user.ProviderID,
	}
	if err := r.db.Create(m).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	user.ID = m.ID
	user.CreatedAt = m.CreatedAt
	user.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *UserRepo) Delete(id string) error {
	if err := r.db.Delete(&userModel{}, "id = ?", id).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	return nil
}

func (r *UserRepo) Update(user *domain.User) error {
	if err := r.db.Model(&userModel{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
		"name":          user.Name,
		"avatar_url":    user.AvatarURL,
		"password_hash": user.PasswordHash,
		"totp_secret":   user.TOTPSecret,
		"totp_enabled":  user.TOTPEnabled,
		"updated_at":    time.Now(),
	}).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	return nil
}
