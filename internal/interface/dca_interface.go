package interfaces

import (
	"context"
	"ledger-dca-engine/internal/dto"
	"ledger-dca-engine/internal/entity"
	"time"
)

type DCARepository interface {
	CreatePlan(ctx context.Context, plan *entity.DCAPlan) error // สร้างเเผน dca
	GetDuePlans(ctx context.Context, limit int) ([]*entity.DCAPlan, error)
	UpdatePlanNextRun(ctx context.Context, planID int, nextRun time.Time) error

	UpdatePlanFailed(ctx context.Context, plan *entity.DCAPlan) error
    UpdatePlanPaused(ctx context.Context, plan *entity.DCAPlan) error
    UpdatePlanSuccess(ctx context.Context, plan *entity.DCAPlan) error
}

type DCAService interface {
	CreatePlan(ctx context.Context, req dto.CreateDCAPlanRequest) (*dto.DCAPlanResponse, error)
	ProcessDuePlans(ctx context.Context) error
}
