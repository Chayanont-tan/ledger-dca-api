package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"ledger-dca-engine/internal/dto"
	"ledger-dca-engine/internal/entity"
	"ledger-dca-engine/internal/service"
)

// --- Mock DCA Repository ---
type mockDCARepo struct {
	createPlanFn        func(ctx context.Context, plan *entity.DCAPlan) error
	getDuePlansFn       func(ctx context.Context, limit int) ([]*entity.DCAPlan, error)
	updatePlanNextRunFn func(ctx context.Context, planID int, nextRun time.Time) error
	updatePlanFailedFn  func(ctx context.Context, plan *entity.DCAPlan) error
	updatePlanPausedFn  func(ctx context.Context, plan *entity.DCAPlan) error
	updatePlanSuccessFn func(ctx context.Context, plan *entity.DCAPlan) error
}

func (m *mockDCARepo) CreatePlan(ctx context.Context, plan *entity.DCAPlan) error {
	if m.createPlanFn != nil {
		return m.createPlanFn(ctx, plan)
	}
	return nil
}

func (m *mockDCARepo) GetDuePlans(ctx context.Context, limit int) ([]*entity.DCAPlan, error) {
	if m.getDuePlansFn != nil {
		return m.getDuePlansFn(ctx, limit)
	}
	return nil, nil
}

func (m *mockDCARepo) UpdatePlanNextRun(ctx context.Context, planID int, nextRun time.Time) error {
	if m.updatePlanNextRunFn != nil {
		return m.updatePlanNextRunFn(ctx, planID, nextRun)
	}
	return nil
}

func (m *mockDCARepo) UpdatePlanFailed(ctx context.Context, plan *entity.DCAPlan) error {
	if m.updatePlanFailedFn != nil {
		return m.updatePlanFailedFn(ctx, plan)
	}
	return nil
}

func (m *mockDCARepo) UpdatePlanPaused(ctx context.Context, plan *entity.DCAPlan) error {
	if m.updatePlanPausedFn != nil {
		return m.updatePlanPausedFn(ctx, plan)
	}
	return nil
}

func (m *mockDCARepo) UpdatePlanSuccess(ctx context.Context, plan *entity.DCAPlan) error {
	if m.updatePlanSuccessFn != nil {
		return m.updatePlanSuccessFn(ctx, plan)
	}
	return nil
}

// --- Mock Transfer Service ---
type mockTransferService struct {
	executeTransferFn func(ctx context.Context, req dto.TransferRequest) (*dto.TransferResponse, error)
}

func (m *mockTransferService) ExecuteTransfer(ctx context.Context, req dto.TransferRequest) (*dto.TransferResponse, error) {
	if m.executeTransferFn != nil {
		return m.executeTransferFn(ctx, req)
	}
	return &dto.TransferResponse{TransactionID: "tx-mock-123"}, nil
}

// --- Test Suites ---

func TestCreatePlanValidation(t *testing.T) {
	ctx := context.Background()

	t.Run("Fail - Amount is zero or negative", func(t *testing.T) {
		svc := service.NewDCAService(&mockDCARepo{}, &mockTransferService{})
		_, err := svc.CreatePlan(ctx, dto.CreateDCAPlanRequest{Amount: -10, IntervalSeconds: 10, SourceAccountID: 1, TargetAccountID: 2})
		if err == nil || err.Error() != "amount must be greater than zero" {
			t.Fatalf("expected amount error, got: %v", err)
		}
	})

	t.Run("Fail - Interval is less than 5 seconds", func(t *testing.T) {
		svc := service.NewDCAService(&mockDCARepo{}, &mockTransferService{})
		_, err := svc.CreatePlan(ctx, dto.CreateDCAPlanRequest{Amount: 100, IntervalSeconds: 2, SourceAccountID: 1, TargetAccountID: 2})
		if err == nil || err.Error() != "interval must be at least 5 seconds" {
			t.Fatalf("expected interval error, got: %v", err)
		}
	})

	t.Run("Fail - Same Source and Target Accounts", func(t *testing.T) {
		svc := service.NewDCAService(&mockDCARepo{}, &mockTransferService{})
		_, err := svc.CreatePlan(ctx, dto.CreateDCAPlanRequest{Amount: 100, IntervalSeconds: 10, SourceAccountID: 1, TargetAccountID: 1})
		if err == nil || err.Error() != "source and target account cannot be the same" {
			t.Fatalf("expected same account error, got: %v", err)
		}
	})

	t.Run("Success - Create Plan Successfully", func(t *testing.T) {
		repo := &mockDCARepo{
			createPlanFn: func(ctx context.Context, plan *entity.DCAPlan) error {
				plan.ID = 7
				return nil
			},
		}
		svc := service.NewDCAService(repo, &mockTransferService{})
		res, err := svc.CreatePlan(ctx, dto.CreateDCAPlanRequest{Amount: 500, IntervalSeconds: 60, SourceAccountID: 1, TargetAccountID: 2})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ID != 7 || res.Status != "ACTIVE" {
			t.Fatalf("expected ID 7 and ACTIVE status, got: %+v", res)
		}
	})
}

func TestProcessDuePlans_ExecutionFlows(t *testing.T) {
	ctx := context.Background()

	t.Run("Success Flow - ExecuteTransfer succeeds and calls UpdatePlanSuccess", func(t *testing.T) {
		duePlan := &entity.DCAPlan{
			ID:              1,
			SourceAccountID: 10,
			TargetAccountID: 20,
			Amount:          300,
			IntervalSeconds: 30,
			NextRunAt:       time.Now().Add(-10 * time.Second),
			Status:          "ACTIVE",
		}

		transferCalled := false
		updateSuccessCalled := false
		updateFailedCalled := false

		repo := &mockDCARepo{
			getDuePlansFn: func(ctx context.Context, limit int) ([]*entity.DCAPlan, error) {
				return []*entity.DCAPlan{duePlan}, nil
			},
			updatePlanSuccessFn: func(ctx context.Context, plan *entity.DCAPlan) error {
				updateSuccessCalled = true
				return nil
			},
			updatePlanFailedFn: func(ctx context.Context, plan *entity.DCAPlan) error {
				updateFailedCalled = true
				return nil
			},
		}

		transferSvc := &mockTransferService{
			executeTransferFn: func(ctx context.Context, req dto.TransferRequest) (*dto.TransferResponse, error) {
				transferCalled = true
				return &dto.TransferResponse{TransactionID: "tx-success"}, nil
			},
		}

		svc := service.NewDCAService(repo, transferSvc)
		err := svc.ProcessDuePlans(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !transferCalled {
			t.Fatal("expected ExecuteTransfer to be called")
		}
		if !updateSuccessCalled {
			t.Fatal("expected UpdatePlanSuccess to be called")
		}
		if updateFailedCalled {
			t.Fatal("UpdatePlanFailed must NOT be called on success")
		}
	})

	t.Run("Failure Flow - ExecuteTransfer fails (e.g. Insufficient Balance) triggers UpdatePlanFailed and increments RetryCount", func(t *testing.T) {
		duePlan := &entity.DCAPlan{
			ID:              2,
			SourceAccountID: 10,
			TargetAccountID: 20,
			Amount:          500,
			IntervalSeconds: 60,
			RetryCount:      0,
			NextRunAt:       time.Now().Add(-10 * time.Second),
			Status:          "ACTIVE",
		}

		updateFailedCalled := false
		updateSuccessCalled := false

		repo := &mockDCARepo{
			getDuePlansFn: func(ctx context.Context, limit int) ([]*entity.DCAPlan, error) {
				return []*entity.DCAPlan{duePlan}, nil
			},
			updatePlanFailedFn: func(ctx context.Context, plan *entity.DCAPlan) error {
				updateFailedCalled = true
				// ตรวจสอบว่าบวก retry_count เพิ่ม 1 จริงไหม
				if plan.RetryCount != 1 {
					t.Errorf("expected RetryCount to be 1, got: %d", plan.RetryCount)
				}
				// ตรวจสอบว่าเก็บ last_error ตรงกับ error ที่ได้รับไหม
				if plan.LastError != "insufficient balance" {
					t.Errorf("expected last error 'insufficient balance', got: %s", plan.LastError)
				}
				return nil
			},
			updatePlanSuccessFn: func(ctx context.Context, plan *entity.DCAPlan) error {
				updateSuccessCalled = true
				return nil
			},
		}

		transferSvc := &mockTransferService{
			executeTransferFn: func(ctx context.Context, req dto.TransferRequest) (*dto.TransferResponse, error) {
				return nil, errors.New("insufficient balance")
			},
		}

		svc := service.NewDCAService(repo, transferSvc)
		err := svc.ProcessDuePlans(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !updateFailedCalled {
			t.Fatal("expected UpdatePlanFailed to be called on failure")
		}
		if updateSuccessCalled {
			t.Fatal("UpdatePlanSuccess must NOT be called on failure (check continue statement)")
		}
	})
}
