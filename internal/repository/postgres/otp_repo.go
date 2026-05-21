package postgres

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

type otpTokenModel struct {
	ID        string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    string     `gorm:"not null;index"`
	Purpose   string     `gorm:"not null"`
	CodeHash  string     `gorm:"not null"`
	ExpiresAt time.Time  `gorm:"not null"`
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (otpTokenModel) TableName() string { return "otp_tokens" }

type OTPRepo struct{ db *gorm.DB }

func NewOTPRepo(db *gorm.DB) *OTPRepo { return &OTPRepo{db: db} }

func (r *OTPRepo) Create(token *domain.OTPToken) error {
	m := &otpTokenModel{
		UserID:    token.UserID,
		Purpose:   token.Purpose,
		CodeHash:  token.CodeHash,
		ExpiresAt: token.ExpiresAt,
	}
	if err := r.db.Create(m).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	token.ID = m.ID
	token.CreatedAt = m.CreatedAt
	return nil
}

func (r *OTPRepo) FindActive(userID, purpose string) (*domain.OTPToken, error) {
	var m otpTokenModel
	err := r.db.Where(
		"user_id = ? AND purpose = ? AND used_at IS NULL AND expires_at > NOW()",
		userID, purpose,
	).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, apperror.Internal(err.Error())
	}
	return &domain.OTPToken{
		ID:        m.ID,
		UserID:    m.UserID,
		Purpose:   m.Purpose,
		CodeHash:  m.CodeHash,
		ExpiresAt: m.ExpiresAt,
		UsedAt:    m.UsedAt,
		CreatedAt: m.CreatedAt,
	}, nil
}

func (r *OTPRepo) MarkUsed(id string) error {
	now := time.Now()
	if err := r.db.Model(&otpTokenModel{}).Where("id = ?", id).Update("used_at", now).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	return nil
}

func (r *OTPRepo) DeleteByUserAndPurpose(userID, purpose string) error {
	if err := r.db.Delete(&otpTokenModel{}, "user_id = ? AND purpose = ?", userID, purpose).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	return nil
}
