package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/internal/middleware"
)

// ── mock budget service ───────────────────────────────────────────────────────

type mockBudgetSvc struct {
	cats      []domain.BudgetCategory
	cat       *domain.BudgetCategory
	listErr   error
	createErr error
	updateErr error
	deleteErr error
}

func (m *mockBudgetSvc) List(_ string, _, _ int) ([]domain.BudgetCategory, error) {
	return m.cats, m.listErr
}
func (m *mockBudgetSvc) Create(_ string, _ domain.CreateBudgetRequest) (*domain.BudgetCategory, error) {
	return m.cat, m.createErr
}
func (m *mockBudgetSvc) Update(_, _ string, _ domain.CreateBudgetRequest) (*domain.BudgetCategory, error) {
	return m.cat, m.updateErr
}
func (m *mockBudgetSvc) Delete(_, _ string) error { return m.deleteErr }

// ── router helper ─────────────────────────────────────────────────────────────

func budgetRouter(svc domain.BudgetServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewBudgetHandler(svc)
	bud := r.Group("/budget", middleware.Auth(testSecret))
	{
		bud.GET("", h.List)
		bud.POST("", h.Create)
		bud.PUT("/:id", h.Update)
		bud.DELETE("/:id", h.Delete)
	}
	return r
}

func sampleBudgetCategory() *domain.BudgetCategory {
	return &domain.BudgetCategory{
		ID:        "bud-1",
		Name:      "Housing",
		Group:     domain.BudgetGroupFixed,
		Budgeted:  1200,
		Spent:     1100,
		Color:     "var(--chart-1)",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestBudget_List_OK(t *testing.T) {
	svc := &mockBudgetSvc{cats: []domain.BudgetCategory{*sampleBudgetCategory()}}
	r := budgetRouter(svc)

	req, _ := http.NewRequest(http.MethodGet, "/budget?year=2026&month=5", nil)
	req.Header.Set("Authorization", "Bearer "+signTestToken("user-1"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestBudget_Create_OK(t *testing.T) {
	svc := &mockBudgetSvc{cat: sampleBudgetCategory()}
	r := budgetRouter(svc)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Housing",
		"group":    "Fixed",
		"budgeted": 1200,
		"color":    "var(--chart-1)",
	})
	req, _ := http.NewRequest(http.MethodPost, "/budget", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+signTestToken("user-1"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
}

func TestBudget_Update_OK(t *testing.T) {
	svc := &mockBudgetSvc{cat: sampleBudgetCategory()}
	r := budgetRouter(svc)

	body, _ := json.Marshal(map[string]interface{}{
		"name":     "Housing",
		"group":    "Fixed",
		"budgeted": 1400,
		"color":    "var(--chart-1)",
	})
	req, _ := http.NewRequest(http.MethodPut, "/budget/bud-1", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+signTestToken("user-1"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestBudget_Delete_OK(t *testing.T) {
	svc := &mockBudgetSvc{}
	r := budgetRouter(svc)

	req, _ := http.NewRequest(http.MethodDelete, "/budget/bud-1", nil)
	req.Header.Set("Authorization", "Bearer "+signTestToken("user-1"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestBudget_Unauthorized(t *testing.T) {
	svc := &mockBudgetSvc{}
	r := budgetRouter(svc)

	req, _ := http.NewRequest(http.MethodGet, "/budget", nil)
	// no auth header
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}
