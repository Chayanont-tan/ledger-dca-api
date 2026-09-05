package dto

type TransferRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	FromAccountID  int    `json:"from_account_id"`
	ToAccountID    int    `json:"to_account_id"`
	Amount         int64  `json:"amount"` // หน่วยสตางค์
	Description    string `json:"description"`
}

type TransferResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}
