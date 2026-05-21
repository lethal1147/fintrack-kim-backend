package service

import (
	"testing"
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
)

// ── mock repos ──────────────────────────────────────────────────────────────

type mockRecurringRepo struct {
	items     []domain.RecurringItem
	createErr error
	getErr    error
	updateErr error
	deleteErr error
}

func (m *mockRecurringRepo) Create(_ string, item *domain.RecurringItem) error {
	if m.createErr != nil {
		return m.createErr
	}
	item.ID = "rec-" + item.Name
	item.CreatedAt = time.Now()
	item.UpdatedAt = time.Now()
	return nil
}

func (m *mockRecurringRepo) List(_ string) ([]domain.RecurringItem, error) {
	return m.items, nil
}

func (m *mockRecurringRepo) GetByID(_, id string) (*domain.RecurringItem, error) {
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

func (m *mockRecurringRepo) Update(_ string, item *domain.RecurringItem) error {
	return m.updateErr
}

func (m *mockRecurringRepo) SetStatus(_, _ string, status domain.RecurringStatus) error {
	return nil
}

func (m *mockRecurringRepo) Delete(_, _ string) error {
	return m.deleteErr
}

func (m *mockRecurringRepo) ListDue(_ string, asOf time.Time) ([]domain.RecurringItem, error) {
	var due []domain.RecurringItem
	for _, item := range m.items {
		if item.Status == domain.StatusActive && !item.NextDue.After(asOf) {
			due = append(due, item)
		}
	}
	return due, nil
}

func (m *mockRecurringRepo) UpdateNextDue(id string, nextDue time.Time) error {
	for i, item := range m.items {
		if item.ID == id {
			m.items[i].NextDue = nextDue
			return nil
		}
	}
	return nil
}

type mockTxCreator struct {
	created []*domain.Transaction
	err     error
}

func (m *mockTxCreator) Create(tx *domain.Transaction) error {
	if m.err != nil {
		return m.err
	}
	tx.ID = "tx-auto"
	m.created = append(m.created, tx)
	return nil
}

func (m *mockTxCreator) GetByID(_, _ string) (*domain.Transaction, error) { return nil, nil }
func (m *mockTxCreator) Update(_ *domain.Transaction) error               { return nil }
func (m *mockTxCreator) SoftDelete(_, _ string) error                     { return nil }
func (m *mockTxCreator) List(_ string, _ domain.TransactionFilter) ([]*domain.Transaction, int, error) {
	return nil, 0, nil
}
func (m *mockTxCreator) Summary(_ string, _, _ time.Time) (*domain.TransactionSummaryResult, error) {
	return &domain.TransactionSummaryResult{}, nil
}
func (m *mockTxCreator) Analytics(_ string, _, _ time.Time) (*domain.AnalyticsResult, error) {
	return &domain.AnalyticsResult{}, nil
}

func newRecurringSvc(repo *mockRecurringRepo, txRepo domain.TransactionRepository) *RecurringService {
	return NewRecurringService(repo, txRepo)
}

// ── Create tests ─────────────────────────────────────────────────────────────

func validRecurringReq() domain.CreateRecurringRequest {
	return domain.CreateRecurringRequest{
		Name:      "Netflix",
		Category:  "Subscriptions",
		Amount:    15.99,
		Frequency: domain.FrequencyMonthly,
		Kind:      domain.KindExpense,
		NextDue:   time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC),
		Color:     "#0ea5e9",
	}
}

func TestRecurring_Create_OK(t *testing.T) {
	repo := &mockRecurringRepo{}
	svc := newRecurringSvc(repo, &mockTxCreator{})
	item, err := svc.Create("user-1", validRecurringReq())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Name != "Netflix" {
		t.Errorf("Name = %q, want Netflix", item.Name)
	}
	if item.Status != domain.StatusActive {
		t.Errorf("Status = %q, want active", item.Status)
	}
}

func TestRecurring_Create_InvalidAmount(t *testing.T) {
	repo := &mockRecurringRepo{}
	svc := newRecurringSvc(repo, &mockTxCreator{})
	req := validRecurringReq()
	req.Amount = 0
	_, err := svc.Create("user-1", req)
	if err == nil {
		t.Fatal("expected error for zero amount, got nil")
	}
}

func TestRecurring_List_Empty(t *testing.T) {
	repo := &mockRecurringRepo{}
	svc := newRecurringSvc(repo, &mockTxCreator{})
	items, err := svc.List("user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items))
	}
}

// ── Process tests ─────────────────────────────────────────────────────────────

func TestProcess_CatchUp(t *testing.T) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	twoMonthsAgo := today.AddDate(0, -2, 0)

	repo := &mockRecurringRepo{items: []domain.RecurringItem{{
		ID:        "rec-1",
		Name:      "Rent",
		Category:  "Housing",
		Amount:    1000,
		Frequency: domain.FrequencyMonthly,
		Kind:      domain.KindExpense,
		Status:    domain.StatusActive,
		NextDue:   twoMonthsAgo,
	}}}
	txCreator := &mockTxCreator{}
	svc := newRecurringSvc(repo, txCreator)

	n, err := svc.Process("user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n < 2 {
		t.Errorf("generated = %d, want >= 2 (catch-up)", n)
	}
	if len(txCreator.created) != n {
		t.Errorf("txCreator.created = %d, want %d", len(txCreator.created), n)
	}
	// next_due should be in the future
	updatedItem := repo.items[0]
	if !updatedItem.NextDue.After(today) {
		t.Errorf("next_due = %v, want > today %v", updatedItem.NextDue, today)
	}
}

func TestProcess_WeekendAdjust_Saturday(t *testing.T) {
	// 2026-01-29 (Thursday) + 1 month = 2026-02-28 (Saturday) → should become Friday 2026-02-27
	thursday := time.Date(2026, 1, 29, 0, 0, 0, 0, time.UTC)
	result := advanceNextDue(thursday, domain.FrequencyMonthly)
	expected := time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC) // Friday
	if !result.Equal(expected) {
		t.Errorf("advanceNextDue(2026-01-29, monthly) = %v, want %v (Friday)", result, expected)
	}
}

func TestProcess_WeekendAdjust_Sunday(t *testing.T) {
	// 2026-02-28 (Saturday) + 1 month = 2026-03-28 (Saturday) → Friday; but let's use a Sunday case
	// 2025-11-30 (Sunday) + 1 month = 2025-12-30 (Tuesday) → no adjust
	// Use: 2026-04-30 + 1 month = 2026-05-30 (Saturday) → Friday 2026-05-29
	// Actually let's find a date where +1 month lands on Sunday
	// 2026-01-31 + 1 month = 2026-02-28 (Saturday). Try 2026-03-01 + 1 month = 2026-04-01 (Wednesday)
	// 2025-08-31 + 1 month = 2025-09-30 (Tuesday). Try to find Sunday:
	// Let's verify adjustWeekend directly
	sunday := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC) // verify it's Sunday
	if sunday.Weekday() != time.Sunday {
		t.Skipf("date not Sunday: %v", sunday.Weekday())
	}
	result := adjustWeekend(sunday)
	expected := time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC) // Friday
	if !result.Equal(expected) {
		t.Errorf("adjustWeekend(Sunday 2026-03-08) = %v, want %v (Friday)", result, expected)
	}
}

func TestProcess_PausedSkipped(t *testing.T) {
	today := time.Now().UTC()
	repo := &mockRecurringRepo{items: []domain.RecurringItem{{
		ID:        "rec-paused",
		Name:      "Paused",
		Frequency: domain.FrequencyMonthly,
		Kind:      domain.KindExpense,
		Status:    domain.StatusPaused,
		NextDue:   today.AddDate(0, -1, 0),
	}}}
	txCreator := &mockTxCreator{}
	svc := newRecurringSvc(repo, txCreator)

	n, err := svc.Process("user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("generated = %d for paused item, want 0", n)
	}
}

func TestAdvance_EndOfMonth_Monthly(t *testing.T) {
	// June 30 (last day) → monthly → July 31 (last day), no weekend adjust (2026-07-31 is Fri)
	june30 := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	result := advanceNextDue(june30, domain.FrequencyMonthly)
	expected := time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("advance(2026-06-30, monthly) = %v, want %v", result, expected)
	}
}

func TestAdvance_EndOfMonth_KeepsEndOfMonth(t *testing.T) {
	// May 31 → June 30 → July 31 — end-of-month stays at end-of-month
	may31 := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	jun30 := advanceNextDue(may31, domain.FrequencyMonthly)
	if jun30.Month() != time.June || !isLastDayOfMonth(jun30) {
		t.Errorf("advance(May 31) = %v, want last day of June", jun30)
	}
	jul31 := advanceNextDue(jun30, domain.FrequencyMonthly)
	if jul31.Month() != time.July || !isLastDayOfMonth(jul31) {
		t.Errorf("advance(Jun 30) = %v, want last day of July", jul31)
	}
}

func TestAdvance_EndOfMonth_Annual(t *testing.T) {
	// March 31 → annual → March 31 next year
	mar31 := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	result := advanceNextDue(mar31, domain.FrequencyAnnual)
	expected := time.Date(2027, 3, 31, 0, 0, 0, 0, time.UTC)
	if !result.Equal(expected) {
		t.Errorf("advance(2026-03-31, annual) = %v, want %v", result, expected)
	}
}

func TestProcess_NothingDue(t *testing.T) {
	tomorrow := time.Now().UTC().AddDate(0, 0, 1)
	repo := &mockRecurringRepo{items: []domain.RecurringItem{{
		ID:        "rec-future",
		Name:      "Future",
		Frequency: domain.FrequencyMonthly,
		Kind:      domain.KindExpense,
		Status:    domain.StatusActive,
		NextDue:   tomorrow,
	}}}
	txCreator := &mockTxCreator{}
	svc := newRecurringSvc(repo, txCreator)

	n, err := svc.Process("user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("generated = %d for future item, want 0", n)
	}
}
