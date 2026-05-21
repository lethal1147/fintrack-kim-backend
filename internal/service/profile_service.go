package service

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
	"github.com/joakim/fintrack-api/pkg/r2client"
)

// ProfileServiceInterface is the contract the handler depends on.
type ProfileServiceInterface interface {
	UpdateProfile(userID string, req UpdateProfileRequest) (*UserInfo, error)
	UploadAvatar(userID, filename, contentType string, size int64, r io.Reader) (string, error)
}

// UpdateProfileRequest carries the fields the user may update.
type UpdateProfileRequest struct {
	Name  string
	Email string
}

// ProfileService implements ProfileServiceInterface.
type ProfileService struct {
	userRepo domain.UserRepository
	uploader r2client.Uploader // nil when R2 is not configured
}

// NewProfileService creates a ProfileService.
// uploader may be nil; UploadAvatar returns an error in that case.
func NewProfileService(userRepo domain.UserRepository, uploader r2client.Uploader) *ProfileService {
	return &ProfileService{userRepo: userRepo, uploader: uploader}
}

var emailRE = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// UpdateProfile updates name and email for the given user.
func (s *ProfileService) UpdateProfile(userID string, req UpdateProfileRequest) (*UserInfo, error) {
	if req.Name == "" {
		return nil, apperror.BadRequest("name is required")
	}
	if !emailRE.MatchString(req.Email) {
		return nil, apperror.BadRequest("invalid email address")
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	// Check email uniqueness only when it changed.
	if req.Email != user.Email {
		existing, _ := s.userRepo.FindByEmail(req.Email)
		if existing != nil && existing.ID != userID {
			return nil, apperror.Conflict("email is already taken")
		}
	}

	user.Name = req.Name
	user.Email = req.Email
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return &UserInfo{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
		Provider:  string(user.Provider),
		CreatedAt: user.CreatedAt,
	}, nil
}

var allowedContentTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

const maxAvatarBytes = 5 * 1024 * 1024 // 5 MB

// UploadAvatar uploads the avatar to R2 and persists the URL on the user record.
func (s *ProfileService) UploadAvatar(userID, filename, contentType string, size int64, r io.Reader) (string, error) {
	if s.uploader == nil {
		return "", apperror.Internal("R2 not configured")
	}

	ext, ok := allowedContentTypes[contentType]
	if !ok {
		return "", apperror.BadRequest("only JPEG, PNG, and WebP images are supported")
	}
	if size > maxAvatarBytes {
		return "", apperror.BadRequest("image must be under 5 MB")
	}

	_ = filename // original filename is not used; we generate our own key
	key := r2client.RandomKey(fmt.Sprintf("avatars/%s/", userID), ext)

	url, err := s.uploader.Upload(context.Background(), key, contentType, r)
	if err != nil {
		return "", apperror.Internal(err.Error())
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return "", err
	}
	user.AvatarURL = url
	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(user); err != nil {
		return "", err
	}

	return url, nil
}
