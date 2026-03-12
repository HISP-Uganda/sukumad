package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"sukumad/internal/types"
)

type DeliveryRepo struct{ db *pgxpool.Pool }

func NewDeliveryRepo(db *pgxpool.Pool) *DeliveryRepo { return &DeliveryRepo{db: db} }

func (r *DeliveryRepo) List(ctx context.Context, status string, limit, offset int) ([]types.Delivery, error) {
	// var rows pgxpool.Rows // <-- WRONG: keep this comment for clarity; don't use this type
	// _ = rows              // avoid “declared and not used” if you keep the comment

	// Correct way: rely on type inference; r.db.Query returns pgx.Rows
	var (
		sql  string
		args []any
	)
	if status != "" {
		sql = `
			SELECT id, request_id, server_id, status, attempts, next_run_at, retry_after_at, last_error,
			       status_code, response_body, priority, scheduled_at, sent_at, updated_at,
			       rate_class, rps_override, burst_override, timeout_ms, max_attempts,
			       depends_on_delivery_id, async_jobid, async_status, async_response, async_poll_url, async_last_polled
			  FROM deliveries WHERE status=$1
			  ORDER BY next_run_at, id
			  LIMIT $2 OFFSET $3`
		args = []any{status, limit, offset}
	} else {
		sql = `
			SELECT id, request_id, server_id, status, attempts, next_run_at, retry_after_at, last_error,
			       status_code, response_body, priority, scheduled_at, sent_at, updated_at,
			       rate_class, rps_override, burst_override, timeout_ms, max_attempts,
			       depends_on_delivery_id, async_jobid, async_status, async_response, async_poll_url, async_last_polled
			  FROM deliveries
			  ORDER BY next_run_at, id
			  LIMIT $1 OFFSET $2`
		args = []any{limit, offset}
	}

	rows2, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	var out []types.Delivery
	for rows2.Next() {
		var d types.Delivery
		if err := rows2.Scan(
			&d.ID, &d.RequestID, &d.ServerID, &d.Status, &d.Attempts, &d.NextRunAt, &d.RetryAfterAt, &d.LastError,
			&d.StatusCode, &d.ResponseBody, &d.Priority, &d.ScheduledAt, &d.SentAt, &d.UpdatedAt,
			&d.RateClass, &d.RPSOverride, &d.BurstOverride, &d.TimeoutMS, &d.MaxAttempts,
			&d.DependsOnDeliveryID, &d.AsyncJobID, &d.AsyncStatus, &d.AsyncResponse, &d.AsyncPollURL, &d.AsyncLastPolled,
		); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows2.Err()
}

func (r *DeliveryRepo) Get(ctx context.Context, id int64) (*types.Delivery, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, request_id, server_id, status, attempts, next_run_at, retry_after_at, last_error,
		       status_code, response_body, priority, scheduled_at, sent_at, updated_at,
		       rate_class, rps_override, burst_override, timeout_ms, max_attempts,
		       depends_on_delivery_id, async_jobid, async_status, async_response, async_poll_url, async_last_polled
		  FROM deliveries WHERE id=$1`, id)

	var d types.Delivery
	if err := row.Scan(
		&d.ID, &d.RequestID, &d.ServerID, &d.Status, &d.Attempts, &d.NextRunAt, &d.RetryAfterAt, &d.LastError,
		&d.StatusCode, &d.ResponseBody, &d.Priority, &d.ScheduledAt, &d.SentAt, &d.UpdatedAt,
		&d.RateClass, &d.RPSOverride, &d.BurstOverride, &d.TimeoutMS, &d.MaxAttempts,
		&d.DependsOnDeliveryID, &d.AsyncJobID, &d.AsyncStatus, &d.AsyncResponse, &d.AsyncPollURL, &d.AsyncLastPolled,
	); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *DeliveryRepo) Patch(ctx context.Context, id int64, status *string, nextRunAt *time.Time, priority *int32) error {
	_, err := r.db.Exec(ctx, `
		UPDATE deliveries SET
			status = COALESCE($2, status),
			next_run_at = COALESCE($3, next_run_at),
			priority = COALESCE($4, priority),
			updated_at = now()
		WHERE id=$1`, id, status, nextRunAt, priority)
	return err
}

func (r *DeliveryRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `DELETE FROM deliveries WHERE id=$1`, id)
	return err
}
