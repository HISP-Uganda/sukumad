package seeder

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
)

type Options struct {
	AdditionalPermCodes []string // e.g., []string{"can_view_deliveries","can_expand_requests"}
}

func Run(ctx context.Context, db *pgxpool.Pool, opts Options) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			_ = tx.Commit(ctx)
		}
	}()

	// Ensure Admin role
	var adminRoleID int32
	if err = tx.QueryRow(ctx, `
		INSERT INTO user_roles(name, description)
		VALUES ('Admin','Full administrative access')
		ON CONFLICT (name) DO UPDATE SET name=EXCLUDED.name
		RETURNING id`).Scan(&adminRoleID); err != nil {
		return err
	}

	// Ensure "admin" permission
	var adminPermID int32
	if err = tx.QueryRow(ctx, `
		INSERT INTO permissions(name, code, system_module)
		VALUES ('Admin', 'admin', 'System')
		ON CONFLICT (code) DO UPDATE SET code=EXCLUDED.code
		RETURNING id`).Scan(&adminPermID); err != nil {
		return err
	}

	// Grant "admin" to Admin role
	if _, err = tx.Exec(ctx, `
		INSERT INTO user_role_permissions(role_id, permission_id)
		VALUES ($1,$2) ON CONFLICT DO NOTHING`, adminRoleID, adminPermID); err != nil {
		return err
	}

	// Optionally ensure extra permissions exist and grant to Admin
	for _, code := range opts.AdditionalPermCodes {
		var pid int32
		if err = tx.QueryRow(ctx, `
			WITH upsert AS (
			  INSERT INTO permissions(name, code, system_module)
			  VALUES ($1, $2, $3)
			  ON CONFLICT (code) DO UPDATE SET code=EXCLUDED.code
			  RETURNING id
			)
			SELECT id FROM upsert`, prettify(code), code, moduleFrom(code)).Scan(&pid); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `
			INSERT INTO user_role_permissions(role_id, permission_id)
			VALUES ($1,$2) ON CONFLICT DO NOTHING`, adminRoleID, pid); err != nil {
			return err
		}
	}

	log.WithFields(log.Fields{"role_id": adminRoleID}).Info("seeder: admin bootstrap done")
	return nil
}

func prettify(s string) string { // "can_view_deliveries" -> "Can View Deliveries"
	out := make([]rune, 0, len(s))
	capNext := true
	for _, r := range s {
		if r == '_' {
			out = append(out, ' ')
			capNext = true
			continue
		}
		if capNext && r >= 'a' && r <= 'z' {
			r -= 32
			capNext = false
		}
		out = append(out, r)
	}
	return string(out)
}
func moduleFrom(code string) string {
	// crude mapping by prefix; adjust to your taxonomy
	switch {
	case hasPrefix(code, "can_view_deliveries"), hasPrefix(code, "can_expand"):
		return "Deliveries"
	case hasPrefix(code, "can_view_reporters"), hasPrefix(code, "can_edit_reporters"):
		return "Reporters"
	default:
		return "System"
	}
}
func hasPrefix(s, p string) bool { return len(s) >= len(p) && s[:len(p)] == p }
