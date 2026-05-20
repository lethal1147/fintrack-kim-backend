package domain

import "time"

type RecurringFrequency string
type RecurringStatus string
type RecurringKind string

const (
	FrequencyWeekly  RecurringFrequency = "weekly"
	FrequencyMonthly RecurringFrequency = "monthly"
	FrequencyAnnual  RecurringFrequency = "annual"

	StatusActive RecurringStatus = "active"
	StatusPaused RecurringStatus = "paused"

	KindExpense RecurringKind = "expense"
	KindIncome  RecurringKind = "income"
)

type RecurringItem struct {
	ID        string
	UserID    string
	Name      string
	Category  string
	Amount    float64
	Frequency RecurringFrequency
	NextDue   time.Time
	Kind      RecurringKind
	Status    RecurringStatus
	Color     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateRecurringRequest struct {
	Name      string
	Category  string
	Amount    float64
	Frequency RecurringFrequency
	Kind      RecurringKind
	NextDue   time.Time
	Color     string
}

type RecurringRepository interface {
	Create(userID string, item *RecurringItem) error
	List(userID string) ([]RecurringItem, error)
	GetByID(userID, id string) (*RecurringItem, error)
	Update(userID string, item *RecurringItem) error
	SetStatus(userID, id string, status RecurringStatus) error
	Delete(userID, id string) error
	ListDue(userID string, asOf time.Time) ([]RecurringItem, error)
	UpdateNextDue(id string, nextDue time.Time) error
}

type RecurringServiceInterface interface {
	List(userID string) ([]RecurringItem, error)
	Create(userID string, req CreateRecurringRequest) (*RecurringItem, error)
	Update(userID, id string, req CreateRecurringRequest) (*RecurringItem, error)
	SetStatus(userID, id string, status RecurringStatus) (*RecurringItem, error)
	Delete(userID, id string) error
	Process(userID string) (int, error)
}
