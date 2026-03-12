package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshRepo struct{ db *pgxpool.Pool }

func NewRefreshRepo(db *pgxpool.Pool) *RefreshRepo { return &RefreshRepo{db: db} }

func (r *RefreshRepo) Create(ctx context.Context, userID int32, jti string, expiresAt time.Time, ua, ip string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO refresh_tokens(user_id, jti, expires_at, user_agent, ip_address)
		VALUES ($1,$2,$3,$4,$5)`, userID, jti, expiresAt, ua, ip)
	return err
}

func (r *RefreshRepo) Revoke(ctx context.Context, jti string) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked=true WHERE jti=$1`, jti)
	return err
}

func (r *RefreshRepo) IsActive(ctx context.Context, jti string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `
		SELECT (revoked=false AND now()<expires_at) AS active
		  FROM refresh_tokens WHERE jti=$1`, jti).Scan(&ok)
	return ok, err
}

func (r *RefreshRepo) RevokeAllForUser(ctx context.Context, userID int32) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked=true WHERE user_id=$1`, userID)
	return err
}
