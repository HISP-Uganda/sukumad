package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"sukumad/internal/types"
)

type RoleRepo struct{ db *pgxpool.Pool }

func NewRoleRepo(db *pgxpool.Pool) *RoleRepo { return &RoleRepo{db: db} }

func (r *RoleRepo) List(ctx context.Context) ([]types.Role, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, description, created, updated FROM user_roles ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []types.Role
	for rows.Next() {
		var v types.Role
		if err := rows.Scan(&v.ID, &v.Name, &v.Description, &v.Created, &v.Updated); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *RoleRepo) Create(ctx context.Context, v *types.Role) (*types.Role, error) {
	row := r.db.QueryRow(ctx, `INSERT INTO user_roles(name, description) VALUES ($1,$2) RETURNING id, created, updated`, v.Name, v.Description)
	if err := row.Scan(&v.ID, &v.Created, &v.Updated); err != nil {
		return nil, err
	}
	return v, nil
}
func (r *RoleRepo) Update(ctx context.Context, v *types.Role) (*types.Role, error) {
	row := r.db.QueryRow(ctx, `UPDATE user_roles SET name=$1, description=$2, updated=now() WHERE id=$3 RETURNING updated`, v.Name, v.Description, v.ID)
	if err := row.Scan(&v.Updated); err != nil {
		return nil, err
	}
	return v, nil
}
func (r *RoleRepo) Delete(ctx context.Context, id int32) error {
	_, err := r.db.Exec(ctx, `DELETE FROM user_roles WHERE id=$1`, id)
	return err
}
func (r *RoleRepo) GrantRolePermission(ctx context.Context, roleID, permID int32) error {
	_, err := r.db.Exec(ctx, `INSERT INTO user_role_permissions(role_id, permission_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, roleID, permID)
	return err
}
func (r *RoleRepo) RevokeRolePermission(ctx context.Context, roleID, permID int32) error {
	_, err := r.db.Exec(ctx, `DELETE FROM user_role_permissions WHERE role_id=$1 AND permission_id=$2`, roleID, permID)
	return err
}
