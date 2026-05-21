package postgres

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

type sessionModel struct {
	ID           string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID       string    `gorm:"not null;index"`
	RefreshToken string    `gorm:"uniqueIndex;not null"`
	UserAgent    string
	IPAddress    string
	LastActiveAt time.Time `gorm:"not null;default:now()"`
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
	return toSessionDomain(&m), nil
}

func (r *SessionRepo) ListByUserID(userID string) ([]*domain.Session, error) {
	var models []sessionModel
	if err := r.db.Where("user_id = ?", userID).Order("last_active_at desc").Find(&models).Error; err != nil {
		return nil, apperror.Internal(err.Error())
	}
	sessions := make([]*domain.Session, len(models))
	for i := range models {
		sessions[i] = toSessionDomain(&models[i])
	}
	return sessions, nil
}

func (r *SessionRepo) UpdateLastActive(id string, t time.Time) error {
	if err := r.db.Model(&sessionModel{}).Where("id = ?", id).Update("last_active_at", t).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	return nil
}

func toSessionDomain(m *sessionModel) *domain.Session {
	return &domain.Session{
		ID:           m.ID,
		UserID:       m.UserID,
		RefreshToken: m.RefreshToken,
		UserAgent:    m.UserAgent,
		IPAddress:    m.IPAddress,
		LastActiveAt: m.LastActiveAt,
		ExpiresAt:    m.ExpiresAt,
		CreatedAt:    m.CreatedAt,
	}
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
