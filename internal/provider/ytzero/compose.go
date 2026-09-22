package ytzero

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

const defaultDockerProbe = 15 * time.Second

func wrap(err error, op, message string) *apperr.Error {
	return apperr.Wrap(err, apperr.CodeFailed, op, message)
}

// EnsureEnvFile copies the example env when deploy/ytzero/.env is missing.
func EnsureEnvFile(paths Paths) error {
	const op = "ytzero.EnsureEnvFile"
	if _, err := os.Stat(paths.EnvFile); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return apperr.Wrap(err, apperr.CodeFailed, op, "stat env file").With("path", paths.EnvFile)
	}
	data, err := os.ReadFile(paths.EnvExample)
	if err != nil {
		return apperr.Wrap(err, apperr.CodeNotFound, op, "read env example").With("path", paths.EnvExample)
	}
	if err := os.WriteFile(paths.EnvFile, data, 0o600); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "write env file").With("path", paths.EnvFile)
	}
	return nil
}

// EnsureDataDir creates the host data directory for the provider volume.
func EnsureDataDir(paths Paths) error {
	const op = "ytzero.EnsureDataDir"
	if err := os.MkdirAll(paths.DataDir, 0o755); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "create data directory").With("path", paths.DataDir)
	}
	return nil
}

// ProbeDocker checks that the Docker daemon answers within timeout.
func ProbeDocker(ctx context.Context, timeout time.Duration) error {
	const op = "ytzero.ProbeDocker"
	if timeout <= 0 {
		timeout = defaultDockerProbe
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(probeCtx, "docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		code := apperr.CodeUnavailable
		if probeCtx.Err() != nil {
			return apperr.Wrap(err, code, op, "docker daemon probe timed out")
		}
		return apperr.Wrap(err, code, op, "docker daemon unavailable")
	}
	return nil
}

// Up starts the host YT Zero compose project.
func Up(ctx context.Context, paths Paths) error {
	const op = "ytzero.Up"
	if err := EnsureDataDir(paths); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "ensure data dir")
	}
	if err := EnsureEnvFile(paths); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "ensure env file")
	}
	if err := ProbeDocker(ctx, defaultDockerProbe); err != nil {
		return apperr.Wrap(err, apperr.CodeUnavailable, op, "docker preflight")
	}
	if err := runCompose(ctx, paths, "up", "-d", "--remove-orphans"); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "compose up")
	}
	return nil
}

// Down stops the host YT Zero compose project without removing volumes.
func Down(ctx context.Context, paths Paths) error {
	const op = "ytzero.Down"
	if err := ProbeDocker(ctx, defaultDockerProbe); err != nil {
		return apperr.Wrap(err, apperr.CodeUnavailable, op, "docker preflight")
	}
	if err := runCompose(ctx, paths, "down", "--remove-orphans"); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "compose down")
	}
	return nil
}

// ComposePS returns docker compose ps output for the overlay project.
func ComposePS(ctx context.Context, paths Paths) (string, error) {
	const op = "ytzero.ComposePS"
	if err := ProbeDocker(ctx, defaultDockerProbe); err != nil {
		return "", apperr.Wrap(err, apperr.CodeUnavailable, op, "docker preflight")
	}
	out, err := runComposeOutput(ctx, paths, "ps")
	if err != nil {
		return "", apperr.Wrap(err, apperr.CodeFailed, op, "compose ps")
	}
	return out, nil
}

func runCompose(ctx context.Context, paths Paths, args ...string) error {
	cmd := composeCommand(ctx, paths, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose %v: %w", args, err)
	}
	return nil
}

func runComposeOutput(ctx context.Context, paths Paths, args ...string) (string, error) {
	cmd := composeCommand(ctx, paths, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("docker compose %v: %w", args, err)
	}
	return string(out), nil
}

func composeCommand(ctx context.Context, paths Paths, args ...string) *exec.Cmd {
	full := []string{
		"compose",
		"-p", ComposeProject,
		"-f", paths.ComposeFile,
	}
	full = append(full, args...)
	cmd := exec.CommandContext(ctx, "docker", full...)
	cmd.Dir = filepath.Dir(paths.ComposeFile)
	return cmd
}
