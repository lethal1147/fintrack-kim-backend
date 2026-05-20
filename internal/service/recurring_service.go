package service

import (
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

type RecurringService struct {
	repo  domain.RecurringRepository
	txRepo domain.TransactionRepository
}

func NewRecurringService(repo domain.RecurringRepository, txRepo domain.TransactionRepository) *RecurringService {
	return &RecurringService{repo: repo, txRepo: txRepo}
}

func (s *RecurringService) List(userID string) ([]domain.RecurringItem, error) {
	return s.repo.List(userID)
}

func (s *RecurringService) Create(userID string, req domain.CreateRecurringRequest) (*domain.RecurringItem, error) {
	if err := validateRecurringRequest(req); err != nil {
		return nil, err
	}
	item := &domain.RecurringItem{
		UserID:    userID,
		Name:      req.Name,
		Category:  req.Category,
		Amount:    req.Amount,
		Frequency: req.Frequency,
		Kind:      req.Kind,
		Status:    domain.StatusActive,
		Color:     req.Color,
		NextDue:   req.NextDue,
	}
	if item.Color == "" {
		item.Color = "#0ea5e9"
	}
	if err := s.repo.Create(userID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *RecurringService) Update(userID, id string, req domain.CreateRecurringRequest) (*domain.RecurringItem, error) {
	if err := validateRecurringRequest(req); err != nil {
		return nil, err
	}
	item, err := s.repo.GetByID(userID, id)
	if err != nil {
		return nil, err
	}
	item.Name = req.Name
	item.Category = req.Category
	item.Amount = req.Amount
	item.Frequency = req.Frequency
	item.Kind = req.Kind
	item.Color = req.Color
	item.NextDue = req.NextDue
	if err := s.repo.Update(userID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *RecurringService) SetStatus(userID, id string, status domain.RecurringStatus) (*domain.RecurringItem, error) {
	item, err := s.repo.GetByID(userID, id)
	if err != nil {
		return nil, err
	}
	newStatus := domain.StatusActive
	if item.Status == domain.StatusActive {
		newStatus = domain.StatusPaused
	}
	if err := s.repo.SetStatus(userID, id, newStatus); err != nil {
		return nil, err
	}
	item.Status = newStatus
	return item, nil
}

func (s *RecurringService) Delete(userID, id string) error {
	return s.repo.Delete(userID, id)
}

func (s *RecurringService) Process(userID string) (int, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	items, err := s.repo.ListDue(userID, today)
	if err != nil {
		return 0, err
	}

	generated := 0
	for _, item := range items {
		nextDue := item.NextDue.UTC().Truncate(24 * time.Hour)
		for !nextDue.After(today) {
			txType := domain.TransactionExpense
			if item.Kind == domain.KindIncome {
				txType = domain.TransactionIncome
			}
			tx := &domain.Transaction{
				UserID:   userID,
				Merchant: item.Name,
				Category: item.Category,
				Note:     "Auto-generated",
				Date:     nextDue,
				Amount:   item.Amount,
				Type:     txType,
			}
			if err := s.txRepo.Create(tx); err != nil {
				return generated, err
			}
			generated++
			nextDue = advanceNextDue(nextDue, item.Frequency)
		}
		if err := s.repo.UpdateNextDue(item.ID, nextDue); err != nil {
			return generated, err
		}
	}
	return generated, nil
}

func advanceNextDue(d time.Time, freq domain.RecurringFrequency) time.Time {
	var next time.Time
	switch freq {
	case domain.FrequencyWeekly:
		return d.AddDate(0, 0, 7)
	case domain.FrequencyMonthly:
		next = d.AddDate(0, 1, 0)
	case domain.FrequencyAnnual:
		next = d.AddDate(1, 0, 0)
	default:
		return d.AddDate(0, 1, 0)
	}
	return adjustWeekend(next)
}

func adjustWeekend(d time.Time) time.Time {
	switch d.Weekday() {
	case time.Saturday:
		return d.AddDate(0, 0, -1)
	case time.Sunday:
		return d.AddDate(0, 0, -2)
	}
	return d
}

func validateRecurringRequest(req domain.CreateRecurringRequest) error {
	if req.Name == "" {
		return apperror.BadRequest("name is required")
	}
	if req.Amount <= 0 {
		return apperror.BadRequest("amount must be greater than zero")
	}
	if req.Frequency != domain.FrequencyWeekly && req.Frequency != domain.FrequencyMonthly && req.Frequency != domain.FrequencyAnnual {
		return apperror.BadRequest("frequency must be weekly, monthly, or annual")
	}
	if req.Kind != domain.KindExpense && req.Kind != domain.KindIncome {
		return apperror.BadRequest("kind must be expense or income")
	}
	if req.NextDue.IsZero() {
		return apperror.BadRequest("next_due is required")
	}
	return nil
}
