package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ledger-dca-engine/internal/dto"
	"ledger-dca-engine/internal/entity"
	"ledger-dca-engine/internal/service"
)

// Helper ฟังก์ชันสำหรับสร้าง *sql.Tx ปลอมที่ปลอดภัย ไม่ panic ตอน Rollback/Commit
// Helper สร้าง mock tx โดยไม่บังคับลำดับ Commit/Rollback
func newDummyTx(t *testing.T) *sql.Tx {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	// อนุญาตให้เรียก Rollback หรือ Commit ตัวไหนก่อนก็ได้
	mock.MatchExpectationsInOrder(false)
	mock.ExpectBegin()
	mock.ExpectCommit()
	mock.ExpectRollback()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("failed to begin mock tx: %v", err)
	}
	return tx
}

// --- Mock Ledger Repository ---
type mockLedgerRepo struct {
	tx                               *sql.Tx
	beginTxFn                        func(ctx context.Context) (*sql.Tx, error)
	getTransactionByIdempotencyKeyFn func(ctx context.Context, key string) (*entity.Transaction, error)
	getAccountForUpdateFn            func(ctx context.Context, tx *sql.Tx, accountID int) (*entity.Account, error)
	createTransactionFn              func(ctx context.Context, tx *sql.Tx, record *entity.Transaction) error
	createEntryFn                    func(ctx context.Context, tx *sql.Tx, entry *entity.Entry) error
	updateAccountBalanceFn           func(ctx context.Context, tx *sql.Tx, accountID int, newBalance int64) error

	lockedAccountIDs []int
}

func (m *mockLedgerRepo) BeginTx(ctx context.Context) (*sql.Tx, error) {
	if m.beginTxFn != nil {
		return m.beginTxFn(ctx)
	}
	return m.tx, nil
}

func (m *mockLedgerRepo) GetTransactionByIdempotencyKey(ctx context.Context, key string) (*entity.Transaction, error) {
	if m.getTransactionByIdempotencyKeyFn != nil {
		return m.getTransactionByIdempotencyKeyFn(ctx, key)
	}
	return nil, nil
}

func (m *mockLedgerRepo) GetAccountForUpdate(ctx context.Context, tx *sql.Tx, accountID int) (*entity.Account, error) {
	m.lockedAccountIDs = append(m.lockedAccountIDs, accountID)
	if m.getAccountForUpdateFn != nil {
		return m.getAccountForUpdateFn(ctx, tx, accountID)
	}
	return &entity.Account{ID: accountID, Balance: 1000}, nil
}

func (m *mockLedgerRepo) CreateTransaction(ctx context.Context, tx *sql.Tx, record *entity.Transaction) error {
	if m.createTransactionFn != nil {
		return m.createTransactionFn(ctx, tx, record)
	}
	return nil
}

func (m *mockLedgerRepo) CreateEntry(ctx context.Context, tx *sql.Tx, entry *entity.Entry) error {
	if m.createEntryFn != nil {
		return m.createEntryFn(ctx, tx, entry)
	}
	return nil
}

func (m *mockLedgerRepo) UpdateAccountBalance(ctx context.Context, tx *sql.Tx, accountID int, newBalance int64) error {
	if m.updateAccountBalanceFn != nil {
		return m.updateAccountBalanceFn(ctx, tx, accountID, newBalance)
	}
	return nil
}

// --- Test Cases ---

func TestExecuteTransfer(t *testing.T) {
	ctx := context.Background()

	t.Run("Fail - Invalid Amount (Amount <= 0)", func(t *testing.T) {
		svc := service.NewTransferService(&mockLedgerRepo{tx: newDummyTx(t)})
		req := dto.TransferRequest{FromAccountID: 1, ToAccountID: 2, Amount: 0}

		_, err := svc.ExecuteTransfer(ctx, req)
		if !errors.Is(err, service.ErrInvalidAmount) {
			t.Fatalf("expected ErrInvalidAmount, got: %v", err)
		}
	})

	t.Run("Fail - Same Account Transfer", func(t *testing.T) {
		svc := service.NewTransferService(&mockLedgerRepo{tx: newDummyTx(t)})
		req := dto.TransferRequest{FromAccountID: 1, ToAccountID: 1, Amount: 100}

		_, err := svc.ExecuteTransfer(ctx, req)
		if !errors.Is(err, service.ErrSameAccountTransfer) {
			t.Fatalf("expected ErrSameAccountTransfer, got: %v", err)
		}
	})

	t.Run("Success - Idempotency Key already exists (Skip Transfer)", func(t *testing.T) {
		repo := &mockLedgerRepo{
			tx: newDummyTx(t),
			getTransactionByIdempotencyKeyFn: func(ctx context.Context, key string) (*entity.Transaction, error) {
				return &entity.Transaction{ID: "tx-existing-123"}, nil
			},
		}
		svc := service.NewTransferService(repo)
		req := dto.TransferRequest{IdempotencyKey: "idem-dup", FromAccountID: 1, ToAccountID: 2, Amount: 100}

		res, err := svc.ExecuteTransfer(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Status != "ALREADY_PROCESSED" || res.TransactionID != "tx-existing-123" {
			t.Fatalf("expected ALREADY_PROCESSED with id tx-existing-123, got: %+v", res)
		}
	})

	t.Run("Fail - Insufficient Balance", func(t *testing.T) {
		repo := &mockLedgerRepo{
			tx: newDummyTx(t),
			getAccountForUpdateFn: func(ctx context.Context, tx *sql.Tx, accountID int) (*entity.Account, error) {
				if accountID == 1 {
					return &entity.Account{ID: 1, Balance: 50}, nil
				}
				return &entity.Account{ID: 2, Balance: 0}, nil
			},
		}
		svc := service.NewTransferService(repo)
		req := dto.TransferRequest{FromAccountID: 1, ToAccountID: 2, Amount: 100}

		_, err := svc.ExecuteTransfer(ctx, req)
		if !errors.Is(err, service.ErrInsufficientBalance) {
			t.Fatalf("expected ErrInsufficientBalance, got: %v", err)
		}
	})

	t.Run("Success - Deadlock Prevention Lock Ordering (ID น้อยต้องโดน Lock ก่อนเสมอ)", func(t *testing.T) {
		repo1 := &mockLedgerRepo{tx: newDummyTx(t)}
		svc1 := service.NewTransferService(repo1)
		_, err := svc1.ExecuteTransfer(ctx, dto.TransferRequest{FromAccountID: 1, ToAccountID: 2, Amount: 50})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo1.lockedAccountIDs[0] != 1 || repo1.lockedAccountIDs[1] != 2 {
			t.Fatalf("expected lock order [1, 2], got: %v", repo1.lockedAccountIDs)
		}

		repo2 := &mockLedgerRepo{tx: newDummyTx(t)}
		svc2 := service.NewTransferService(repo2)
		_, err = svc2.ExecuteTransfer(ctx, dto.TransferRequest{FromAccountID: 20, ToAccountID: 5, Amount: 50})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo2.lockedAccountIDs[0] != 5 || repo2.lockedAccountIDs[1] != 20 {
			t.Fatalf("expected lock order [5, 20], got: %v", repo2.lockedAccountIDs)
		}
	})

	t.Run("Success - Complete Transfer Flow", func(t *testing.T) {
		var updatedBalances = make(map[int]int64)

		repo := &mockLedgerRepo{
			tx: newDummyTx(t),
			getAccountForUpdateFn: func(ctx context.Context, tx *sql.Tx, accountID int) (*entity.Account, error) {
				if accountID == 1 {
					return &entity.Account{ID: 1, Balance: 500}, nil
				}
				return &entity.Account{ID: 2, Balance: 100}, nil
			},
			updateAccountBalanceFn: func(ctx context.Context, tx *sql.Tx, accountID int, newBalance int64) error {
				updatedBalances[accountID] = newBalance
				return nil
			},
		}

		svc := service.NewTransferService(repo)
		req := dto.TransferRequest{
			IdempotencyKey: "idem-success",
			FromAccountID:  1,
			ToAccountID:    2,
			Amount:         150,
			Description:    "Transfer Test",
		}

		res, err := svc.ExecuteTransfer(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if res.Status != "SUCCESS" {
			t.Fatalf("expected status SUCCESS, got: %s", res.Status)
		}

		if updatedBalances[1] != 350 {
			t.Errorf("expected FromAccount balance 350, got: %d", updatedBalances[1])
		}
		if updatedBalances[2] != 250 {
			t.Errorf("expected ToAccount balance 250, got: %d", updatedBalances[2])
		}
	})
}
