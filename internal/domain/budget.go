package domain

type BudgetGroup string

const (
	BudgetGroupFixed      BudgetGroup = "Fixed"
	BudgetGroupFlexible   BudgetGroup = "Flexible"
	BudgetGroupNonMonthly BudgetGroup = "Non-Monthly"
)

type BudgetCategory struct {
	ID       string
	UserID   string
	Name     string
	Budgeted float64
	Spent    float64
	Color    string
	Group    BudgetGroup
}
