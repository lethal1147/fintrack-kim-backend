package domain

import "time"

type Session struct {
	ID           string
	UserID       string
	RefreshToken string
	UserAgent    string
	IPAddress    string
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

type SessionRepository interface {
	Create(session *Session) error
	FindByRefreshToken(token string) (*Session, error)
	DeleteByID(id string) error
	DeleteAllByUserID(userID string) error
}
