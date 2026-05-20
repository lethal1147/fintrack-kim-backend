package postgres

import (
	"time"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

type merchantRow struct {
	Merchant string
	Total    float64
	Count    int
}

type txCountRow struct {
	Count int
}

func (r *TransactionRepo) Analytics(userID string, from, to time.Time) (*domain.AnalyticsResult, error) {
	cats, err := r.categorySummary(userID, from, to)
	if err != nil {
		return nil, err
	}
	months, err := r.monthSummary(userID, from, to)
	if err != nil {
		return nil, err
	}
	merchants, err := r.topMerchants(userID, from, to)
	if err != nil {
		return nil, err
	}
	txCount, err := r.txCount(userID, from, to)
	if err != nil {
		return nil, err
	}

	result := &domain.AnalyticsResult{
		TxCount:      txCount,
		ByCategory:   cats,
		ByMonth:      months,
		TopMerchants: merchants,
	}

	// Compute totals and pct from by_category
	var totalIncome, totalExpense float64
	for _, c := range cats {
		if c.Type == domain.TransactionIncome {
			totalIncome += c.Total
		} else {
			totalExpense += c.Total
		}
	}
	result.TotalIncome = totalIncome
	result.TotalExpense = totalExpense
	result.Net = totalIncome - totalExpense

	if totalIncome > 0 {
		result.SavingsRate = (totalIncome - totalExpense) / totalIncome * 100
	}

	// Back-fill pct into ByCategory
	for i, c := range result.ByCategory {
		var denom float64
		if c.Type == domain.TransactionIncome {
			denom = totalIncome
		} else {
			denom = totalExpense
		}
		if denom > 0 {
			result.ByCategory[i].Pct = c.Total / denom * 100
		}
	}

	return result, nil
}

func (r *TransactionRepo) topMerchants(userID string, from, to time.Time) ([]domain.MerchantStat, error) {
	var rows []merchantRow
	err := r.db.Raw(`
		SELECT merchant, SUM(amount) AS total, COUNT(*) AS count
		FROM transactions
		WHERE user_id = ? AND deleted_at IS NULL AND type = 'expense'
		  AND date >= ? AND date <= ?
		GROUP BY merchant
		ORDER BY total DESC
		LIMIT 5
	`, userID, from, to).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Internal(err.Error())
	}
	stats := make([]domain.MerchantStat, len(rows))
	for i, row := range rows {
		stats[i] = domain.MerchantStat{
			Merchant: row.Merchant,
			Total:    row.Total,
			Count:    row.Count,
		}
	}
	return stats, nil
}

func (r *TransactionRepo) txCount(userID string, from, to time.Time) (int, error) {
	var row txCountRow
	err := r.db.Raw(`
		SELECT COUNT(*) AS count
		FROM transactions
		WHERE user_id = ? AND deleted_at IS NULL AND date >= ? AND date <= ?
	`, userID, from, to).Scan(&row).Error
	if err != nil {
		return 0, apperror.Internal(err.Error())
	}
	return row.Count, nil
}
