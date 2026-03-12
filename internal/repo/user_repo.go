package repo

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sukumad/internal/types"
)

type UserRepo struct{ db *pgxpool.Pool }

func NewUserRepo(db *pgxpool.Pool) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) List(ctx context.Context, limit, offset int) ([]types.User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_role, firstname, lastname, username, telephone, email,
		       allowed_ips, denied_ips, failed_attempts, transaction_limit, is_active, is_system_user,
		       last_login, last_passwd_update, created, updated
		  FROM users
		  ORDER BY id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []types.User
	for rows.Next() {
		var u types.User
		if err := rows.Scan(
			&u.ID, &u.UserRole, &u.Firstname, &u.Lastname, &u.Username, &u.Telephone, &u.Email,
			&u.AllowedIPs, &u.DeniedIPs, &u.FailedAttempts, &u.TransactionLimit, &u.IsActive, &u.IsSystemUser,
			&u.LastLogin, &u.LastPasswdUpdate, &u.Created, &u.Updated,
		); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *UserRepo) Get(ctx context.Context, id int32) (*types.User, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, user_role, firstname, lastname, username, telephone, email,
		       allowed_ips, denied_ips, failed_attempts, transaction_limit, is_active, is_system_user,
		       last_login, last_passwd_update, created, updated
		  FROM users WHERE id=$1`, id)
	var u types.User
	if err := row.Scan(
		&u.ID, &u.UserRole, &u.Firstname, &u.Lastname, &u.Username, &u.Telephone, &u.Email,
		&u.AllowedIPs, &u.DeniedIPs, &u.FailedAttempts, &u.TransactionLimit, &u.IsActive, &u.IsSystemUser,
		&u.LastLogin, &u.LastPasswdUpdate, &u.Created, &u.Updated,
	); err != nil {
		return nil, err
	}
	return &u, nil
}

func hashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func (r *UserRepo) Create(ctx context.Context, v *types.User) (*types.User, error) {
	var pwHash string
	if v.Password != nil && *v.Password != "" {
		var err error
		pwHash, err = hashPassword(*v.Password)
		if err != nil {
			return nil, err
		}
	} else {
		// require password on create
		return nil, pgx.ErrNoRows
	}

	row := r.db.QueryRow(ctx, `
		INSERT INTO users (user_role, firstname, lastname, username, telephone, password_hash, email,
		                   allowed_ips, denied_ips, transaction_limit, is_active, is_system_user)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,COALESCE($11,true),COALESCE($12,false))
		RETURNING id, created, updated`,
		v.UserRole, v.Firstname, v.Lastname, v.Username, v.Telephone, pwHash, v.Email,
		v.AllowedIPs, v.DeniedIPs, v.TransactionLimit, v.IsActive, v.IsSystemUser,
	)

	if err := row.Scan(&v.ID, &v.Created, &v.Updated); err != nil {
		return nil, err
	}
	v.Password = nil
	return v, nil
}

func (r *UserRepo) Update(ctx context.Context, v *types.User) (*types.User, error) {
	var pwHash *string
	if v.Password != nil && *v.Password != "" {
		h, err := hashPassword(*v.Password)
		if err != nil {
			return nil, err
		}
		pwHash = &h
	}

	row := r.db.QueryRow(ctx, `
		UPDATE users SET
			user_role=$1, firstname=$2, lastname=$3, telephone=$4,
			password_hash = COALESCE($5,password_hash),
			email=$6, allowed_ips=$7, denied_ips=$8,
			transaction_limit=$9, is_active=COALESCE($10,is_active), is_system_user=COALESCE($11,is_system_user),
			updated=now()
		WHERE id=$12
		RETURNING updated`,
		v.UserRole, v.Firstname, v.Lastname, v.Telephone,
		pwHash, v.Email, v.AllowedIPs, v.DeniedIPs,
		v.TransactionLimit, v.IsActive, v.IsSystemUser,
		v.ID,
	)
	if err := row.Scan(&v.Updated); err != nil {
		return nil, err
	}
	v.Password = nil
	return v, nil
}

func (r *UserRepo) Delete(ctx context.Context, id int32) error {
	_, err := r.db.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
	return err
}

func (r *UserRepo) GrantUserPermission(ctx context.Context, userID, permID int32) error {
	_, err := r.db.Exec(ctx, `INSERT INTO user_permissions(user_id, permission_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`, userID, permID)
	return err
}
func (r *UserRepo) RevokeUserPermission(ctx context.Context, userID, permID int32) error {
	_, err := r.db.Exec(ctx, `DELETE FROM user_permissions WHERE user_id=$1 AND permission_id=$2`, userID, permID)
	return err
}
func (r *UserRepo) SetUserRole(ctx context.Context, userID, roleID int32) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET user_role=$2, updated=now() WHERE id=$1`, userID, roleID)
	return err
}
func (r *UserRepo) ListEffectivePermissions(ctx context.Context, userID int32) ([]types.Permission, error) {
	rows, err := r.db.Query(ctx, `
		WITH role_perms AS (
			SELECT rp.permission_id
			  FROM users u
			  JOIN user_role_permissions rp ON rp.role_id = u.user_role
			 WHERE u.id=$1
		), direct AS (
			SELECT up.permission_id
			  FROM user_permissions up
			 WHERE up.user_id=$1
		)
		SELECT p.id, p.name, p.code, p.system_module, p.created, p.updated
		  FROM permissions p
		 WHERE p.id IN (SELECT permission_id FROM role_perms UNION SELECT permission_id FROM direct)
		 ORDER BY p.system_module, p.code`, userID)
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

func (r *UserRepo) GetWithHashByUsername(ctx context.Context, username string) (*types.User, string, error) {
	row := r.db.QueryRow(ctx, `
        SELECT id, user_role, firstname, lastname, username, telephone, email,
               allowed_ips, denied_ips, failed_attempts, transaction_limit, is_active, is_system_user,
               last_login, last_failed_at, locked_until, last_passwd_update, created, updated, password_hash
          FROM users WHERE username=$1`, username)
	var u types.User
	var hash string
	if err := row.Scan(
		&u.ID, &u.UserRole, &u.Firstname, &u.Lastname, &u.Username, &u.Telephone, &u.Email,
		&u.AllowedIPs, &u.DeniedIPs, &u.FailedAttempts, &u.TransactionLimit, &u.IsActive, &u.IsSystemUser,
		&u.LastLogin, &u.LastFailedAt, &u.LockedUntil, &u.LastPasswdUpdate, &u.Created, &u.Updated, &hash,
	); err != nil {
		return nil, "", err
	}
	return &u, hash, nil
}

func (r *UserRepo) RecordLoginSuccess(ctx context.Context, userID int32) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET last_login=now(), failed_attempts=0, updated=now() WHERE id=$1`, userID)
	return err
}
func (r *UserRepo) RecordLoginFailure(ctx context.Context, username string) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET failed_attempts = failed_attempts + 1, updated=now() WHERE username=$1`, username)
	return err
}

func (r *UserRepo) RecordLoginFailureWithLockout(ctx context.Context, username string, now time.Time, threshold int, window, lockout time.Duration) error {
	// One-shot SQL that handles windowed counter + lockout
	_, err := r.db.Exec(ctx, `
		UPDATE users SET
		  failed_attempts = CASE
		    WHEN last_failed_at IS NULL OR (now() - last_failed_at) > $2 THEN 1         -- reset window
		    ELSE failed_attempts + 1
		  END,
		  last_failed_at  = now(),
		  locked_until    = CASE
		    WHEN (
		      CASE WHEN last_failed_at IS NULL OR (now() - last_failed_at) > $2 THEN 1 ELSE failed_attempts + 1 END
		    ) >= $1 THEN now() + $3
		    ELSE locked_until
		  END,
		  updated = now()
		WHERE username=$4
	`, threshold, window, lockout, username)
	return err
}

func (r *UserRepo) ClearFailuresAndUnlock(ctx context.Context, userID int32) error {
	_, err := r.db.Exec(ctx, `UPDATE users SET failed_attempts=0, last_failed_at=NULL, locked_until=NULL, updated=now() WHERE id=$1`, userID)
	return err
}

func (r *UserRepo) SetPassword(ctx context.Context, userID int32, newPlain string) error {
	h, err := bcrypt.GenerateFromPassword([]byte(newPlain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		UPDATE users SET password_hash=$2, last_passwd_update=now(), updated=now(),
		                failed_attempts=0, last_failed_at=NULL, locked_until=NULL
		WHERE id=$1`, userID, string(h))
	return err
}

// FindUserIDByUsernameOrEmail helper to find user by username/email
func (r *UserRepo) FindUserIDByUsernameOrEmail(ctx context.Context, key string) (int32, error) {
	var id int32
	err := r.db.QueryRow(ctx, `SELECT id FROM users WHERE username=$1 OR email=$1`, key).Scan(&id)
	return id, err
}

// UnlockAccount clears lockout state and failed attempts.
func (r *UserRepo) UnlockAccount(ctx context.Context, userID int32) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users
		   SET failed_attempts = 0,
		       last_failed_at  = NULL,
		       locked_until    = NULL,
		       updated         = now()
		 WHERE id = $1`, userID)
	return err
}
