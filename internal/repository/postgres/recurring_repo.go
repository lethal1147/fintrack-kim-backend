package postgres

import (
	"time"

	"gorm.io/gorm"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

type recurringModel struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    string    `gorm:"not null;index"`
	Name      string    `gorm:"not null"`
	Category  string    `gorm:"not null"`
	Amount    float64   `gorm:"not null;type:decimal(12,2)"`
	Frequency string    `gorm:"not null"`
	Kind      string    `gorm:"not null"`
	Status    string    `gorm:"not null;default:'active'"`
	Color     string    `gorm:"not null;default:'#0ea5e9'"`
	NextDue   time.Time `gorm:"not null;type:date"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (recurringModel) TableName() string { return "recurring_items" }

func toRecurringDomain(m *recurringModel) *domain.RecurringItem {
	return &domain.RecurringItem{
		ID:        m.ID,
		UserID:    m.UserID,
		Name:      m.Name,
		Category:  m.Category,
		Amount:    m.Amount,
		Frequency: domain.RecurringFrequency(m.Frequency),
		Kind:      domain.RecurringKind(m.Kind),
		Status:    domain.RecurringStatus(m.Status),
		Color:     m.Color,
		NextDue:   m.NextDue,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

type RecurringRepo struct{ db *gorm.DB }

func NewRecurringRepo(db *gorm.DB) *RecurringRepo { return &RecurringRepo{db: db} }

func (r *RecurringRepo) Create(userID string, item *domain.RecurringItem) error {
	m := &recurringModel{
		UserID:    userID,
		Name:      item.Name,
		Category:  item.Category,
		Amount:    item.Amount,
		Frequency: string(item.Frequency),
		Kind:      string(item.Kind),
		Status:    string(item.Status),
		Color:     item.Color,
		NextDue:   item.NextDue,
	}
	if err := r.db.Create(m).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	item.ID = m.ID
	item.CreatedAt = m.CreatedAt
	item.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *RecurringRepo) List(userID string) ([]domain.RecurringItem, error) {
	var rows []recurringModel
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, apperror.Internal(err.Error())
	}
	items := make([]domain.RecurringItem, len(rows))
	for i, row := range rows {
		items[i] = *toRecurringDomain(&row)
	}
	return items, nil
}

func (r *RecurringRepo) GetByID(userID, id string) (*domain.RecurringItem, error) {
	var m recurringModel
	err := r.db.Where("user_id = ? AND id = ?", userID, id).First(&m).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("recurring item not found")
		}
		return nil, apperror.Internal(err.Error())
	}
	return toRecurringDomain(&m), nil
}

func (r *RecurringRepo) Update(userID string, item *domain.RecurringItem) error {
	result := r.db.Model(&recurringModel{}).
		Where("user_id = ? AND id = ?", userID, item.ID).
		Updates(map[string]interface{}{
			"name":      item.Name,
			"category":  item.Category,
			"amount":    item.Amount,
			"frequency": string(item.Frequency),
			"kind":      string(item.Kind),
			"color":     item.Color,
			"next_due":  item.NextDue,
		})
	if result.Error != nil {
		return apperror.Internal(result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("recurring item not found")
	}
	return nil
}

func (r *RecurringRepo) SetStatus(userID, id string, status domain.RecurringStatus) error {
	result := r.db.Model(&recurringModel{}).
		Where("user_id = ? AND id = ?", userID, id).
		Update("status", string(status))
	if result.Error != nil {
		return apperror.Internal(result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("recurring item not found")
	}
	return nil
}

func (r *RecurringRepo) Delete(userID, id string) error {
	result := r.db.Where("user_id = ? AND id = ?", userID, id).Delete(&recurringModel{})
	if result.Error != nil {
		return apperror.Internal(result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("recurring item not found")
	}
	return nil
}

func (r *RecurringRepo) ListDue(userID string, asOf time.Time) ([]domain.RecurringItem, error) {
	var rows []recurringModel
	err := r.db.Where("user_id = ? AND status = 'active' AND next_due <= ?", userID, asOf).
		Find(&rows).Error
	if err != nil {
		return nil, apperror.Internal(err.Error())
	}
	items := make([]domain.RecurringItem, len(rows))
	for i, row := range rows {
		items[i] = *toRecurringDomain(&row)
	}
	return items, nil
}

func (r *RecurringRepo) UpdateNextDue(id string, nextDue time.Time) error {
	result := r.db.Model(&recurringModel{}).Where("id = ?", id).Update("next_due", nextDue)
	if result.Error != nil {
		return apperror.Internal(result.Error.Error())
	}
	return nil
}
