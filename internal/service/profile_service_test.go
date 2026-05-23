package service

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
	"github.com/joakim/fintrack-api/pkg/hashutil"
)

// ── mock user repo ────────────────────────────────────────────────────────────

type mockProfileUserRepo struct {
	user        *domain.User
	findByEmail *domain.User
	updateErr   error
	deleteErr   error
	deleted     bool
}

func (m *mockProfileUserRepo) FindByID(_ string) (*domain.User, error) {
	if m.user == nil {
		return nil, apperror.NotFound("user not found")
	}
	return m.user, nil
}

func (m *mockProfileUserRepo) FindByEmail(_ string) (*domain.User, error) {
	return m.findByEmail, nil
}

func (m *mockProfileUserRepo) FindByProviderID(_ domain.AuthProvider, _ string) (*domain.User, error) {
	return nil, nil
}

func (m *mockProfileUserRepo) Create(_ *domain.User) error { return nil }

func (m *mockProfileUserRepo) Update(u *domain.User) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.user = u
	return nil
}

func (m *mockProfileUserRepo) Delete(_ string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deleted = true
	return nil
}

// ── mock uploader ─────────────────────────────────────────────────────────────

type mockUploader struct {
	url      string
	uploadErr error
}

func (m *mockUploader) Upload(_ context.Context, _, _ string, _ io.Reader) (string, error) {
	return m.url, m.uploadErr
}

// ── helpers ───────────────────────────────────────────────────────────────────

func sampleUser() *domain.User {
	return &domain.User{
		ID:        "user-1",
		Email:     "kim@example.com",
		Name:      "Kim Johnson",
		AvatarURL: "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ── UpdateProfile tests ───────────────────────────────────────────────────────

func TestProfile_Update_OK(t *testing.T) {
	repo := &mockProfileUserRepo{user: sampleUser()}
	svc := NewProfileService(repo, nil)

	info, err := svc.UpdateProfile("user-1", UpdateProfileRequest{Name: "New Name", Email: "new@example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Name != "New Name" {
		t.Errorf("name: got %q, want %q", info.Name, "New Name")
	}
	if info.Email != "new@example.com" {
		t.Errorf("email: got %q, want %q", info.Email, "new@example.com")
	}
}

func TestProfile_Update_EmptyName(t *testing.T) {
	repo := &mockProfileUserRepo{user: sampleUser()}
	svc := NewProfileService(repo, nil)

	_, err := svc.UpdateProfile("user-1", UpdateProfileRequest{Name: "", Email: "a@b.com"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ae, ok := err.(*apperror.AppError); !ok || ae.Code != "BAD_REQUEST" {
		t.Errorf("expected BAD_REQUEST, got %v", err)
	}
}

func TestProfile_Update_InvalidEmail(t *testing.T) {
	repo := &mockProfileUserRepo{user: sampleUser()}
	svc := NewProfileService(repo, nil)

	_, err := svc.UpdateProfile("user-1", UpdateProfileRequest{Name: "Kim", Email: "not-an-email"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ae, ok := err.(*apperror.AppError); !ok || ae.Code != "BAD_REQUEST" {
		t.Errorf("expected BAD_REQUEST, got %v", err)
	}
}

func TestProfile_Update_EmailTaken(t *testing.T) {
	existing := &domain.User{ID: "user-2", Email: "taken@example.com"}
	repo := &mockProfileUserRepo{user: sampleUser(), findByEmail: existing}
	svc := NewProfileService(repo, nil)

	_, err := svc.UpdateProfile("user-1", UpdateProfileRequest{Name: "Kim", Email: "taken@example.com"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ae, ok := err.(*apperror.AppError); !ok || ae.Code != "CONFLICT" {
		t.Errorf("expected CONFLICT, got %v", err)
	}
}

// ── UploadAvatar tests ────────────────────────────────────────────────────────

func TestProfile_UploadAvatar_OK(t *testing.T) {
	repo := &mockProfileUserRepo{user: sampleUser()}
	uploader := &mockUploader{url: "https://r2.example.com/avatars/user-1/abc.jpg"}
	svc := NewProfileService(repo, uploader)

	data := strings.NewReader("fake-image-bytes")
	url, err := svc.UploadAvatar("user-1", "photo.jpg", "image/jpeg", int64(data.Len()), data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(url, "https://") {
		t.Errorf("expected URL, got %q", url)
	}
	if repo.user.AvatarURL == "" {
		t.Error("expected AvatarURL to be set on user")
	}
}

func TestProfile_UploadAvatar_UnsupportedType(t *testing.T) {
	repo := &mockProfileUserRepo{user: sampleUser()}
	svc := NewProfileService(repo, &mockUploader{url: "https://r2.example.com/x"})

	_, err := svc.UploadAvatar("user-1", "file.gif", "image/gif", 100, bytes.NewReader([]byte("data")))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ae, ok := err.(*apperror.AppError); !ok || ae.Code != "BAD_REQUEST" {
		t.Errorf("expected BAD_REQUEST, got %v", err)
	}
}

func TestProfile_UploadAvatar_TooLarge(t *testing.T) {
	repo := &mockProfileUserRepo{user: sampleUser()}
	svc := NewProfileService(repo, &mockUploader{url: "https://r2.example.com/x"})

	const sixMB = 6 * 1024 * 1024
	_, err := svc.UploadAvatar("user-1", "big.jpg", "image/jpeg", sixMB, bytes.NewReader([]byte{}))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ae, ok := err.(*apperror.AppError); !ok || ae.Code != "BAD_REQUEST" {
		t.Errorf("expected BAD_REQUEST, got %v", err)
	}
}

// ── DeleteAccount tests ───────────────────────────────────────────────────────

func sampleUserWithPassword(pw string) *domain.User {
	hash, _ := hashutil.Hash(pw)
	u := sampleUser()
	u.PasswordHash = hash
	return u
}

func TestProfile_DeleteAccount_OK(t *testing.T) {
	repo := &mockProfileUserRepo{user: sampleUserWithPassword("correct-pw")}
	svc := NewProfileService(repo, nil)

	err := svc.DeleteAccount("user-1", "correct-pw")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.deleted {
		t.Error("expected Delete to be called on repo")
	}
}

func TestProfile_DeleteAccount_WrongPassword(t *testing.T) {
	repo := &mockProfileUserRepo{user: sampleUserWithPassword("correct-pw")}
	svc := NewProfileService(repo, nil)

	err := svc.DeleteAccount("user-1", "wrong-pw")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ae, ok := err.(*apperror.AppError); !ok || ae.Code != "BAD_REQUEST" {
		t.Errorf("expected BAD_REQUEST, got %v", err)
	}
	if repo.deleted {
		t.Error("expected Delete NOT to be called on wrong password")
	}
}

func TestProfile_DeleteAccount_MissingPassword(t *testing.T) {
	repo := &mockProfileUserRepo{user: sampleUserWithPassword("correct-pw")}
	svc := NewProfileService(repo, nil)

	err := svc.DeleteAccount("user-1", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ae, ok := err.(*apperror.AppError); !ok || ae.Code != "BAD_REQUEST" {
		t.Errorf("expected BAD_REQUEST, got %v", err)
	}
}

// ── locale tests ──────────────────────────────────────────────────────────────

func TestProfile_UpdateProfile_UpdatesLocale_OK(t *testing.T) {
	repo := &mockProfileUserRepo{user: sampleUser()}
	svc := NewProfileService(repo, nil)

	info, err := svc.UpdateProfile("user-1", UpdateProfileRequest{Name: "Kim", Email: "kim@example.com", Locale: "th"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Locale != "th" {
		t.Errorf("locale: got %q, want %q", info.Locale, "th")
	}
	if repo.user.Locale != "th" {
		t.Error("expected Locale to be persisted on user")
	}
}

func TestProfile_UpdateProfile_InvalidLocale(t *testing.T) {
	repo := &mockProfileUserRepo{user: sampleUser()}
	svc := NewProfileService(repo, nil)

	_, err := svc.UpdateProfile("user-1", UpdateProfileRequest{Name: "Kim", Email: "kim@example.com", Locale: "fr"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ae, ok := err.(*apperror.AppError); !ok || ae.Code != "BAD_REQUEST" {
		t.Errorf("expected BAD_REQUEST, got %v", err)
	}
}
