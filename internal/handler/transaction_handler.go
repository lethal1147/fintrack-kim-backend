package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/internal/middleware"
	"github.com/joakim/fintrack-api/pkg/apperror"
	"github.com/joakim/fintrack-api/pkg/response"
)

type TransactionHandler struct {
	svc domain.TransactionServiceInterface
}

func NewTransactionHandler(svc domain.TransactionServiceInterface) *TransactionHandler {
	return &TransactionHandler{svc: svc}
}

type transactionRequest struct {
	Merchant string  `json:"merchant" binding:"required"`
	Category string  `json:"category" binding:"required"`
	Note     string  `json:"note"`
	Date     string  `json:"date"     binding:"required"`
	Amount   float64 `json:"amount"   binding:"required"`
	Type     string  `json:"type"     binding:"required"`
}

type transactionResponse struct {
	ID        string  `json:"id"`
	Merchant  string  `json:"merchant"`
	Category  string  `json:"category"`
	Note      string  `json:"note"`
	Date      string  `json:"date"`
	Amount    float64 `json:"amount"`
	Type      string  `json:"type"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func toTransactionResponse(tx *domain.Transaction) transactionResponse {
	return transactionResponse{
		ID:        tx.ID,
		Merchant:  tx.Merchant,
		Category:  tx.Category,
		Note:      tx.Note,
		Date:      tx.Date.Format("2006-01-02"),
		Amount:    tx.Amount,
		Type:      string(tx.Type),
		CreatedAt: tx.CreatedAt.Format(time.RFC3339),
		UpdatedAt: tx.UpdatedAt.Format(time.RFC3339),
	}
}

// Create godoc
// @Summary      Create a transaction
// @Description  Saves a new income or expense transaction for the authenticated user.
// @Tags         transactions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      transactionRequest  true  "Transaction payload"
// @Success      201   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /transactions [post]
func (h *TransactionHandler) Create(c *gin.Context) {
	var req transactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	userID := c.GetString(middleware.ContextUserID)
	tx, err := h.svc.Create(userID, domain.CreateTransactionRequest{
		Merchant: req.Merchant,
		Category: req.Category,
		Note:     req.Note,
		Date:     req.Date,
		Amount:   req.Amount,
		Type:     domain.TransactionType(req.Type),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, toTransactionResponse(tx))
}

// List godoc
// @Summary      List transactions
// @Description  Returns a paginated, filterable list of transactions for the authenticated user.
// @Tags         transactions
// @Produce      json
// @Security     BearerAuth
// @Param        type      query  string  false  "income or expense"
// @Param        category  query  string  false  "Exact category name"
// @Param        search    query  string  false  "Search merchant or note"
// @Param        from      query  string  false  "Start date YYYY-MM-DD"
// @Param        to        query  string  false  "End date YYYY-MM-DD"
// @Param        page      query  int     false  "Page number (default 1)"
// @Param        limit     query  int     false  "Items per page (default 8, max 100)"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /transactions [get]
func (h *TransactionHandler) List(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "8"))

	result, err := h.svc.List(userID, domain.TransactionFilter{
		Type:     c.Query("type"),
		Category: c.Query("category"),
		Search:   c.Query("search"),
		From:     c.Query("from"),
		To:       c.Query("to"),
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		response.Error(c, err)
		return
	}

	items := make([]transactionResponse, len(result.Transactions))
	for i, tx := range result.Transactions {
		items[i] = toTransactionResponse(tx)
	}
	response.Success(c, gin.H{
		"transactions": items,
		"total":        result.Total,
		"page":         result.Page,
		"limit":        result.Limit,
		"pages":        result.Pages,
	})
}

// Get godoc
// @Summary      Get a transaction
// @Description  Returns a single transaction by ID for the authenticated user.
// @Tags         transactions
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Transaction ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /transactions/{id} [get]
func (h *TransactionHandler) Get(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	tx, err := h.svc.Get(c.Param("id"), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, toTransactionResponse(tx))
}

// Update godoc
// @Summary      Update a transaction
// @Description  Replaces all fields of an existing transaction.
// @Tags         transactions
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  string              true  "Transaction ID"
// @Param        body  body  transactionRequest  true  "Updated transaction"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Router       /transactions/{id} [put]
func (h *TransactionHandler) Update(c *gin.Context) {
	var req transactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	userID := c.GetString(middleware.ContextUserID)
	tx, err := h.svc.Update(c.Param("id"), userID, domain.UpdateTransactionRequest{
		Merchant: req.Merchant,
		Category: req.Category,
		Note:     req.Note,
		Date:     req.Date,
		Amount:   req.Amount,
		Type:     domain.TransactionType(req.Type),
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, toTransactionResponse(tx))
}

// Delete godoc
// @Summary      Soft-delete a transaction
// @Description  Marks a transaction as deleted (sets deleted_at). The row is preserved.
// @Tags         transactions
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "Transaction ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /transactions/{id} [delete]
func (h *TransactionHandler) Delete(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	if err := h.svc.Delete(c.Param("id"), userID); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// Summary godoc
// @Summary      Transaction summary
// @Description  Returns total income/expense, breakdown by category, and monthly aggregation.
// @Tags         transactions
// @Produce      json
// @Security     BearerAuth
// @Param        from  query  string  false  "Start date YYYY-MM-DD (default: first of current month)"
// @Param        to    query  string  false  "End date YYYY-MM-DD (default: today)"
// @Success      200   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /transactions/summary [get]
func (h *TransactionHandler) Summary(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)

	var from, to time.Time
	if s := c.Query("from"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			from = t
		}
	}
	if s := c.Query("to"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			to = t
		}
	}

	result, err := h.svc.Summary(userID, from, to)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{
		"total_income":  result.TotalIncome,
		"total_expense": result.TotalExpense,
		"net":           result.Net,
		"by_category":   result.ByCategory,
		"by_month":      result.ByMonth,
	})
}
