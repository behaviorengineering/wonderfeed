// Package store persists Wonderfeed control-plane child profiles and policies.
package store

import (
	"context"
	"embed"
	"io/fs"
	"sort"
	"strings"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// MigrateUp applies pending .up.sql migrations in lexical order.
func MigrateUp(ctx context.Context, pool *pgxpool.Pool) error {
	const op = "store.MigrateUp"
	if pool == nil {
		return apperr.New(apperr.CodeInvalid, op, "pool is nil")
	}
	if ctx == nil {
		return apperr.New(apperr.CodeInvalid, op, "context is required")
	}
	if _, ok := ctx.Deadline(); !ok {
		return apperr.New(apperr.CodeInvalid, op, "outbound: missing deadline")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return apperr.Wrap(err, apperr.CodeUnavailable, op, "begin transaction")
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS wonderfeed`); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "create schema")
	}
	if _, err := tx.Exec(ctx, `
CREATE TABLE IF NOT EXISTS wonderfeed.schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "ensure schema_migrations")
	}

	names, err := listUpMigrations()
	if err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "list migrations")
	}
	for _, name := range names {
		version := strings.TrimSuffix(name, ".up.sql")
		var exists bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM wonderfeed.schema_migrations WHERE version = $1)`,
			version,
		).Scan(&exists); err != nil {
			return apperr.Wrap(err, apperr.CodeFailed, op, "check migration").With("version", version)
		}
		if exists {
			continue
		}
		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return apperr.Wrap(err, apperr.CodeFailed, op, "read migration").With("version", version)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			return apperr.Wrap(err, apperr.CodeFailed, op, "apply migration").With("version", version)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO wonderfeed.schema_migrations (version) VALUES ($1)`,
			version,
		); err != nil {
			return apperr.Wrap(err, apperr.CodeFailed, op, "record migration").With("version", version)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "commit migrations")
	}
	return nil
}

func listUpMigrations() ([]string, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".up.sql") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names, nil
}

// OpenPool creates a pgx pool from a database URL.
func OpenPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	const op = "store.OpenPool"
	if ctx == nil {
		return nil, apperr.New(apperr.CodeInvalid, op, "context is required")
	}
	if strings.TrimSpace(databaseURL) == "" {
		return nil, apperr.New(apperr.CodeInvalid, op, "database URL is required")
	}
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeInvalid, op, "parse database URL")
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeUnavailable, op, "open pool")
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, apperr.Wrap(err, apperr.CodeUnavailable, op, "ping database")
	}
	return pool, nil
}
