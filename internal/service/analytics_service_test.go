package service

import (
	"testing"
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
)

// ── mock analytics repo ────────────────────────────────────────────────────

type mockAnalyticsRepo struct {
	result *domain.AnalyticsResult
	err    error
}

func (m *mockAnalyticsRepo) Analytics(_ string, _, _ time.Time) (*domain.AnalyticsResult, error) {
	return m.result, m.err
}

func newAnalyticsSvc(r domain.AnalyticsRepository) *AnalyticsService {
	return NewAnalyticsService(r)
}

// ── tests ──────────────────────────────────────────────────────────────────

func TestAnalytics_NormalCase(t *testing.T) {
	repo := &mockAnalyticsRepo{result: &domain.AnalyticsResult{
		TotalIncome:  100_000,
		TotalExpense: 70_000,
		Net:          30_000,
		TxCount:      42,
		SavingsRate:  30.0,
		ByCategory:   []domain.CategoryStat{{Category: "Salary", Type: domain.TransactionIncome, Total: 100_000, Count: 1, Pct: 100}},
		ByMonth:      []domain.MonthStat{{Month: "2026-01", Income: 100_000, Expense: 70_000, Net: 30_000}},
		TopMerchants: []domain.MerchantStat{{Merchant: "Starbucks", Total: 5_000, Count: 20}},
	}}

	svc := newAnalyticsSvc(repo)
	result, err := svc.Analytics("user-1", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalIncome != 100_000 {
		t.Errorf("TotalIncome = %v, want 100000", result.TotalIncome)
	}
	if result.SavingsRate != 30.0 {
		t.Errorf("SavingsRate = %v, want 30.0", result.SavingsRate)
	}
	if len(result.TopMerchants) != 1 {
		t.Errorf("len(TopMerchants) = %d, want 1", len(result.TopMerchants))
	}
}

func TestAnalytics_ZeroIncome_SavingsRateIsZero(t *testing.T) {
	repo := &mockAnalyticsRepo{result: &domain.AnalyticsResult{
		TotalIncome:  0,
		TotalExpense: 5_000,
		SavingsRate:  0,
	}}

	svc := newAnalyticsSvc(repo)
	result, err := svc.Analytics("user-1", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SavingsRate != 0 {
		t.Errorf("SavingsRate = %v, want 0 when income is zero", result.SavingsRate)
	}
}

func TestAnalytics_EmptyRange(t *testing.T) {
	repo := &mockAnalyticsRepo{result: &domain.AnalyticsResult{}}

	svc := newAnalyticsSvc(repo)
	result, err := svc.Analytics("user-1", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for empty range")
	}
}

func TestAnalytics_DefaultDates(t *testing.T) {
	var capturedFrom, capturedTo time.Time
	repo := &captureAnalyticsRepo{result: &domain.AnalyticsResult{}, captureFrom: &capturedFrom, captureTo: &capturedTo}
	svc := newAnalyticsSvc(repo)

	_, err := svc.Analytics("user-1", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	now := time.Now()
	if capturedFrom.Year() != now.Year() || capturedFrom.Month() != now.Month() || capturedFrom.Day() != 1 {
		t.Errorf("from = %v, want first of current month", capturedFrom)
	}
	if capturedTo.Before(now.Add(-time.Minute)) {
		t.Errorf("to = %v, want approximately now", capturedTo)
	}
}

type captureAnalyticsRepo struct {
	result      *domain.AnalyticsResult
	captureFrom *time.Time
	captureTo   *time.Time
}

func (m *captureAnalyticsRepo) Analytics(_ string, from, to time.Time) (*domain.AnalyticsResult, error) {
	*m.captureFrom = from
	*m.captureTo = to
	return m.result, nil
}
