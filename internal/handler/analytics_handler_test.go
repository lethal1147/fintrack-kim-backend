package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/internal/middleware"
)

// ── mock analytics service ─────────────────────────────────────────────────

type mockAnalyticsSvc struct {
	result *domain.AnalyticsResult
	err    error
}

func (m *mockAnalyticsSvc) Analytics(_ string, _, _ time.Time) (*domain.AnalyticsResult, error) {
	return m.result, m.err
}

// ── router helper ──────────────────────────────────────────────────────────

func analyticsRouter(svc domain.AnalyticsServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAnalyticsHandler(svc)
	r.GET("/transactions/analytics", middleware.Auth(testSecret), h.Analytics)
	return r
}

// ── tests ──────────────────────────────────────────────────────────────────

func TestAnalyticsHandler_OK(t *testing.T) {
	svc := &mockAnalyticsSvc{result: &domain.AnalyticsResult{
		TotalIncome:  100_000,
		TotalExpense: 70_000,
		Net:          30_000,
		TxCount:      42,
		SavingsRate:  30.0,
		ByCategory:   []domain.CategoryStat{{Category: "Salary", Type: domain.TransactionIncome, Total: 100_000, Count: 1, Pct: 100}},
		ByMonth:      []domain.MonthStat{{Month: "2026-01", Income: 100_000, Expense: 70_000, Net: 30_000}},
		TopMerchants: []domain.MerchantStat{{Merchant: "Starbucks", Total: 5_000, Count: 20}},
	}}

	r := analyticsRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/transactions/analytics", nil)
	req.Header.Set("Authorization", "Bearer "+signTestToken("user-1"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data field missing or wrong type: %v", body)
	}
	if data["total_income"] != 100_000.0 {
		t.Errorf("total_income = %v, want 100000", data["total_income"])
	}
	if data["savings_rate"] != 30.0 {
		t.Errorf("savings_rate = %v, want 30.0", data["savings_rate"])
	}
}

func TestAnalyticsHandler_Unauthorized(t *testing.T) {
	svc := &mockAnalyticsSvc{result: &domain.AnalyticsResult{}}
	r := analyticsRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/transactions/analytics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
