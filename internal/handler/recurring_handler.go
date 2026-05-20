package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/internal/middleware"
	"github.com/joakim/fintrack-api/pkg/response"
)

type RecurringHandler struct {
	svc domain.RecurringServiceInterface
}

func NewRecurringHandler(svc domain.RecurringServiceInterface) *RecurringHandler {
	return &RecurringHandler{svc: svc}
}

type recurringRequest struct {
	Name      string  `json:"name"      binding:"required"`
	Category  string  `json:"category"  binding:"required"`
	Amount    float64 `json:"amount"    binding:"required,gt=0"`
	Frequency string  `json:"frequency" binding:"required"`
	Kind      string  `json:"kind"      binding:"required"`
	NextDue   string  `json:"next_due"  binding:"required"`
	Color     string  `json:"color"`
}

type recurringResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Category  string  `json:"category"`
	Amount    float64 `json:"amount"`
	Frequency string  `json:"frequency"`
	Kind      string  `json:"kind"`
	Status    string  `json:"status"`
	NextDue   string  `json:"next_due"`
	Color     string  `json:"color"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

func toRecurringResponse(item *domain.RecurringItem) recurringResponse {
	return recurringResponse{
		ID:        item.ID,
		Name:      item.Name,
		Category:  item.Category,
		Amount:    item.Amount,
		Frequency: string(item.Frequency),
		Kind:      string(item.Kind),
		Status:    string(item.Status),
		NextDue:   item.NextDue.Format("2006-01-02"),
		Color:     item.Color,
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
		UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
	}
}

// List godoc
// @Summary      List recurring items
// @Tags         recurring
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /recurring [get]
func (h *RecurringHandler) List(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	items, err := h.svc.List(userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	resp := make([]recurringResponse, len(items))
	for i, item := range items {
		cp := item
		resp[i] = toRecurringResponse(&cp)
	}
	response.Success(c, resp)
}

// Create godoc
// @Summary      Create recurring item
// @Tags         recurring
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      recurringRequest  true  "Recurring item"
// @Success      201   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Router       /recurring [post]
func (h *RecurringHandler) Create(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	var req recurringRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	nextDue, err := time.Parse("2006-01-02", req.NextDue)
	if err != nil {
		response.Error(c, err)
		return
	}
	item, err := h.svc.Create(userID, domain.CreateRecurringRequest{
		Name:      req.Name,
		Category:  req.Category,
		Amount:    req.Amount,
		Frequency: domain.RecurringFrequency(req.Frequency),
		Kind:      domain.RecurringKind(req.Kind),
		NextDue:   nextDue,
		Color:     req.Color,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, toRecurringResponse(item))
}

// Update godoc
// @Summary      Update recurring item
// @Tags         recurring
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id    path      string            true  "Item ID"
// @Param        body  body      recurringRequest  true  "Recurring item"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  map[string]interface{}
// @Failure      401   {object}  map[string]interface{}
// @Failure      404   {object}  map[string]interface{}
// @Router       /recurring/{id} [put]
func (h *RecurringHandler) Update(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	id := c.Param("id")
	var req recurringRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	nextDue, err := time.Parse("2006-01-02", req.NextDue)
	if err != nil {
		response.Error(c, err)
		return
	}
	item, err := h.svc.Update(userID, id, domain.CreateRecurringRequest{
		Name:      req.Name,
		Category:  req.Category,
		Amount:    req.Amount,
		Frequency: domain.RecurringFrequency(req.Frequency),
		Kind:      domain.RecurringKind(req.Kind),
		NextDue:   nextDue,
		Color:     req.Color,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, toRecurringResponse(item))
}

// ToggleStatus godoc
// @Summary      Toggle recurring item status
// @Tags         recurring
// @Security     BearerAuth
// @Produce      json
// @Param        id  path      string  true  "Item ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /recurring/{id}/status [patch]
func (h *RecurringHandler) ToggleStatus(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	id := c.Param("id")
	item, err := h.svc.SetStatus(userID, id, "")
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, toRecurringResponse(item))
}

// Delete godoc
// @Summary      Delete recurring item
// @Tags         recurring
// @Security     BearerAuth
// @Produce      json
// @Param        id  path      string  true  "Item ID"
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Failure      404  {object}  map[string]interface{}
// @Router       /recurring/{id} [delete]
func (h *RecurringHandler) Delete(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	id := c.Param("id")
	if err := h.svc.Delete(userID, id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// Process godoc
// @Summary      Generate due recurring transactions
// @Tags         recurring
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  map[string]interface{}
// @Failure      401  {object}  map[string]interface{}
// @Router       /recurring/process [post]
func (h *RecurringHandler) Process(c *gin.Context) {
	userID := c.GetString(middleware.ContextUserID)
	n, err := h.svc.Process(userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"generated": n})
}
