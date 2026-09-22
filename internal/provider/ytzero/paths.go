// Package ytzero operates the pinned YT Zero provider from the Wonderfeed host.
package ytzero

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// RelComposeFile is the host compose overlay relative to the repo root.
	RelComposeFile = "deploy/ytzero/compose.yaml"
	// RelEnvFile is the optional host env file relative to the repo root.
	RelEnvFile = "deploy/ytzero/.env"
	// RelEnvExample is the committed env template relative to the repo root.
	RelEnvExample = "deploy/ytzero/.env.example"
	// RelDataDir is the host data volume relative to the repo root.
	RelDataDir = "data/ytzero"
	// RelProviderDir is the git submodule path relative to the repo root.
	RelProviderDir = "providers/ytzero"
	// ComposeProject names the docker compose project.
	ComposeProject = "wonderfeed-ytzero"
	// DefaultBaseURL is the local provider UI and API origin.
	DefaultBaseURL = "http://127.0.0.1:3001"
	// HealthPath is the unauthenticated health endpoint.
	HealthPath = "/api/health"
)

// Paths resolves host-owned paths for the YT Zero overlay.
type Paths struct {
	Root        string
	ComposeFile string
	EnvFile     string
	EnvExample  string
	DataDir     string
	ProviderDir string
}

// ResolvePaths walks upward from start for go.mod and returns provider paths.
func ResolvePaths(start string) (Paths, error) {
	const op = "ytzero.ResolvePaths"
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return Paths{}, wrap(err, op, "get working directory")
		}
	}
	root, err := findRepoRoot(start)
	if err != nil {
		return Paths{}, wrap(err, op, "find repository root")
	}
	return Paths{
		Root:        root,
		ComposeFile: filepath.Join(root, RelComposeFile),
		EnvFile:     filepath.Join(root, RelEnvFile),
		EnvExample:  filepath.Join(root, RelEnvExample),
		DataDir:     filepath.Join(root, RelDataDir),
		ProviderDir: filepath.Join(root, RelProviderDir),
	}, nil
}

func findRepoRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found from %s", start)
		}
		dir = parent
	}
}
