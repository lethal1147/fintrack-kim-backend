package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/internal/middleware"
	"github.com/joakim/fintrack-api/pkg/response"
)

type AnalyticsHandler struct {
	svc domain.AnalyticsServiceInterface
}

func NewAnalyticsHandler(svc domain.AnalyticsServiceInterface) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

type categoryStatResponse struct {
	Category string  `json:"category"`
	Type     string  `json:"type"`
	Total    float64 `json:"total"`
	Count    int     `json:"count"`
	Pct      float64 `json:"pct"`
}

type monthStatResponse struct {
	Month   string  `json:"month"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
	Net     float64 `json:"net"`
}

type merchantStatResponse struct {
	Merchant string  `json:"merchant"`
	Total    float64 `json:"total"`
	Count    int     `json:"count"`
}

// Analytics godoc
// @Summary      Transaction analytics
// @Description  Returns enriched financial analytics for the authenticated user over a date range.
// @Tags         transactions
// @Produce      json
// @Security     BearerAuth
// @Param        from  query  string  false  "Start date YYYY-MM-DD (default: Jan 1 of current year)"
// @Param        to    query  string  false  "End date YYYY-MM-DD (default: today)"
// @Success      200   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /transactions/analytics [get]
func (h *AnalyticsHandler) Analytics(c *gin.Context) {
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

	result, err := h.svc.Analytics(userID, from, to)
	if err != nil {
		response.Error(c, err)
		return
	}

	cats := make([]categoryStatResponse, len(result.ByCategory))
	for i, c := range result.ByCategory {
		cats[i] = categoryStatResponse{
			Category: c.Category,
			Type:     string(c.Type),
			Total:    c.Total,
			Count:    c.Count,
			Pct:      c.Pct,
		}
	}

	months := make([]monthStatResponse, len(result.ByMonth))
	for i, m := range result.ByMonth {
		months[i] = monthStatResponse{
			Month:   m.Month,
			Income:  m.Income,
			Expense: m.Expense,
			Net:     m.Net,
		}
	}

	merchants := make([]merchantStatResponse, len(result.TopMerchants))
	for i, m := range result.TopMerchants {
		merchants[i] = merchantStatResponse{
			Merchant: m.Merchant,
			Total:    m.Total,
			Count:    m.Count,
		}
	}

	response.Success(c, gin.H{
		"total_income":  result.TotalIncome,
		"total_expense": result.TotalExpense,
		"net":           result.Net,
		"tx_count":      result.TxCount,
		"savings_rate":  result.SavingsRate,
		"by_category":   cats,
		"by_month":      months,
		"top_merchants": merchants,
	})
}
