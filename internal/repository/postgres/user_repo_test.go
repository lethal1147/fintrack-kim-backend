package postgres

import (
	"testing"

	"github.com/joakim/fintrack-api/internal/domain"
)

// mockUserRepo verifies UserRepo satisfies the domain.UserRepository interface.
// Real DB integration tests live in *_integration_test.go.
func TestUserRepo_ImplementsInterface(t *testing.T) {
	var _ domain.UserRepository = (*UserRepo)(nil)
}

func TestToUserDomain(t *testing.T) {
	m := &userModel{
		ID:       "u1",
		Email:    "a@b.com",
		Name:     "Alice",
		Provider: "local",
	}
	u := toUserDomain(m)
	if u.ID != "u1" || u.Email != "a@b.com" || u.Name != "Alice" {
		t.Errorf("unexpected domain user: %+v", u)
	}
	if u.Provider != domain.ProviderLocal {
		t.Errorf("want provider local, got %s", u.Provider)
	}
}
