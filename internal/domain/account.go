package domain

type AccountType string

const (
	AccountChecking    AccountType = "checking"
	AccountSavings     AccountType = "savings"
	AccountCreditCard  AccountType = "credit_card"
	AccountLoan        AccountType = "loan"
	AccountInvestment  AccountType = "investment"
)

type Account struct {
	ID          string
	UserID      string
	Name        string
	Institution string
	Type        AccountType
	Balance     float64
	LastFour    string
}
