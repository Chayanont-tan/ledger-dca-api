package entity

import "time"

type Account struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	AccountType string    `json:"account_type"`
	Currency    string    `json:"currency"`
	Balance     int64     `json:"balance"` 
	CreatedAt   time.Time `json:"created_at"`
}

type Transaction struct {
	ID              string    `json:"id"`
	IdempotencyKey  string    `json:"idempotency_key"`
	TransactionType string    `json:"transaction_type"`
	Description     string    `json:"description"`
	CreatedAt       time.Time `json:"created_at"`
}

type Entry struct {
	ID            int       `json:"id"`
	TransactionID string    `json:"transaction_id"`
	AccountID     int       `json:"account_id"`
	Amount        int64     `json:"amount"` 
	CreatedAt     time.Time `json:"created_at"`
}