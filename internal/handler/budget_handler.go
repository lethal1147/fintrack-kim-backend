package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/internal/middleware"
	"github.com/joakim/fintrack-api/pkg/apperror"
	"github.com/joakim/fintrack-api/pkg/response"
)

type BudgetHandler struct {
	svc domain.BudgetServiceInterface
}

func NewBudgetHandler(svc domain.BudgetServiceInterface) *BudgetHandler {
	return &BudgetHandler{svc: svc}
}

type budgetRequest struct {
	Name     string  `json:"name"     binding:"required"`
	Group    string  `json:"group"    binding:"required"`
	Budgeted float64 `json:"budgeted"`
	Color    string  `json:"color"`
}

type budgetResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Group     string  `json:"group"`
	Budgeted  float64 `json:"budgeted"`
	Spent     float64 `json:"spent"`
	Color     string  `json:"color"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func toBudgetResponse(cat *domain.BudgetCategory) budgetResponse {
	return budgetResponse{
		ID:        cat.ID,
		Name:      cat.Name,
		Group:     string(cat.Group),
		Budgeted:  cat.Budgeted,
		Spent:     cat.Spent,
		Color:     cat.Color,
		CreatedAt: cat.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: cat.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// List godoc
// @Summary      List budget categories
// @Description  Returns all budget categories with computed spent for the requested month
// @Tags         budget
// @Produce      json
// @Param        year   query  int  false  "Year (default: current year)"
// @Param        month  query  int  false  "Month 1-12 (default: current month)"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /budget [get]
func (h *BudgetHandler) List(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	now := time.Now()

	year := now.Year()
	if y, err := strconv.Atoi(c.Query("year")); err == nil && y > 0 {
		year = y
	}
	month := int(now.Month())
	if m, err := strconv.Atoi(c.Query("month")); err == nil && m >= 1 && m <= 12 {
		month = m
	}

	cats, err := h.svc.List(userID, year, month)
	if err != nil {
		response.Error(c, err)
		return
	}

	out := make([]budgetResponse, len(cats))
	for i, cat := range cats {
		cat := cat
		out[i] = toBudgetResponse(&cat)
	}
	response.Success(c, out)
}

// Create godoc
// @Summary      Create budget category
// @Description  Creates a new budget category for the authenticated user
// @Tags         budget
// @Accept       json
// @Produce      json
// @Success      201  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /budget [post]
func (h *BudgetHandler) Create(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	var req budgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	item, err := h.svc.Create(userID, domain.CreateBudgetRequest{
		Name:     req.Name,
		Group:    domain.BudgetGroup(req.Group),
		Budgeted: req.Budgeted,
		Color:    req.Color,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": toBudgetResponse(item)})
}

// Update godoc
// @Summary      Update budget category
// @Description  Updates name, group, budgeted amount, or color
// @Tags         budget
// @Accept       json
// @Produce      json
// @Param        id   path  string  true  "Budget category ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      400  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /budget/{id} [put]
func (h *BudgetHandler) Update(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	id := c.Param("id")
	var req budgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.BadRequest(err.Error()))
		return
	}
	item, err := h.svc.Update(userID, id, domain.CreateBudgetRequest{
		Name:     req.Name,
		Group:    domain.BudgetGroup(req.Group),
		Budgeted: req.Budgeted,
		Color:    req.Color,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, toBudgetResponse(item))
}

// Delete godoc
// @Summary      Delete budget category
// @Description  Permanently deletes a budget category
// @Tags         budget
// @Produce      json
// @Param        id   path  string  true  "Budget category ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /budget/{id} [delete]
func (h *BudgetHandler) Delete(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	id := c.Param("id")
	if err := h.svc.Delete(userID, id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{})
}
