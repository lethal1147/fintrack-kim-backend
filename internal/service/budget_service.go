package service

import (
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

type BudgetService struct {
	repo   domain.BudgetRepository
	txRepo domain.TransactionRepository
}

func NewBudgetService(repo domain.BudgetRepository, txRepo domain.TransactionRepository) *BudgetService {
	return &BudgetService{repo: repo, txRepo: txRepo}
}

func (s *BudgetService) List(userID string, year, month int) ([]domain.BudgetCategory, error) {
	cats, err := s.repo.List(userID)
	if err != nil {
		return nil, err
	}

	from := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0).Add(-time.Nanosecond)

	summary, err := s.txRepo.Summary(userID, from, to)
	if err != nil {
		return nil, err
	}

	spentByCategory := make(map[string]float64)
	for _, cs := range summary.ByCategory {
		if cs.Type == domain.TransactionExpense {
			spentByCategory[cs.Category] += cs.Total
		}
	}

	for i := range cats {
		cats[i].Spent = spentByCategory[cats[i].Name]
	}
	return cats, nil
}

func (s *BudgetService) Create(userID string, req domain.CreateBudgetRequest) (*domain.BudgetCategory, error) {
	if err := validateBudgetRequest(req); err != nil {
		return nil, err
	}
	item := &domain.BudgetCategory{
		UserID:   userID,
		Name:     req.Name,
		Group:    req.Group,
		Budgeted: req.Budgeted,
		Color:    req.Color,
	}
	if item.Color == "" {
		item.Color = "var(--chart-1)"
	}
	if err := s.repo.Create(userID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *BudgetService) Update(userID, id string, req domain.CreateBudgetRequest) (*domain.BudgetCategory, error) {
	if err := validateBudgetRequest(req); err != nil {
		return nil, err
	}
	item, err := s.repo.GetByID(userID, id)
	if err != nil {
		return nil, err
	}
	item.Name = req.Name
	item.Group = req.Group
	item.Budgeted = req.Budgeted
	item.Color = req.Color
	if err := s.repo.Update(userID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *BudgetService) Delete(userID, id string) error {
	return s.repo.Delete(userID, id)
}

func validateBudgetRequest(req domain.CreateBudgetRequest) error {
	if req.Name == "" {
		return apperror.BadRequest("name is required")
	}
	if req.Budgeted < 0 {
		return apperror.BadRequest("budgeted amount cannot be negative")
	}
	if req.Group != domain.BudgetGroupFixed &&
		req.Group != domain.BudgetGroupFlexible &&
		req.Group != domain.BudgetGroupNonMonthly {
		return apperror.BadRequest("group must be Fixed, Flexible, or Non-Monthly")
	}
	return nil
}
