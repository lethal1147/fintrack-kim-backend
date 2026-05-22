package postgres

import (
	"time"

	"gorm.io/gorm"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

type totpBackupCodeModel struct {
	ID        string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    string     `gorm:"not null;index"`
	CodeHash  string     `gorm:"not null"`
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (totpBackupCodeModel) TableName() string { return "totp_backup_codes" }

type TOTPRepo struct{ db *gorm.DB }

func NewTOTPRepo(db *gorm.DB) *TOTPRepo { return &TOTPRepo{db: db} }

func (r *TOTPRepo) CreateBackupCodes(codes []*domain.TOTPBackupCode) error {
	models := make([]*totpBackupCodeModel, len(codes))
	for i, c := range codes {
		models[i] = &totpBackupCodeModel{UserID: c.UserID, CodeHash: c.CodeHash}
	}
	if err := r.db.Create(&models).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	for i, m := range models {
		codes[i].ID = m.ID
		codes[i].CreatedAt = m.CreatedAt
	}
	return nil
}

func (r *TOTPRepo) FindUnusedBackupCodes(userID string) ([]*domain.TOTPBackupCode, error) {
	var models []totpBackupCodeModel
	if err := r.db.Where("user_id = ? AND used_at IS NULL", userID).Find(&models).Error; err != nil {
		return nil, apperror.Internal(err.Error())
	}
	codes := make([]*domain.TOTPBackupCode, len(models))
	for i, m := range models {
		codes[i] = &domain.TOTPBackupCode{
			ID: m.ID, UserID: m.UserID, CodeHash: m.CodeHash,
			UsedAt: m.UsedAt, CreatedAt: m.CreatedAt,
		}
	}
	return codes, nil
}

func (r *TOTPRepo) MarkBackupCodeUsed(id string) error {
	now := time.Now()
	if err := r.db.Model(&totpBackupCodeModel{}).Where("id = ?", id).Update("used_at", now).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	return nil
}

func (r *TOTPRepo) DeleteBackupCodes(userID string) error {
	if err := r.db.Delete(&totpBackupCodeModel{}, "user_id = ?", userID).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	return nil
}
