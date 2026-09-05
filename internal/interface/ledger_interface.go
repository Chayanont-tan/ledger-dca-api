package interfaces

import (
	"context"
	"database/sql"
	"ledger-dca-engine/internal/dto"
	"ledger-dca-engine/internal/entity"
)

type LedgerRepository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	GetTransactionByIdempotencyKey(ctx context.Context, key string) (*entity.Transaction, error)
	CreateTransaction(ctx context.Context, tx *sql.Tx, t *entity.Transaction) error
	GetAccountForUpdate(ctx context.Context, tx *sql.Tx, accountID int) (*entity.Account, error)
	UpdateAccountBalance(ctx context.Context, tx *sql.Tx, accountID int, newBalance int64) error
	CreateEntry(ctx context.Context, tx *sql.Tx, entry *entity.Entry) error
}

type TransferService interface {
	ExecuteTransfer(ctx context.Context, req dto.TransferRequest) (*dto.TransferResponse, error)
}
