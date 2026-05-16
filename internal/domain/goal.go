package domain

import "time"

type Goal struct {
	ID                  string
	UserID              string
	Name                string
	Emoji               string
	Target              float64
	Current             float64
	TargetDate          time.Time
	MonthlyContribution float64
	Color               string
}
