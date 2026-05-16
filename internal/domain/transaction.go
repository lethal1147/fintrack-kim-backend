package domain

import "time"

type TransactionType string

const (
	TransactionIncome  TransactionType = "income"
	TransactionExpense TransactionType = "expense"
)

type Transaction struct {
	ID       string
	UserID   string
	Merchant string
	Category string
	Date     time.Time
	Amount   float64
	Type     TransactionType
}
