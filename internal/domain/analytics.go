package domain

import "time"

type MerchantStat struct {
	Merchant string
	Total    float64
	Count    int
}

type AnalyticsResult struct {
	TotalIncome  float64
	TotalExpense float64
	Net          float64
	TxCount      int
	SavingsRate  float64
	ByCategory   []CategoryStat
	ByMonth      []MonthStat
	TopMerchants []MerchantStat
}

type AnalyticsRepository interface {
	Analytics(userID string, from, to time.Time) (*AnalyticsResult, error)
}

type AnalyticsServiceInterface interface {
	Analytics(userID string, from, to time.Time) (*AnalyticsResult, error)
}
