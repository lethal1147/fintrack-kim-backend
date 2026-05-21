package domain

import "time"

type BudgetGroup string

const (
	BudgetGroupFixed      BudgetGroup = "Fixed"
	BudgetGroupFlexible   BudgetGroup = "Flexible"
	BudgetGroupNonMonthly BudgetGroup = "Non-Monthly"
)

type BudgetCategory struct {
	ID        string
	UserID    string
	Name      string      // matches transaction.Category exactly
	Group     BudgetGroup
	Budgeted  float64
	Spent     float64 // computed by service — never stored in DB
	Color     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateBudgetRequest struct {
	Name     string
	Group    BudgetGroup
	Budgeted float64
	Color    string
}

type BudgetRepository interface {
	List(userID string) ([]BudgetCategory, error)
	Create(userID string, item *BudgetCategory) error
	GetByID(userID, id string) (*BudgetCategory, error)
	Update(userID string, item *BudgetCategory) error
	Delete(userID, id string) error
}

type BudgetServiceInterface interface {
	List(userID string, year, month int) ([]BudgetCategory, error)
	Create(userID string, req CreateBudgetRequest) (*BudgetCategory, error)
	Update(userID, id string, req CreateBudgetRequest) (*BudgetCategory, error)
	Delete(userID, id string) error
}
