package domain

import "time"

type TransactionType string

const (
	TransactionIncome  TransactionType = "income"
	TransactionExpense TransactionType = "expense"
)

// ValidCategories is the authoritative set accepted by the service layer.
var ValidCategories = map[string]bool{
	// income
	"Salary": true, "Freelance": true, "Business": true,
	"Gift Received": true, "Investment Returns": true,
	"Rental Income": true, "Other Income": true,
	// expense
	"Housing": true, "Food & Dining": true, "Transport": true,
	"Entertainment": true, "Health": true, "Shopping": true,
	"Utilities": true, "Education": true, "Investment": true,
	"Travel": true, "Gifts & Donations": true, "Subscriptions": true,
	"Other": true,
}

type Transaction struct {
	ID        string
	UserID    string
	Merchant  string
	Category  string
	Note      string
	Date      time.Time
	Amount    float64
	Type      TransactionType
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time // nil = active
}

type TransactionFilter struct {
	Type     string // "" = all
	Category string // "" = all
	Search   string // ILIKE against merchant + note
	From     string // YYYY-MM-DD, "" = unset
	To       string // YYYY-MM-DD, "" = unset
	Page     int    // 1-based, default 1
	Limit    int    // default 8, max 100
}

type TransactionListResult struct {
	Transactions []*Transaction
	Total        int
	Page         int
	Limit        int
	Pages        int
}

type CategoryStat struct {
	Category string
	Type     TransactionType
	Total    float64
	Count    int
	Pct      float64 // % of total for this type within the queried range
}

type MonthStat struct {
	Month   string // "YYYY-MM"
	Income  float64
	Expense float64
	Net     float64
}

type TransactionSummaryResult struct {
	TotalIncome  float64
	TotalExpense float64
	Net          float64
	ByCategory   []CategoryStat
	ByMonth      []MonthStat
}

type CreateTransactionRequest struct {
	Merchant string
	Category string
	Note     string
	Date     string // YYYY-MM-DD
	Amount   float64
	Type     TransactionType
}

type UpdateTransactionRequest = CreateTransactionRequest

type TransactionRepository interface {
	Create(tx *Transaction) error
	GetByID(id, userID string) (*Transaction, error)
	Update(tx *Transaction) error
	SoftDelete(id, userID string) error
	List(userID string, f TransactionFilter) ([]*Transaction, int, error)
	Summary(userID string, from, to time.Time) (*TransactionSummaryResult, error)
}

type TransactionServiceInterface interface {
	Create(userID string, req CreateTransactionRequest) (*Transaction, error)
	Get(id, userID string) (*Transaction, error)
	Update(id, userID string, req UpdateTransactionRequest) (*Transaction, error)
	Delete(id, userID string) error
	List(userID string, f TransactionFilter) (*TransactionListResult, error)
	Summary(userID string, from, to time.Time) (*TransactionSummaryResult, error)
}
