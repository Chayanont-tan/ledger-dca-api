package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"ledger-dca-engine/internal/dto"
	"ledger-dca-engine/internal/entity"
	interfaces "ledger-dca-engine/internal/interface"
)

type dcaService struct {
	dcaRepo         interfaces.DCARepository
	transferService interfaces.TransferService
}

func NewDCAService(dcaRepo interfaces.DCARepository, transferService interfaces.TransferService) interfaces.DCAService {
	return &dcaService{
		dcaRepo:         dcaRepo,
		transferService: transferService,
	}
}

func (s *dcaService) CreatePlan(ctx context.Context, req dto.CreateDCAPlanRequest) (*dto.DCAPlanResponse, error) {

	if req.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}
	if req.IntervalSeconds < 5 {
		return nil, errors.New("interval must be at least 5 seconds")
	}
	if req.SourceAccountID == req.TargetAccountID {
		return nil, errors.New("source and target account cannot be the same")
	}
	firstRun := time.Now().Add(time.Duration(req.IntervalSeconds) * time.Second)
	plan := &entity.DCAPlan{
		UserID:          req.UserID,
		SourceAccountID: req.SourceAccountID,
		TargetAccountID: req.TargetAccountID,
		Amount:          req.Amount,
		IntervalSeconds: req.IntervalSeconds,
		Status:          "ACTIVE",
		NextRunAt:       firstRun,
	}

	if err := s.dcaRepo.CreatePlan(ctx, plan); err != nil {
		return nil, err
	}

	return &dto.DCAPlanResponse{
		ID:        plan.ID,
		Status:    plan.Status,
		NextRunAt: plan.NextRunAt,
		Message:   "DCA plan initialized successfully",
	}, nil
}

func (s *dcaService) ProcessDuePlans(ctx context.Context) error {
	plans, err := s.dcaRepo.GetDuePlans(ctx, 10)
	if err != nil {
		return err
	}
	for _, plan := range plans {
		idempotencyKey := fmt.Sprintf("dca-%d-%d", plan.ID, plan.NextRunAt.Unix())

		transferReq := dto.TransferRequest{
			IdempotencyKey: idempotencyKey,
			FromAccountID:  plan.SourceAccountID,
			ToAccountID:    plan.TargetAccountID,
			Amount:         plan.Amount,
			Description:    fmt.Sprintf("DCA Execution for Plan #%d", plan.ID),
		}

		res, err := s.transferService.ExecuteTransfer(ctx, transferReq)
		if err != nil {
			plan.RetryCount++
			plan.NextRunAt = time.Now().Add(15 * time.Second)
			plan.LastError = err.Error()

			if updateErr := s.dcaRepo.UpdatePlanFailed(ctx, plan); updateErr != nil {
				log.Printf("failed to update plan status: %v", updateErr)
			}
			continue 
		}

		log.Printf("✅ [DCA] Plan #%d executed successfully (TxID: %s)", plan.ID, res.TransactionID)
		plan.NextRunAt = time.Now().Add(time.Duration(plan.IntervalSeconds) * time.Second)
		if updateErr := s.dcaRepo.UpdatePlanSuccess(ctx, plan); updateErr != nil {
			log.Printf("❌ [DCA] Failed to update success plan: %v", updateErr)
		}
	}

	return nil
}
