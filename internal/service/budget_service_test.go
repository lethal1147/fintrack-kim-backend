package service

import (
	"testing"
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
)

// ── mock repos ───────────────────────────────────────────────────────────────

type mockBudgetRepo struct {
	items     []domain.BudgetCategory
	createErr error
	getErr    error
	updateErr error
	deleteErr error
}

func (m *mockBudgetRepo) List(_ string) ([]domain.BudgetCategory, error) {
	return m.items, nil
}

func (m *mockBudgetRepo) Create(_ string, item *domain.BudgetCategory) error {
	if m.createErr != nil {
		return m.createErr
	}
	item.ID = "bud-" + item.Name
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()
	return nil
}

func (m *mockBudgetRepo) GetByID(_, id string) (*domain.BudgetCategory, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, item := range m.items {
		if item.ID == id {
			cp := item
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *mockBudgetRepo) Update(_ string, item *domain.BudgetCategory) error {
	return m.updateErr
}

func (m *mockBudgetRepo) Delete(_, _ string) error {
	return m.deleteErr
}

// mockBudgetTxRepo satisfies domain.TransactionRepository; only Summary is used by BudgetService.
type mockBudgetTxRepo struct {
	summary *domain.TransactionSummaryResult
	sumErr  error
}

func (m *mockBudgetTxRepo) Create(_ *domain.Transaction) error { return nil }
func (m *mockBudgetTxRepo) GetByID(_, _ string) (*domain.Transaction, error) {
	return nil, nil
}
func (m *mockBudgetTxRepo) Update(_ *domain.Transaction) error { return nil }
func (m *mockBudgetTxRepo) SoftDelete(_, _ string) error        { return nil }
func (m *mockBudgetTxRepo) List(_ string, _ domain.TransactionFilter) ([]*domain.Transaction, int, error) {
	return nil, 0, nil
}
func (m *mockBudgetTxRepo) Summary(_ string, _, _ time.Time) (*domain.TransactionSummaryResult, error) {
	if m.sumErr != nil {
		return nil, m.sumErr
	}
	if m.summary != nil {
		return m.summary, nil
	}
	return &domain.TransactionSummaryResult{}, nil
}
func (m *mockBudgetTxRepo) Analytics(_ string, _, _ time.Time) (*domain.AnalyticsResult, error) {
	return &domain.AnalyticsResult{}, nil
}

func newBudgetSvc(repo *mockBudgetRepo, txRepo domain.TransactionRepository) *BudgetService {
	return NewBudgetService(repo, txRepo)
}

// ── List tests ────────────────────────────────────────────────────────────────

func TestBudget_List_ComputesSpent(t *testing.T) {
	repo := &mockBudgetRepo{items: []domain.BudgetCategory{
		{ID: "b1", Name: "Housing", Group: domain.BudgetGroupFixed, Budgeted: 1200},
		{ID: "b2", Name: "Food & Dining", Group: domain.BudgetGroupFlexible, Budgeted: 500},
	}}
	txRepo := &mockBudgetTxRepo{summary: &domain.TransactionSummaryResult{
		ByCategory: []domain.CategoryStat{
			{Category: "Housing", Type: domain.TransactionExpense, Total: 1200},
			{Category: "Food & Dining", Type: domain.TransactionExpense, Total: 320},
			{Category: "Housing", Type: domain.TransactionIncome, Total: 50}, // income — should be ignored
		},
	}}
	svc := newBudgetSvc(repo, txRepo)
	cats, err := svc.List("user-1", 2026, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cats) != 2 {
		t.Fatalf("len = %d, want 2", len(cats))
	}
	for _, c := range cats {
		switch c.Name {
		case "Housing":
			if c.Spent != 1200 {
				t.Errorf("Housing.Spent = %v, want 1200", c.Spent)
			}
		case "Food & Dining":
			if c.Spent != 320 {
				t.Errorf("Food.Spent = %v, want 320", c.Spent)
			}
		}
	}
}

func TestBudget_List_NoMatchingTransactions(t *testing.T) {
	repo := &mockBudgetRepo{items: []domain.BudgetCategory{
		{ID: "b1", Name: "Housing", Group: domain.BudgetGroupFixed, Budgeted: 1200},
	}}
	txRepo := &mockBudgetTxRepo{summary: &domain.TransactionSummaryResult{
		ByCategory: []domain.CategoryStat{
			{Category: "Transport", Type: domain.TransactionExpense, Total: 100},
		},
	}}
	svc := newBudgetSvc(repo, txRepo)
	cats, err := svc.List("user-1", 2026, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cats[0].Spent != 0 {
		t.Errorf("Spent = %v, want 0 (no matching transaction)", cats[0].Spent)
	}
}

// ── Create tests ──────────────────────────────────────────────────────────────

func validBudgetReq() domain.CreateBudgetRequest {
	return domain.CreateBudgetRequest{
		Name:     "Housing",
		Group:    domain.BudgetGroupFixed,
		Budgeted: 1200,
		Color:    "var(--chart-1)",
	}
}

func TestBudget_Create_OK(t *testing.T) {
	svc := newBudgetSvc(&mockBudgetRepo{}, &mockBudgetTxRepo{})
	item, err := svc.Create("user-1", validBudgetReq())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Name != "Housing" {
		t.Errorf("Name = %q, want Housing", item.Name)
	}
	if item.ID == "" {
		t.Error("ID should be set after Create")
	}
}

func TestBudget_Create_EmptyName(t *testing.T) {
	svc := newBudgetSvc(&mockBudgetRepo{}, &mockBudgetTxRepo{})
	req := validBudgetReq()
	req.Name = ""
	_, err := svc.Create("user-1", req)
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
}

func TestBudget_Create_NegativeBudget(t *testing.T) {
	svc := newBudgetSvc(&mockBudgetRepo{}, &mockBudgetTxRepo{})
	req := validBudgetReq()
	req.Budgeted = -1
	_, err := svc.Create("user-1", req)
	if err == nil {
		t.Fatal("expected error for negative budget, got nil")
	}
}

// ── Delete test ───────────────────────────────────────────────────────────────

func TestBudget_Delete_OK(t *testing.T) {
	repo := &mockBudgetRepo{items: []domain.BudgetCategory{
		{ID: "b1", Name: "Housing", Group: domain.BudgetGroupFixed, Budgeted: 1200},
	}}
	svc := newBudgetSvc(repo, &mockBudgetTxRepo{})
	if err := svc.Delete("user-1", "b1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
