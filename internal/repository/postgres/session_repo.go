package postgres

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

type sessionModel struct {
	ID           string `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID       string `gorm:"not null;index"`
	RefreshToken string `gorm:"uniqueIndex;not null"`
	UserAgent    string
	IPAddress    string
	ExpiresAt    time.Time `gorm:"not null"`
	CreatedAt    time.Time
}

func (sessionModel) TableName() string { return "sessions" }

type SessionRepo struct{ db *gorm.DB }

func NewSessionRepo(db *gorm.DB) *SessionRepo { return &SessionRepo{db: db} }

func (r *SessionRepo) Create(s *domain.Session) error {
	m := &sessionModel{
		UserID:       s.UserID,
		RefreshToken: s.RefreshToken,
		UserAgent:    s.UserAgent,
		IPAddress:    s.IPAddress,
		ExpiresAt:    s.ExpiresAt,
	}
	if err := r.db.Create(m).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	s.ID = m.ID
	s.CreatedAt = m.CreatedAt
	return nil
}

func (r *SessionRepo) FindByRefreshToken(token string) (*domain.Session, error) {
	var m sessionModel
	if err := r.db.First(&m, "refresh_token = ?", token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("session not found")
		}
		return nil, apperror.Internal(err.Error())
	}
	return &domain.Session{
		ID:           m.ID,
		UserID:       m.UserID,
		RefreshToken: m.RefreshToken,
		UserAgent:    m.UserAgent,
		IPAddress:    m.IPAddress,
		ExpiresAt:    m.ExpiresAt,
		CreatedAt:    m.CreatedAt,
	}, nil
}

func (r *SessionRepo) DeleteByID(id string) error {
	if err := r.db.Delete(&sessionModel{}, "id = ?", id).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	return nil
}

func (r *SessionRepo) DeleteAllByUserID(userID string) error {
	if err := r.db.Delete(&sessionModel{}, "user_id = ?", userID).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	return nil
}
