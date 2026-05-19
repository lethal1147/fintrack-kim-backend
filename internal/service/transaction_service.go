package service

import (
	"math"
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

const (
	defaultPage  = 1
	defaultLimit = 8
	maxLimit     = 100
)

type TransactionService struct {
	repo domain.TransactionRepository
}

func NewTransactionService(repo domain.TransactionRepository) *TransactionService {
	return &TransactionService{repo: repo}
}

func (s *TransactionService) Create(userID string, req domain.CreateTransactionRequest) (*domain.Transaction, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	date, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}
	tx := &domain.Transaction{
		UserID:   userID,
		Merchant: req.Merchant,
		Category: req.Category,
		Note:     req.Note,
		Date:     date,
		Amount:   req.Amount,
		Type:     req.Type,
	}
	if err := s.repo.Create(tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *TransactionService) Get(id, userID string) (*domain.Transaction, error) {
	return s.repo.GetByID(id, userID)
}

func (s *TransactionService) Update(id, userID string, req domain.UpdateTransactionRequest) (*domain.Transaction, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}
	date, err := parseDate(req.Date)
	if err != nil {
		return nil, err
	}
	tx, err := s.repo.GetByID(id, userID)
	if err != nil {
		return nil, err
	}
	tx.Merchant = req.Merchant
	tx.Category = req.Category
	tx.Note = req.Note
	tx.Date = date
	tx.Amount = req.Amount
	tx.Type = req.Type
	tx.UpdatedAt = time.Now()
	if err := s.repo.Update(tx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (s *TransactionService) Delete(id, userID string) error {
	return s.repo.SoftDelete(id, userID)
}

func (s *TransactionService) List(userID string, f domain.TransactionFilter) (*domain.TransactionListResult, error) {
	if f.Page < 1 {
		f.Page = defaultPage
	}
	if f.Limit < 1 {
		f.Limit = defaultLimit
	}
	if f.Limit > maxLimit {
		f.Limit = maxLimit
	}
	txs, total, err := s.repo.List(userID, f)
	if err != nil {
		return nil, err
	}
	pages := int(math.Ceil(float64(total) / float64(f.Limit)))
	if pages < 1 {
		pages = 1
	}
	return &domain.TransactionListResult{
		Transactions: txs,
		Total:        total,
		Page:         f.Page,
		Limit:        f.Limit,
		Pages:        pages,
	}, nil
}

func (s *TransactionService) Summary(userID string, from, to time.Time) (*domain.TransactionSummaryResult, error) {
	if from.IsZero() {
		now := time.Now()
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}
	if to.IsZero() {
		to = time.Now()
	}
	result, err := s.repo.Summary(userID, from, to)
	if err != nil {
		return nil, err
	}
	for _, c := range result.ByCategory {
		if c.Type == domain.TransactionIncome {
			result.TotalIncome += c.Total
		} else {
			result.TotalExpense += c.Total
		}
	}
	result.Net = result.TotalIncome - result.TotalExpense
	return result, nil
}

// ── validation helpers ─────────────────────────────────────────────────────

func validateRequest(req domain.CreateTransactionRequest) error {
	if req.Merchant == "" {
		return apperror.BadRequest("merchant is required")
	}
	if req.Type != domain.TransactionIncome && req.Type != domain.TransactionExpense {
		return apperror.BadRequest("type must be 'income' or 'expense'")
	}
	if !domain.ValidCategories[req.Category] {
		return apperror.BadRequest("invalid category: " + req.Category)
	}
	if req.Amount <= 0 {
		return apperror.BadRequest("amount must be greater than 0")
	}
	return nil
}

func parseDate(s string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, apperror.BadRequest("date must be YYYY-MM-DD format")
	}
	return t, nil
}
