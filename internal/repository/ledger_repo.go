package repository

import (
	"context"
	"database/sql"
	"errors"

	"ledger-dca-engine/internal/entity"
	interfaces "ledger-dca-engine/internal/interface"
)

type ledgerRepo struct {
	db *sql.DB
}

func NewLedgerRepository(db *sql.DB) interfaces.LedgerRepository {
	return &ledgerRepo{db: db}
}

func (r *ledgerRepo) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *ledgerRepo) GetTransactionByIdempotencyKey(ctx context.Context, key string) (*entity.Transaction, error) {

	query := `SELECT id, idempotency_key, transaction_type, description, created_at FROM transactions WHERE idempotency_key = $1`
	var t entity.Transaction

	err := r.db.QueryRowContext(ctx, query, key).Scan(&t.ID, &t.IdempotencyKey, &t.TransactionType, &t.Description, &t.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *ledgerRepo) CreateTransaction(ctx context.Context, tx *sql.Tx, t *entity.Transaction) error {
	query := `INSERT INTO transactions (id, idempotency_key, transaction_type, description) VALUES ($1, $2, $3, $4)`

	_, err := tx.ExecContext(ctx, query, t.ID, t.IdempotencyKey, t.TransactionType, t.Description)

	return err

}

func (r *ledgerRepo) GetAccountForUpdate(ctx context.Context, tx *sql.Tx, accountID int) (*entity.Account, error) {
	query := `SELECT id, user_id, account_type, currency, balance, created_at FROM accounts WHERE id = $1 FOR UPDATE`
	var a entity.Account

	err := tx.QueryRowContext(ctx, query, accountID).Scan(&a.ID, &a.UserID, &a.AccountType, &a.Currency, &a.Balance, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *ledgerRepo) UpdateAccountBalance(ctx context.Context, tx *sql.Tx, accountID int, newBalance int64) error {
	query := `UPDATE accounts SET balance = $1 WHERE id = $2`

	_, err := tx.ExecContext(ctx, query, newBalance, accountID)
	return err
}

func (r *ledgerRepo) CreateEntry(ctx context.Context, tx *sql.Tx, entry *entity.Entry) error {
	query := `INSERT INTO entries (transaction_id, account_id, amount) VALUES ($1, $2, $3)`

	_, err := tx.ExecContext(ctx, query, entry.TransactionID, entry.AccountID, entry.Amount)
	return err
}
