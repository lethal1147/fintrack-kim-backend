package postgres

import (
	"time"

	"gorm.io/gorm"

	"github.com/joakim/fintrack-api/internal/domain"
	"github.com/joakim/fintrack-api/pkg/apperror"
)

type transactionModel struct {
	ID        string     `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID    string     `gorm:"not null;index"`
	Merchant  string     `gorm:"not null"`
	Category  string     `gorm:"not null"`
	Note      string     `gorm:"not null;default:''"`
	Date      time.Time  `gorm:"not null;type:date"`
	Amount    float64    `gorm:"not null;type:decimal(12,2)"`
	Type      string     `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time `gorm:"index"`
}

func (transactionModel) TableName() string { return "transactions" }

func toTransactionDomain(m *transactionModel) *domain.Transaction {
	return &domain.Transaction{
		ID:        m.ID,
		UserID:    m.UserID,
		Merchant:  m.Merchant,
		Category:  m.Category,
		Note:      m.Note,
		Date:      m.Date,
		Amount:    m.Amount,
		Type:      domain.TransactionType(m.Type),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

type TransactionRepo struct{ db *gorm.DB }

func NewTransactionRepo(db *gorm.DB) *TransactionRepo { return &TransactionRepo{db: db} }

func (r *TransactionRepo) Create(tx *domain.Transaction) error {
	m := &transactionModel{
		UserID:   tx.UserID,
		Merchant: tx.Merchant,
		Category: tx.Category,
		Note:     tx.Note,
		Date:     tx.Date,
		Amount:   tx.Amount,
		Type:     string(tx.Type),
	}
	if err := r.db.Create(m).Error; err != nil {
		return apperror.Internal(err.Error())
	}
	tx.ID = m.ID
	tx.CreatedAt = m.CreatedAt
	tx.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *TransactionRepo) GetByID(id, userID string) (*domain.Transaction, error) {
	var m transactionModel
	err := r.db.Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).First(&m).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound("transaction not found")
	}
	if err != nil {
		return nil, apperror.Internal(err.Error())
	}
	return toTransactionDomain(&m), nil
}

func (r *TransactionRepo) Update(tx *domain.Transaction) error {
	result := r.db.Model(&transactionModel{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", tx.ID, tx.UserID).
		Updates(map[string]interface{}{
			"merchant":   tx.Merchant,
			"category":   tx.Category,
			"note":       tx.Note,
			"date":       tx.Date,
			"amount":     tx.Amount,
			"type":       string(tx.Type),
			"updated_at": time.Now(),
		})
	if result.Error != nil {
		return apperror.Internal(result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("transaction not found")
	}
	return nil
}

func (r *TransactionRepo) SoftDelete(id, userID string) error {
	result := r.db.Model(&transactionModel{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).
		Update("deleted_at", time.Now())
	if result.Error != nil {
		return apperror.Internal(result.Error.Error())
	}
	if result.RowsAffected == 0 {
		return apperror.NotFound("transaction not found")
	}
	return nil
}

func (r *TransactionRepo) List(userID string, f domain.TransactionFilter) ([]*domain.Transaction, int, error) {
	page, limit := normalizePage(f.Page, f.Limit)

	q := r.db.Model(&transactionModel{}).Where("user_id = ? AND deleted_at IS NULL", userID)
	q = applyFilters(q, f)

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, apperror.Internal(err.Error())
	}

	var models []transactionModel
	offset := (page - 1) * limit
	if err := q.Order("date DESC, created_at DESC").Limit(limit).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, apperror.Internal(err.Error())
	}

	txs := make([]*domain.Transaction, len(models))
	for i := range models {
		txs[i] = toTransactionDomain(&models[i])
	}
	return txs, int(total), nil
}

func applyFilters(q *gorm.DB, f domain.TransactionFilter) *gorm.DB {
	if f.Type != "" {
		q = q.Where("type = ?", f.Type)
	}
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	if f.Search != "" {
		like := "%" + f.Search + "%"
		q = q.Where("(merchant ILIKE ? OR note ILIKE ?)", like, like)
	}
	if f.From != "" {
		q = q.Where("date >= ?", f.From)
	}
	if f.To != "" {
		q = q.Where("date <= ?", f.To)
	}
	return q
}

func normalizePage(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 8
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

// ── Summary ────────────────────────────────────────────────────────────────

type categoryRow struct {
	Category string
	Type     string
	Total    float64
	Count    int
}

type monthRow struct {
	Month   string
	Income  float64
	Expense float64
}

func (r *TransactionRepo) Summary(userID string, from, to time.Time) (*domain.TransactionSummaryResult, error) {
	cats, err := r.categorySummary(userID, from, to)
	if err != nil {
		return nil, err
	}
	months, err := r.monthSummary(userID, from, to)
	if err != nil {
		return nil, err
	}
	return &domain.TransactionSummaryResult{
		ByCategory: cats,
		ByMonth:    months,
	}, nil
}

func (r *TransactionRepo) categorySummary(userID string, from, to time.Time) ([]domain.CategoryStat, error) {
	var rows []categoryRow
	err := r.db.Raw(`
		SELECT category, type, SUM(amount) AS total, COUNT(*) AS count
		FROM transactions
		WHERE user_id = ? AND deleted_at IS NULL AND date >= ? AND date <= ?
		GROUP BY category, type
		ORDER BY total DESC
	`, userID, from, to).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Internal(err.Error())
	}
	stats := make([]domain.CategoryStat, len(rows))
	for i, row := range rows {
		stats[i] = domain.CategoryStat{
			Category: row.Category,
			Type:     domain.TransactionType(row.Type),
			Total:    row.Total,
			Count:    row.Count,
		}
	}
	return stats, nil
}

func (r *TransactionRepo) monthSummary(userID string, from, to time.Time) ([]domain.MonthStat, error) {
	var rows []monthRow
	err := r.db.Raw(`
		SELECT to_char(date_trunc('month', date), 'YYYY-MM') AS month,
		       SUM(CASE WHEN type = 'income'  THEN amount ELSE 0 END) AS income,
		       SUM(CASE WHEN type = 'expense' THEN amount ELSE 0 END) AS expense
		FROM transactions
		WHERE user_id = ? AND deleted_at IS NULL AND date >= ? AND date <= ?
		GROUP BY date_trunc('month', date)
		ORDER BY date_trunc('month', date)
	`, userID, from, to).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Internal(err.Error())
	}
	stats := make([]domain.MonthStat, len(rows))
	for i, row := range rows {
		stats[i] = domain.MonthStat{
			Month:   row.Month,
			Income:  row.Income,
			Expense: row.Expense,
			Net:     row.Income - row.Expense,
		}
	}
	return stats, nil
}
