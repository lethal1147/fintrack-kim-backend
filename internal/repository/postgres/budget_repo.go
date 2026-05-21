package postgres

import (
	"time"

	"gorm.io/gorm"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

type budgetModel struct {
	ID        string `gorm:"primaryKey"`
	UserID    string
	Name      string
	GroupName string `gorm:"column:group_name"`
	Budgeted  float64
	Color     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (budgetModel) TableName() string { return "budget_categories" }

func toBudgetDomain(m *budgetModel) domain.BudgetCategory {
	return domain.BudgetCategory{
		ID:        m.ID,
		UserID:    m.UserID,
		Name:      m.Name,
		Group:     domain.BudgetGroup(m.GroupName),
		Budgeted:  m.Budgeted,
		Color:     m.Color,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

type BudgetRepo struct {
	db *gorm.DB
}

func NewBudgetRepo(db *gorm.DB) *BudgetRepo {
	return &BudgetRepo{db: db}
}

func (r *BudgetRepo) List(userID string) ([]domain.BudgetCategory, error) {
	var models []budgetModel
	if err := r.db.Where("user_id = ?", userID).Order("created_at asc").Find(&models).Error; err != nil {
		return nil, apperror.Internal(err.Error())
	}
	cats := make([]domain.BudgetCategory, len(models))
	for i, m := range models {
		cats[i] = toBudgetDomain(&m)
	}
	return cats, nil
}

func (r *BudgetRepo) Create(userID string, item *domain.BudgetCategory) error {
	m := &budgetModel{
		UserID:    userID,
		Name:      item.Name,
		GroupName: string(item.Group),
		Budgeted:  item.Budgeted,
		Color:     item.Color,
	}
	if err := r.db.Create(m).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	item.ID = m.ID
	item.CreatedAt = m.CreatedAt
	item.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *BudgetRepo) GetByID(userID, id string) (*domain.BudgetCategory, error) {
	var m budgetModel
	if err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&m).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("budget category not found")
		}
		return nil, apperror.Internal(err.Error())
	}
	cat := toBudgetDomain(&m)
	return &cat, nil
}

func (r *BudgetRepo) Update(userID string, item *domain.BudgetCategory) error {
	result := r.db.Model(&budgetModel{}).
		Where("id = ? AND user_id = ?", item.ID, userID).
		Updates(map[string]interface{}{
			"name":       item.Name,
			"group_name": string(item.Group),
			"budgeted":   item.Budgeted,
			"color":      item.Color,
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return apperror.Internal(result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("budget category not found")
	}
	return nil
}

func (r *BudgetRepo) Delete(userID, id string) error {
	result := r.db.Where("id = ? AND user_id = ?", id, userID).Delete(&budgetModel{})
	if result.Error != nil {
		return apperror.Internal(result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("budget category not found")
	}
	return nil
}
