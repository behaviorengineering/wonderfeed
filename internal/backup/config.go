// Package backup creates, lists, and restores encrypted Wonderfeed home-state backups.
//
// A backup holds a PostgreSQL dump plus small portable provider files. It excludes
// video libraries, image caches, logs, and cookie/secret dirs by default.
package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
	"github.com/behaviorengineering/wonderfeed/internal/provider/ytzero"
)

const (
	// RelBackupDir is the host backup root relative to the repo.
	RelBackupDir = "data/backups"
	// RelBackupObjects is where encrypted local objects land.
	RelBackupObjects = "data/backups/objects"
	// RelBackupSafety is where pre-restore safety snapshots land.
	RelBackupSafety = "data/backups/safety"
	// RelAgeIdentity is the default age identity path (gitignored).
	RelAgeIdentity = "data/backups/age-identity.txt"
	// ManifestName is the JSON document inside the plaintext archive.
	ManifestName = "manifest.json"
	// DumpRelPath is the pg_dump path inside the archive.
	DumpRelPath = "postgres/dump.sql"
	// ArchiveVersion is the on-disk backup format version.
	ArchiveVersion = 1
)

// Paths holds resolved host paths for backup operations.
type Paths struct {
	Root         string
	BackupDir    string
	ObjectsDir   string
	SafetyDir    string
	AgeIdentity  string
	ProviderData string
	EnvFile      string
	ComposeFile  string
}

// Config is operator configuration for create/list/restore.
type Config struct {
	Paths Paths
	// AgeIdentityPath overrides Paths.AgeIdentity when non-empty.
	AgeIdentityPath string
	// LocalOnly skips remote upload even when S3 env is set.
	LocalOnly bool
	// ConfirmRestore must be true for Restore.
	ConfirmRestore bool
	Postgres       PostgresConfig
	S3             S3Config
}

// PostgresConfig selects the dump target inside the compose project.
type PostgresConfig struct {
	Container string
	User      string
	Database  string
}

// S3Config is optional S3-compatible object storage (R2/B2/MinIO).
type S3Config struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	Prefix    string
}

// CreateConfig builds Config from ytzero paths and process environment.
func CreateConfig(provider ytzero.Paths, env map[string]string) (Config, error) {
	const op = "backup.CreateConfig"
	if env == nil {
		env = environMap()
	}
	if provider.Root == "" {
		return Config{}, apperr.New(apperr.CodeInvalid, op, "provider root is empty")
	}
	cfg := Config{
		Paths: Paths{
			Root:         provider.Root,
			BackupDir:    filepath.Join(provider.Root, RelBackupDir),
			ObjectsDir:   filepath.Join(provider.Root, RelBackupObjects),
			SafetyDir:    filepath.Join(provider.Root, RelBackupSafety),
			AgeIdentity:  filepath.Join(provider.Root, RelAgeIdentity),
			ProviderData: provider.DataDir,
			EnvFile:      provider.EnvFile,
			ComposeFile:  provider.ComposeFile,
		},
		AgeIdentityPath: strings.TrimSpace(env["WONDERFEED_BACKUP_AGE_IDENTITY"]),
		LocalOnly:       truthy(env["WONDERFEED_BACKUP_LOCAL_ONLY"]),
		Postgres: PostgresConfig{
			Container: firstNonEmpty(env["WONDERFEED_BACKUP_POSTGRES_CONTAINER"], "wonderfeed-postgres"),
			User:      firstNonEmpty(env["POSTGRES_USER"], "ytzero"),
			Database:  firstNonEmpty(env["POSTGRES_DB"], "ytzero"),
		},
		S3: S3Config{
			Endpoint:  strings.TrimSpace(env["WONDERFEED_BACKUP_S3_ENDPOINT"]),
			Region:    firstNonEmpty(env["WONDERFEED_BACKUP_S3_REGION"], "auto"),
			Bucket:    strings.TrimSpace(env["WONDERFEED_BACKUP_S3_BUCKET"]),
			AccessKey: strings.TrimSpace(env["WONDERFEED_BACKUP_S3_ACCESS_KEY"]),
			SecretKey: strings.TrimSpace(env["WONDERFEED_BACKUP_S3_SECRET_KEY"]),
			Prefix:    strings.Trim(firstNonEmpty(env["WONDERFEED_BACKUP_S3_PREFIX"], "wonderfeed/backups"), "/"),
		},
	}
	if fileEnv, err := loadEnvFile(provider.EnvFile); err == nil {
		if cfg.Postgres.User == "ytzero" {
			if v := strings.TrimSpace(fileEnv["POSTGRES_USER"]); v != "" {
				cfg.Postgres.User = v
			}
		}
		if cfg.Postgres.Database == "ytzero" {
			if v := strings.TrimSpace(fileEnv["POSTGRES_DB"]); v != "" {
				cfg.Postgres.Database = v
			}
		}
	}
	return cfg, nil
}

// AgeIdentityFile returns the identity path actually used.
func (c Config) AgeIdentityFile() string {
	if c.AgeIdentityPath != "" {
		return c.AgeIdentityPath
	}
	return c.Paths.AgeIdentity
}

// EnsureDirs creates backup directories.
func (c Config) EnsureDirs() error {
	const op = "backup.EnsureDirs"
	for _, dir := range []string{c.Paths.BackupDir, c.Paths.ObjectsDir, c.Paths.SafetyDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return apperr.Wrap(err, apperr.CodeFailed, op, "create backup directory").With("path", dir)
		}
	}
	return nil
}

func environMap() map[string]string {
	out := map[string]string{}
	for _, kv := range os.Environ() {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		out[k] = v
	}
	return out
}

func loadEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func wrap(err error, op, message string) *apperr.Error {
	return apperr.Wrap(err, apperr.CodeFailed, op, message)
}

func requireNonEmpty(op, name, value string) error {
	if strings.TrimSpace(value) == "" {
		return apperr.New(apperr.CodeInvalid, op, fmt.Sprintf("%s is required", name))
	}
	return nil
}
