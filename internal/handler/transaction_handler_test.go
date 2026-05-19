package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/internal/middleware"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

// ── mock service ───────────────────────────────────────────────────────────

type mockTxSvc struct {
	createResp  *domain.Transaction
	createErr   error
	getResp     *domain.Transaction
	getErr      error
	updateResp  *domain.Transaction
	updateErr   error
	deleteErr   error
	listResp    *domain.TransactionListResult
	listErr     error
	summaryResp *domain.TransactionSummaryResult
	summaryErr  error
}

func (m *mockTxSvc) Create(_ string, _ domain.CreateTransactionRequest) (*domain.Transaction, error) {
	return m.createResp, m.createErr
}
func (m *mockTxSvc) Get(_, _ string) (*domain.Transaction, error) { return m.getResp, m.getErr }
func (m *mockTxSvc) Update(_, _ string, _ domain.UpdateTransactionRequest) (*domain.Transaction, error) {
	return m.updateResp, m.updateErr
}
func (m *mockTxSvc) Delete(_, _ string) error { return m.deleteErr }
func (m *mockTxSvc) List(_ string, _ domain.TransactionFilter) (*domain.TransactionListResult, error) {
	return m.listResp, m.listErr
}
func (m *mockTxSvc) Summary(_ string, _, _ time.Time) (*domain.TransactionSummaryResult, error) {
	return m.summaryResp, m.summaryErr
}

// ── router helper ──────────────────────────────────────────────────────────

func txRouter(svc domain.TransactionServiceInterface) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewTransactionHandler(svc)
	tx := r.Group("/transactions")
	tx.Use(middleware.Auth(testSecret))
	{
		tx.GET("", h.List)
		tx.POST("", h.Create)
		tx.GET("/summary", h.Summary)
		tx.GET("/:id", h.Get)
		tx.PUT("/:id", h.Update)
		tx.DELETE("/:id", h.Delete)
	}
	return r
}

func txToken() string { return signTestToken("user-1") }

func doTxPost(r *gin.Engine, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+txToken())
	r.ServeHTTP(w, req)
	return w
}

func doTxGet(r *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+txToken())
	r.ServeHTTP(w, req)
	return w
}

func doTxPut(r *gin.Engine, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+txToken())
	r.ServeHTTP(w, req)
	return w
}

func doTxDelete(r *gin.Engine, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodDelete, path, nil)
	req.Header.Set("Authorization", "Bearer "+txToken())
	r.ServeHTTP(w, req)
	return w
}

func sampleTx() *domain.Transaction {
	return &domain.Transaction{
		ID:       "tx-1",
		UserID:   "user-1",
		Merchant: "SCB Bank",
		Category: "Salary",
		Note:     "March",
		Date:     time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC),
		Amount:   42800,
		Type:     domain.TransactionIncome,
	}
}

// ── Create ─────────────────────────────────────────────────────────────────

func TestCreateTransaction_Valid(t *testing.T) {
	svc := &mockTxSvc{createResp: sampleTx()}
	r := txRouter(svc)

	body := `{"merchant":"SCB Bank","category":"Salary","date":"2026-03-25","amount":42800,"type":"income"}`
	w := doTxPost(r, "/transactions", body)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["id"] != "tx-1" {
		t.Errorf("id = %v, want tx-1", data["id"])
	}
}

func TestCreateTransaction_ServiceError(t *testing.T) {
	svc := &mockTxSvc{createErr: apperror.BadRequest("invalid category")}
	r := txRouter(svc)

	body := `{"merchant":"Test","category":"Bad","date":"2026-03-25","amount":100,"type":"expense"}`
	w := doTxPost(r, "/transactions", body)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCreateTransaction_MissingBody(t *testing.T) {
	svc := &mockTxSvc{}
	r := txRouter(svc)

	w := doTxPost(r, "/transactions", "")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCreateTransaction_Unauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := txRouter(&mockTxSvc{})
	req, _ := http.NewRequest(http.MethodPost, "/transactions", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

// ── List ───────────────────────────────────────────────────────────────────

func TestListTransactions_Valid(t *testing.T) {
	svc := &mockTxSvc{listResp: &domain.TransactionListResult{
		Transactions: []*domain.Transaction{sampleTx()},
		Total:        1, Page: 1, Limit: 8, Pages: 1,
	}}
	r := txRouter(svc)

	w := doTxGet(r, "/transactions")
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["total"] == nil {
		t.Error("expected total field in response")
	}
}

func TestListTransactions_WithFilters(t *testing.T) {
	svc := &mockTxSvc{listResp: &domain.TransactionListResult{Transactions: nil, Total: 0, Page: 1, Limit: 8, Pages: 1}}
	r := txRouter(svc)

	w := doTxGet(r, "/transactions?type=expense&category=Food+%26+Dining&page=2")
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// ── Get ────────────────────────────────────────────────────────────────────

func TestGetTransaction_Valid(t *testing.T) {
	svc := &mockTxSvc{getResp: sampleTx()}
	r := txRouter(svc)

	w := doTxGet(r, "/transactions/tx-1")
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestGetTransaction_NotFound(t *testing.T) {
	svc := &mockTxSvc{getErr: apperror.NotFound("transaction not found")}
	r := txRouter(svc)

	w := doTxGet(r, "/transactions/no-such-id")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ── Update ─────────────────────────────────────────────────────────────────

func TestUpdateTransaction_Valid(t *testing.T) {
	updated := sampleTx()
	updated.Merchant = "Updated"
	svc := &mockTxSvc{updateResp: updated}
	r := txRouter(svc)

	body := `{"merchant":"Updated","category":"Salary","date":"2026-03-25","amount":999,"type":"income"}`
	w := doTxPut(r, "/transactions/tx-1", body)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestUpdateTransaction_NotFound(t *testing.T) {
	svc := &mockTxSvc{updateErr: apperror.NotFound("transaction not found")}
	r := txRouter(svc)

	body := `{"merchant":"X","category":"Salary","date":"2026-03-25","amount":1,"type":"income"}`
	w := doTxPut(r, fmt.Sprintf("/transactions/%s", "no-such-id"), body)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ── Delete ─────────────────────────────────────────────────────────────────

func TestDeleteTransaction_Valid(t *testing.T) {
	svc := &mockTxSvc{}
	r := txRouter(svc)

	w := doTxDelete(r, "/transactions/tx-1")
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}

func TestDeleteTransaction_NotFound(t *testing.T) {
	svc := &mockTxSvc{deleteErr: apperror.NotFound("transaction not found")}
	r := txRouter(svc)

	w := doTxDelete(r, "/transactions/ghost-id")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// ── Summary ────────────────────────────────────────────────────────────────

func TestSummaryTransaction_Valid(t *testing.T) {
	svc := &mockTxSvc{summaryResp: &domain.TransactionSummaryResult{
		TotalIncome: 1000, TotalExpense: 500, Net: 500,
	}}
	r := txRouter(svc)

	w := doTxGet(r, "/transactions/summary")
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if data["total_income"] == nil {
		t.Error("expected total_income in summary response")
	}
}
