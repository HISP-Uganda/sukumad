// internal/migrate/migrate.go
package migrate

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	// mgpostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

//go:embed ../../db/migrations/*.sql
var migrationsFS embed.FS

//go:embed ../../db/fixtures/*.sql
var fixturesFS embed.FS

//func Run(ctx context.Context, dbURL string) error {
//	// Build pgx ConnConfig then open *sql.DB via stdlib
//	cfg, err := pgx.ParseConfig(dbURL)
//	if err != nil {
//		return err
//	}
//	sqlDB := stdlib.OpenDB(*cfg)
//	defer sqlDB.Close()
//
//	src, err := iofs.New(migrationsFS, "db/migrations")
//	if err != nil {
//		return fmt.Errorf("iofs source: %w", err)
//	}
//
//	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
//	if err != nil {
//		return fmt.Errorf("postgres driver: %w", err)
//	}
//
//	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
//	if err != nil {
//		return fmt.Errorf("migrate instance: %w", err)
//	}
//	defer m.Close()
//
//	if err := m.Up(); err != migrate.ErrNoChange && err != nil {
//		return err
//	}
//	return nil
//}

func RunWithPostgresDriver(ctx context.Context, dbURL string) (*sql.DB, error) {
	// Build pgx config & wrap as *sql.DB
	cfg, err := pgx.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("parse db url: %w", err)
	}
	sqlDB := stdlib.OpenDB(*cfg)

	// Optionally ping with context to fail early
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("db ping: %w", err)
	}

	src, err := iofs.New(migrationsFS, "db/migrations")
	if err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("iofs source: %w", err)
	}
	// Note: src has no Close for postgres path (iofs.Source has Close(), but it's fine to keep symmetrical)
	defer src.Close()

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("postgres driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate up: %w", err)
	}
	return sqlDB, nil
}

//func LoadFixtures(ctx context.Context, db *sql.DB) error {
//	entries, err := fixturesFS.ReadDir("db/fixtures")
//	if err != nil {
//		return fmt.Errorf("read fixtures dir: %w", err)
//	}
//	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
//
//	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
//	if err != nil {
//		return fmt.Errorf("begin tx: %w", err)
//	}
//	defer func() {
//		if err != nil {
//			_ = tx.Rollback()
//		}
//	}()
//
//	for _, e := range entries {
//		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
//			continue
//		}
//		sqlBytes, rerr := fixturesFS.ReadFile("db/fixtures/" + e.Name())
//		if rerr != nil {
//			err = fmt.Errorf("read fixture %s: %w", e.Name(), rerr)
//			return err
//		}
//		if _, execErr := tx.ExecContext(ctx, string(sqlBytes)); execErr != nil {
//			err = fmt.Errorf("exec fixture %s: %w", e.Name(), execErr)
//			return err
//		}
//	}
//
//	if cerr := tx.Commit(); cerr != nil {
//		return fmt.Errorf("commit fixtures: %w", cerr)
//	}
//	return nil
//}

func LoadFixturesWithPostgresDriver(ctx context.Context, db *sql.DB) error {
	entries, err := fixturesFS.ReadDir("db/fixtures")
	if err != nil {
		return fmt.Errorf("read fixtures dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	tx, err := db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		sqlBytes, rerr := fixturesFS.ReadFile("db/fixtures/" + e.Name())
		if rerr != nil {
			err = fmt.Errorf("read fixture %s: %w", e.Name(), rerr)
			return err
		}
		if _, execErr := tx.ExecContext(ctx, string(sqlBytes)); execErr != nil {
			err = fmt.Errorf("exec fixture %s: %w", e.Name(), execErr)
			return err
		}
	}

	if cerr := tx.Commit(); cerr != nil {
		return fmt.Errorf("commit fixtures: %w", cerr)
	}
	return nil
}
