package service

import (
	"context"
	"errors"
	"fmt"

	"ledger-dca-engine/internal/dto"
	"ledger-dca-engine/internal/entity"
	interfaces "ledger-dca-engine/internal/interface"

	"github.com/google/uuid"
)

var (
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInvalidAmount       = errors.New("amount must be greater than zero")
	ErrSameAccountTransfer = errors.New("cannot transfer to the same account")
)

type TransferService struct {
	repo interfaces.LedgerRepository
}

func NewTransferService(repo interfaces.LedgerRepository) interfaces.TransferService {
	return &TransferService{repo: repo}
}

func (s *TransferService) ExecuteTransfer(ctx context.Context, req dto.TransferRequest) (*dto.TransferResponse, error) {

	if req.Amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if req.FromAccountID == req.ToAccountID {
		return nil, ErrSameAccountTransfer
	}

	existingTx, err := s.repo.GetTransactionByIdempotencyKey(ctx, req.IdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing transaction: %w", err)
	}
	if existingTx != nil {
		return &dto.TransferResponse{
			TransactionID: existingTx.ID,
			Status:        "ALREADY_PROCESSED",
			Message:       "Transaction was already processed successfully",
		}, nil
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	firstLockID, secondLockID := req.FromAccountID, req.ToAccountID
	if req.FromAccountID > req.ToAccountID {
		firstLockID, secondLockID = req.FromAccountID, req.ToAccountID
	}
	accMap := make(map[int]*entity.Account)
	
	firstAccount, err := s.repo.GetAccountForUpdate(ctx, tx, firstLockID)
	if err != nil {
		return nil, fmt.Errorf("failed to lock account %d: %w", firstLockID, err)
	}
	accMap[firstLockID] = firstAccount
	secondAccount, err := s.repo.GetAccountForUpdate(ctx, tx, secondLockID)
	if err != nil {
		return nil, fmt.Errorf("failed to lock account %d: %w", secondLockID, err)
	}
	accMap[secondLockID] = secondAccount

	fromAccount := accMap[req.FromAccountID]
	toAccount := accMap[req.ToAccountID]

	if fromAccount.Balance < req.Amount {
		return nil, ErrInsufficientBalance
	}

	txID := uuid.New().String()
	transectionRecord := &entity.Transaction{
		ID:              txID,
		IdempotencyKey:  req.IdempotencyKey,
		TransactionType: "TRANSFER",
		Description:     req.Description,
	}
	if err := s.repo.CreateTransaction(ctx, tx, transectionRecord); err != nil {
		return nil, err
	}

	creditEntry := &entity.Entry{
		TransactionID: txID,
		AccountID:     req.FromAccountID,
		Amount:        -req.Amount,
	}
	if err := s.repo.CreateEntry(ctx, tx, creditEntry); err != nil {
		return nil, err
	}

	debitEntry := &entity.Entry{
		TransactionID: txID,
		AccountID:     req.ToAccountID,
		Amount:        req.Amount,
	}

	if err := s.repo.CreateEntry(ctx, tx, debitEntry); err != nil {
		return nil, err
	}

	if err := s.repo.UpdateAccountBalance(ctx, tx, fromAccount.ID, fromAccount.Balance-req.Amount); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateAccountBalance(ctx, tx, toAccount.ID, toAccount.Balance+req.Amount); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &dto.TransferResponse{
		TransactionID: txID,
		Status:        "SUCCESS",
		Message:       "Transfer completed successfully",
	}, nil

}
