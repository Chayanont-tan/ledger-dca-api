package entity

import "time"

type DCAPlan struct {
	ID              int        `json:"id"`
	UserID          int        `json:"user_id"`
	SourceAccountID int        `json:"source_account_id"`
	TargetAccountID int        `json:"target_account_id"`
	RetryCount      int        `json:"retry_count"`
	MaxCount        int        `json:"max_count"`
	Amount          int64      `json:"amount"`
	IntervalSeconds int        `json:"interval_seconds"`
	Status          string     `json:"status"`
	LastError       string     `json:"last_error"`
	LastRunAt       *time.Time `json:"last_run_at"`
	NextRunAt       time.Time  `json:"next_run_at"`
	CreatedAt       time.Time  `json:"created_at"`
}
