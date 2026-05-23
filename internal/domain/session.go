package domain

import "time"

type Session struct {
	ID           string
	UserID       string
	RefreshToken string
	UserAgent    string
	IPAddress    string
	LastActiveAt time.Time
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

type SessionRepository interface {
	Create(session *Session) error
	FindByID(id string) (*Session, error)
	FindByRefreshToken(token string) (*Session, error)
	DeleteByID(id string) error
	DeleteAllByUserID(userID string) error
	ListByUserID(userID string) ([]*Session, error)
	UpdateLastActive(id string, t time.Time) error
}
