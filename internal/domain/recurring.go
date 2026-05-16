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
}
