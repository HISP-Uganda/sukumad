package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ResetRepo struct{ db *pgxpool.Pool }

func NewResetRepo(db *pgxpool.Pool) *ResetRepo { return &ResetRepo{db: db} }

func (r *ResetRepo) Create(ctx context.Context, userID int32, tokenHash string, exp time.Time, ua, ip string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO password_reset_tokens (user_id, token_hash, expires_at, user_agent, ip_address)
		VALUES ($1,$2,$3,$4,$5)`, userID, tokenHash, exp, ua, ip)
	return err
}

func (r *ResetRepo) Use(ctx context.Context, tokenHash string) (int32, error) {
	// atomically mark used if valid
	var userID int32
	err := r.db.QueryRow(ctx, `
		UPDATE password_reset_tokens
		   SET used_at = now()
		 WHERE token_hash = $1
		   AND used_at IS NULL
		   AND now() < expires_at
		RETURNING user_id`, tokenHash).Scan(&userID)
	return userID, err
}
