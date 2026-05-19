package service

import (
	"testing"
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

// ── mock repo ──────────────────────────────────────────────────────────────

type mockTxRepo struct {
	txs          map[string]*domain.Transaction
	createErr    error
	softDeleteID string // last id passed to SoftDelete
	notFoundID   string // returns NotFound for this id
}

func newMockTxRepo() *mockTxRepo {
	return &mockTxRepo{txs: make(map[string]*domain.Transaction)}
}

func (m *mockTxRepo) Create(tx *domain.Transaction) error {
	if m.createErr != nil {
		return m.createErr
	}
	tx.ID = "tx-" + tx.Merchant
	tx.CreatedAt = time.Now()
	tx.UpdatedAt = time.Now()
	m.txs[tx.ID] = tx
	return nil
}

func (m *mockTxRepo) GetByID(id, _ string) (*domain.Transaction, error) {
	if id == m.notFoundID {
		return nil, apperror.NotFound("transaction not found")
	}
	if tx, ok := m.txs[id]; ok {
		return tx, nil
	}
	return nil, apperror.NotFound("transaction not found")
}

func (m *mockTxRepo) Update(tx *domain.Transaction) error {
	m.txs[tx.ID] = tx
	return nil
}

func (m *mockTxRepo) SoftDelete(id, _ string) error {
	m.softDeleteID = id
	if id == m.notFoundID {
		return apperror.NotFound("transaction not found")
	}
	return nil
}

func (m *mockTxRepo) List(_ string, f domain.TransactionFilter) ([]*domain.Transaction, int, error) {
	var result []*domain.Transaction
	for _, tx := range m.txs {
		result = append(result, tx)
	}
	return result, len(result), nil
}

func (m *mockTxRepo) Summary(_ string, _, _ time.Time) (*domain.TransactionSummaryResult, error) {
	return &domain.TransactionSummaryResult{}, nil
}

// ── helpers ────────────────────────────────────────────────────────────────

func validCreateReq() domain.CreateTransactionRequest {
	return domain.CreateTransactionRequest{
		Merchant: "SCB Bank",
		Category: "Salary",
		Note:     "March pay",
		Date:     "2026-03-25",
		Amount:   42800.00,
		Type:     domain.TransactionIncome,
	}
}

func newSvc(repo domain.TransactionRepository) *TransactionService {
	return NewTransactionService(repo)
}

// ── Create tests ───────────────────────────────────────────────────────────

func TestCreate_Valid(t *testing.T) {
	repo := newMockTxRepo()
	svc := newSvc(repo)

	tx, err := svc.Create("user-1", validCreateReq())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tx.ID == "" {
		t.Error("expected non-empty ID")
	}
	if tx.Merchant != "SCB Bank" {
		t.Errorf("merchant = %q, want SCB Bank", tx.Merchant)
	}
}

func TestCreate_InvalidCategory(t *testing.T) {
	repo := newMockTxRepo()
	svc := newSvc(repo)

	req := validCreateReq()
	req.Category = "NotACategory"

	_, err := svc.Create("user-1", req)
	if err == nil {
		t.Fatal("expected error for invalid category")
	}
}

func TestCreate_ZeroAmount(t *testing.T) {
	repo := newMockTxRepo()
	svc := newSvc(repo)

	req := validCreateReq()
	req.Amount = 0

	_, err := svc.Create("user-1", req)
	if err == nil {
		t.Fatal("expected error for zero amount")
	}
}

func TestCreate_NegativeAmount(t *testing.T) {
	repo := newMockTxRepo()
	svc := newSvc(repo)

	req := validCreateReq()
	req.Amount = -100

	_, err := svc.Create("user-1", req)
	if err == nil {
		t.Fatal("expected error for negative amount")
	}
}

func TestCreate_InvalidDate(t *testing.T) {
	repo := newMockTxRepo()
	svc := newSvc(repo)

	req := validCreateReq()
	req.Date = "not-a-date"

	_, err := svc.Create("user-1", req)
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestCreate_MissingMerchant(t *testing.T) {
	repo := newMockTxRepo()
	svc := newSvc(repo)

	req := validCreateReq()
	req.Merchant = ""

	_, err := svc.Create("user-1", req)
	if err == nil {
		t.Fatal("expected error for empty merchant")
	}
}

func TestCreate_InvalidType(t *testing.T) {
	repo := newMockTxRepo()
	svc := newSvc(repo)

	req := validCreateReq()
	req.Type = "transfer"

	_, err := svc.Create("user-1", req)
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

// ── Get tests ──────────────────────────────────────────────────────────────

func TestGet_NotFound(t *testing.T) {
	repo := newMockTxRepo()
	repo.notFoundID = "missing-id"
	svc := newSvc(repo)

	_, err := svc.Get("missing-id", "user-1")
	if err == nil {
		t.Fatal("expected NotFound error")
	}
}

// ── Update tests ───────────────────────────────────────────────────────────

func TestUpdate_Valid(t *testing.T) {
	repo := newMockTxRepo()
	svc := newSvc(repo)

	// seed a record
	tx, _ := svc.Create("user-1", validCreateReq())

	req := validCreateReq()
	req.Merchant = "Updated Merchant"
	req.Amount = 1234.56

	updated, err := svc.Update(tx.ID, "user-1", req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Merchant != "Updated Merchant" {
		t.Errorf("merchant = %q, want Updated Merchant", updated.Merchant)
	}
}

// ── Delete tests ───────────────────────────────────────────────────────────

func TestDelete_CallsSoftDelete(t *testing.T) {
	repo := newMockTxRepo()
	svc := newSvc(repo)

	tx, _ := svc.Create("user-1", validCreateReq())

	if err := svc.Delete(tx.ID, "user-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.softDeleteID != tx.ID {
		t.Errorf("SoftDelete called with %q, want %q", repo.softDeleteID, tx.ID)
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := newMockTxRepo()
	repo.notFoundID = "ghost-id"
	svc := newSvc(repo)

	err := svc.Delete("ghost-id", "user-1")
	if err == nil {
		t.Fatal("expected NotFound error")
	}
}

// ── List tests ─────────────────────────────────────────────────────────────

func TestList_DefaultPagination(t *testing.T) {
	repo := newMockTxRepo()
	svc := newSvc(repo)

	result, err := svc.List("user-1", domain.TransactionFilter{Page: 0, Limit: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Page != 1 {
		t.Errorf("page = %d, want 1", result.Page)
	}
	if result.Limit != 8 {
		t.Errorf("limit = %d, want 8", result.Limit)
	}
}

func TestList_LimitClamped(t *testing.T) {
	repo := newMockTxRepo()
	svc := newSvc(repo)

	result, err := svc.List("user-1", domain.TransactionFilter{Limit: 999})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Limit != 100 {
		t.Errorf("limit = %d, want 100 (clamped)", result.Limit)
	}
}

// ── Summary tests ──────────────────────────────────────────────────────────

func TestSummary_ReturnsResult(t *testing.T) {
	repo := newMockTxRepo()
	svc := newSvc(repo)

	result, err := svc.Summary("user-1", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}
