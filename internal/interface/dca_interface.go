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
}

type DCAService interface {
	CreatePlan(ctx context.Context, req dto.CreateDCAPlanRequest) (*dto.DCAPlanResponse, error)
	ProcessDuePlans(ctx context.Context) error
}
