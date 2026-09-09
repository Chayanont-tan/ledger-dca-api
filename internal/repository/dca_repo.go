package repository

import (
	"context"
	"database/sql"
	"ledger-dca-engine/internal/entity"
	interfaces "ledger-dca-engine/internal/interface"
	"time"
)

type dcaRepository struct {
	db *sql.DB
}

func NewDCARepository(db *sql.DB) interfaces.DCARepository {
	return &dcaRepository{db: db}
}

func (r *dcaRepository) CreatePlan(ctx context.Context, plan *entity.DCAPlan) error {
	query := `
	INSERT INTO dca_plans (user_id, source_account_id, target_account_id, amount, interval_seconds, status, next_run_at) 
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	RETURNING id, created_at
	`
	return r.db.QueryRowContext(
		ctx, query,
		plan.UserID, plan.SourceAccountID, plan.TargetAccountID,
		plan.Amount, plan.IntervalSeconds, plan.Status, plan.NextRunAt,
	).Scan(&plan.ID, &plan.CreatedAt)
}

func (r *dcaRepository) GetDuePlans(ctx context.Context, limit int) ([]*entity.DCAPlan, error) {
	query := `
	SELECT id, user_id, source_account_id, target_account_id, amount, interval_seconds, status, last_run_at, next_run_at
	FROM dca_plans
	WHERE status = 'ACTIVE' AND next_run_at <= NOW()
	ORDER BY next_run_at ASC
	LIMIT $1
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []*entity.DCAPlan
	for rows.Next() {
		var p entity.DCAPlan
		if err := rows.Scan(&p.ID, &p.UserID, &p.SourceAccountID, &p.TargetAccountID, &p.Amount, &p.IntervalSeconds, &p.Status, &p.LastRunAt, &p.NextRunAt); err != nil {
			return nil, err
		}
		plans = append(plans, &p)
	}
	return plans, nil
}

func (r *dcaRepository) UpdatePlanNextRun(ctx context.Context, planID int, nextRun time.Time) error {
	query := `UPDATE dca_plans SET last_run_at = NOW(), next_run_at = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, nextRun, planID)
	if err != nil {
		return err
	}
	return nil
}

func (r *dcaRepository) UpdatePlanFailed(ctx context.Context, plan *entity.DCAPlan) error {
	query := `
        UPDATE dca_plans 
        SET retry_count = $1,
            next_run_at = $2,
            last_error = $3,
            status = CASE
                        WHEN $1 >= max_retries THEN 'FAILED'
                        ELSE 'ACTIVE'
                     END,
            updated_at = NOW()
        WHERE id = $4
    `
	_, err := r.db.ExecContext(ctx, query, plan.RetryCount, plan.NextRunAt, plan.LastError, plan.ID)
	return err
}

func (r *dcaRepository) UpdatePlanPaused(ctx context.Context, plan *entity.DCAPlan) error {
	query := `
		UPDATE dca_plans 
		SET retry_count = $1, 
		    next_run_at = $2, 
		    status = 'PAUSED',
		    updated_at = NOW() 
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, plan.RetryCount, plan.NextRunAt, plan.ID)
	return err
}

func (r *dcaRepository) UpdatePlanSuccess(ctx context.Context, plan *entity.DCAPlan) error {

	query := `
        UPDATE dca_plans 
        SET next_run_at = $1, 
            retry_count = 0, 
            last_error = NULL, 
            status = 'ACTIVE',
            updated_at = NOW()
        WHERE id = $2
    `
	_, err := r.db.ExecContext(ctx, query, plan.NextRunAt, plan.ID)
	return err
}
