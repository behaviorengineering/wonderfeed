package backup

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

// DumpPostgres runs pg_dump inside the configured Docker container.
func DumpPostgres(ctx context.Context, cfg PostgresConfig) ([]byte, error) {
	const op = "backup.DumpPostgres"
	if err := requireNonEmpty(op, "container", cfg.Container); err != nil {
		return nil, err
	}
	if err := requireNonEmpty(op, "user", cfg.User); err != nil {
		return nil, err
	}
	if err := requireNonEmpty(op, "database", cfg.Database); err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "docker", "exec", cfg.Container,
		"pg_dump",
		"-U", cfg.User,
		"-d", cfg.Database,
		"--no-owner",
		"--no-acl",
		"--format=plain",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, apperr.Wrap(fmt.Errorf("%s", msg), apperr.CodeUnavailable, op, "pg_dump failed")
	}
	if stdout.Len() == 0 {
		return nil, apperr.New(apperr.CodeFailed, op, "pg_dump returned empty output")
	}
	return stdout.Bytes(), nil
}

// RestorePostgres loads a plain SQL dump into the configured database.
// It drops and recreates the public schema first so restore is replace-oriented.
func RestorePostgres(ctx context.Context, cfg PostgresConfig, dump []byte) error {
	const op = "backup.RestorePostgres"
	if err := requireNonEmpty(op, "container", cfg.Container); err != nil {
		return err
	}
	reset := fmt.Sprintf(
		"DROP SCHEMA public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO %s; GRANT ALL ON SCHEMA public TO public;",
		pqQuoteIdent(cfg.User),
	)
	resetCmd := exec.CommandContext(ctx, "docker", "exec", "-i", cfg.Container,
		"psql", "-U", cfg.User, "-d", cfg.Database, "-v", "ON_ERROR_STOP=1", "-c", reset,
	)
	var resetErr bytes.Buffer
	resetCmd.Stderr = &resetErr
	if err := resetCmd.Run(); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "reset public schema").With("detail", strings.TrimSpace(resetErr.String()))
	}

	loadCmd := exec.CommandContext(ctx, "docker", "exec", "-i", cfg.Container,
		"psql", "-U", cfg.User, "-d", cfg.Database, "-v", "ON_ERROR_STOP=1", "-q",
	)
	loadCmd.Stdin = bytes.NewReader(dump)
	var loadOut, loadErr bytes.Buffer
	loadCmd.Stdout = &loadOut
	loadCmd.Stderr = &loadErr
	if err := loadCmd.Run(); err != nil {
		detail := strings.TrimSpace(loadErr.String())
		if detail == "" {
			detail = strings.TrimSpace(loadOut.String())
		}
		return apperr.Wrap(fmt.Errorf("%s", detail), apperr.CodeFailed, op, "psql restore failed")
	}
	return nil
}

func pqQuoteIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}
