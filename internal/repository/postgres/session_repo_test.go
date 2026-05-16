package postgres

import (
	"testing"

	"github.com/joakim/fintrack-api/internal/domain"
)

func TestSessionRepo_ImplementsInterface(t *testing.T) {
	var _ domain.SessionRepository = (*SessionRepo)(nil)
}
