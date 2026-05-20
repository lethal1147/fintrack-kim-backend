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

// ── mock recurring service ─────────────────────────────────────────────────

type mockRecurringSvc struct {
	items       []domain.RecurringItem
	item        *domain.RecurringItem
	generated   int
	listErr     error
	createErr   error
	updateErr   error
	statusErr   error
	deleteErr   error
	processErr  error
}

func (m *mockRecurringSvc) List(_ string) ([]domain.RecurringItem, error) {
	return m.items, m.listErr
}
func (m *mockRecurringSvc) Create(_ string, _ domain.CreateRecurringRequest) (*domain.RecurringItem, error) {
	return m.item, m.createErr
}
func (m *mockRecurringSvc) Update(_, _ string, _ domain.CreateRecurringRequest) (*domain.RecurringItem, error) {
	return m.item, m.updateErr
}
func (m *mockRecurringSvc) SetStatus(_, _ string, _ domain.RecurringStatus) (*domain.RecurringItem, error) {
	return m.item, m.statusErr
}
func (m *mockRecurringSvc) Delete(_, _ string) error { return m.deleteErr }
func (m *mockRecurringSvc) Process(_ string) (int, error) {
	return m.generated, m.processErr
}

// ── router helper ──────────────────────────────────────────────────────────

func recurringRouter(svc domain.RecurringServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewRecurringHandler(svc)
	rec := r.Group("/recurring", middleware.Auth(testSecret))
	{
		rec.GET("", h.List)
		rec.POST("", h.Create)
		rec.POST("/process", h.Process)
		rec.PUT("/:id", h.Update)
		rec.PATCH("/:id/status", h.ToggleStatus)
		rec.DELETE("/:id", h.Delete)
	}
	return r
}

func sampleRecurringItem() *domain.RecurringItem {
	return &domain.RecurringItem{
		ID:        "rec-1",
		Name:      "Netflix",
		Category:  "Subscriptions",
		Amount:    15.99,
		Frequency: domain.FrequencyMonthly,
		Kind:      domain.KindExpense,
		Status:    domain.StatusActive,
		Color:     "#ef4444",
		NextDue:   time.Date(2026, 6, 8, 0, 0, 0, 0, time.UTC),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// ── tests ──────────────────────────────────────────────────────────────────

func TestRecurringHandler_List_OK(t *testing.T) {
	svc := &mockRecurringSvc{items: []domain.RecurringItem{*sampleRecurringItem()}}
	r := recurringRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/recurring", nil)
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
	data, ok := body["data"].([]interface{})
	if !ok {
		t.Fatalf("data not array: %v", body)
	}
	if len(data) != 1 {
		t.Errorf("len(data) = %d, want 1", len(data))
	}
}

func TestRecurringHandler_Create_OK(t *testing.T) {
	svc := &mockRecurringSvc{item: sampleRecurringItem()}
	r := recurringRouter(svc)
	body := `{"name":"Netflix","category":"Subscriptions","amount":15.99,"frequency":"monthly","kind":"expense","next_due":"2026-06-08","color":"#ef4444"}`
	req := httptest.NewRequest(http.MethodPost, "/recurring", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+signTestToken("user-1"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", w.Code, w.Body.String())
	}
}

func TestRecurringHandler_Process_OK(t *testing.T) {
	svc := &mockRecurringSvc{generated: 3}
	r := recurringRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/recurring/process", nil)
	req.Header.Set("Authorization", "Bearer "+signTestToken("user-1"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	data := resp["data"].(map[string]interface{})
	if data["generated"].(float64) != 3 {
		t.Errorf("generated = %v, want 3", data["generated"])
	}
}

func TestRecurringHandler_ToggleStatus_OK(t *testing.T) {
	item := sampleRecurringItem()
	item.Status = domain.StatusPaused
	svc := &mockRecurringSvc{item: item}
	r := recurringRouter(svc)
	req := httptest.NewRequest(http.MethodPatch, "/recurring/rec-1/status", nil)
	req.Header.Set("Authorization", "Bearer "+signTestToken("user-1"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

func TestRecurringHandler_Delete_OK(t *testing.T) {
	svc := &mockRecurringSvc{}
	r := recurringRouter(svc)
	req := httptest.NewRequest(http.MethodDelete, "/recurring/rec-1", nil)
	req.Header.Set("Authorization", "Bearer "+signTestToken("user-1"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

func TestRecurringHandler_Update_OK(t *testing.T) {
	svc := &mockRecurringSvc{item: sampleRecurringItem()}
	r := recurringRouter(svc)
	body := `{"name":"Netflix HD","category":"Subscriptions","amount":19.99,"frequency":"monthly","kind":"expense","next_due":"2026-06-08","color":"#ef4444"}`
	req := httptest.NewRequest(http.MethodPut, "/recurring/rec-1", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+signTestToken("user-1"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

func TestRecurringHandler_Unauthorized(t *testing.T) {
	svc := &mockRecurringSvc{}
	r := recurringRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/recurring", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
