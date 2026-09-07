package dto

import "time"

type CreateDCAPlanRequest struct {
	UserID          int   `json:"user_id"`
	SourceAccountID int   `json:"source_account_id"`
	TargetAccountID int   `json:"target_account_id"`
	Amount          int64 `json:"amount"`           // หน่วยสตางค์
	IntervalSeconds int   `json:"interval_seconds"` // เช่น 60 วินาที
}

type DCAPlanResponse struct {
	ID        int       `json:"id"`
	Status    string    `json:"status"`
	NextRunAt time.Time `json:"next_run_at"`
	Message   string    `json:"message"`
}
