package service

import (
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
)

type AnalyticsService struct {
	repo domain.AnalyticsRepository
}

func NewAnalyticsService(repo domain.AnalyticsRepository) *AnalyticsService {
	return &AnalyticsService{repo: repo}
}

func (s *AnalyticsService) Analytics(userID string, from, to time.Time) (*domain.AnalyticsResult, error) {
	if from.IsZero() {
		now := time.Now()
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}
	if to.IsZero() {
		to = time.Now()
	}
	return s.repo.Analytics(userID, from, to)
}
