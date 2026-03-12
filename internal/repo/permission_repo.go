package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"sukumad/internal/types"
)

type PermissionRepo struct{ db *pgxpool.Pool }

func NewPermissionRepo(db *pgxpool.Pool) *PermissionRepo { return &PermissionRepo{db: db} }

func (r *PermissionRepo) List(ctx context.Context) ([]types.Permission, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, code, system_module, created, updated FROM permissions ORDER BY system_module, code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []types.Permission
	for rows.Next() {
		var p types.Permission
		if err := rows.Scan(&p.ID, &p.Name, &p.Code, &p.SystemModule, &p.Created, &p.Updated); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (r *PermissionRepo) Create(ctx context.Context, p *types.Permission) (*types.Permission, error) {
	row := r.db.QueryRow(ctx, `INSERT INTO permissions(name, code, system_module) VALUES ($1,$2,$3) RETURNING id, created, updated`,
		p.Name, p.Code, p.SystemModule)
	if err := row.Scan(&p.ID, &p.Created, &p.Updated); err != nil {
		return nil, err
	}
	return p, nil
}
func (r *PermissionRepo) Update(ctx context.Context, p *types.Permission) (*types.Permission, error) {
	row := r.db.QueryRow(ctx, `UPDATE permissions SET name=$1, code=$2, system_module=$3, updated=now() WHERE id=$4 RETURNING updated`,
		p.Name, p.Code, p.SystemModule, p.ID)
	if err := row.Scan(&p.Updated); err != nil {
		return nil, err
	}
	return p, nil
}
func (r *PermissionRepo) Delete(ctx context.Context, id int32) error {
	_, err := r.db.Exec(ctx, `DELETE FROM permissions WHERE id=$1`, id)
	return err
}
