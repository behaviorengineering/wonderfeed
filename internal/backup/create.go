package backup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

// CreateResult is returned after a successful backup create.
type CreateResult struct {
	ID           string   `json:"id"`
	LocalPath    string   `json:"localPath"`
	RemotePath   string   `json:"remotePath,omitempty"`
	AgeRecipient string   `json:"ageRecipient"`
	AgeCreated   bool     `json:"ageIdentityCreated"`
	Includes     []string `json:"includes"`
	Excludes     []string `json:"excludes"`
	SizeBytes    int64    `json:"sizeBytes"`
}

// Create runs pg_dump, packs portable state, encrypts with age, and stores locally
// (and optionally on S3 when configured and not LocalOnly).
func Create(ctx context.Context, cfg Config) (CreateResult, error) {
	const op = "backup.Create"
	if err := cfg.EnsureDirs(); err != nil {
		return CreateResult{}, err
	}
	identityPath := cfg.AgeIdentityFile()
	recipient, created, err := EnsureAgeIdentity(identityPath)
	if err != nil {
		return CreateResult{}, err
	}

	dump, err := DumpPostgres(ctx, cfg.Postgres)
	if err != nil {
		return CreateResult{}, err
	}

	now := time.Now().UTC()
	id := NewBackupID(now)
	host, _ := os.Hostname()
	manifest := Manifest{
		Version:   ArchiveVersion,
		ID:        id,
		CreatedAt: now,
		Hostname:  host,
		Excludes:  append([]string{}, DefaultExcludes...),
	}
	manifest.Postgres.Format = "plain"
	manifest.Postgres.Database = cfg.Postgres.Database
	manifest.Postgres.User = cfg.Postgres.User

	plain, includes, err := BuildArchiveTar(manifest, dump, cfg.Paths.ProviderData)
	if err != nil {
		return CreateResult{}, err
	}
	cipher, err := Encrypt(identityPath, plain)
	if err != nil {
		return CreateResult{}, err
	}

	local := LocalStore{Dir: cfg.Paths.ObjectsDir}
	localPath, err := local.Put(ctx, id, cipher)
	if err != nil {
		return CreateResult{}, err
	}

	result := CreateResult{
		ID:           id,
		LocalPath:    localPath,
		AgeRecipient: recipient,
		AgeCreated:   created,
		Includes:     includes,
		Excludes:     manifest.Excludes,
		SizeBytes:    int64(len(cipher)),
	}

	if !cfg.LocalOnly && S3Configured(cfg.S3) {
		remote, err := NewS3Store(cfg.S3)
		if err != nil {
			return CreateResult{}, err
		}
		loc, err := remote.Put(ctx, id, cipher)
		if err != nil {
			return CreateResult{}, apperr.Wrap(err, apperr.CodeFailed, op, "local backup saved but S3 upload failed").With("local", localPath)
		}
		result.RemotePath = loc
	}

	return result, nil
}

// List returns local objects, and remote objects when S3 is configured.
func List(ctx context.Context, cfg Config) ([]ObjectInfo, error) {
	if err := cfg.EnsureDirs(); err != nil {
		return nil, err
	}
	local := LocalStore{Dir: cfg.Paths.ObjectsDir}
	items, err := local.List(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.LocalOnly && S3Configured(cfg.S3) {
		remote, err := NewS3Store(cfg.S3)
		if err != nil {
			return nil, err
		}
		remoteItems, err := remote.List(ctx)
		if err != nil {
			return nil, err
		}
		seen := map[string]bool{}
		for _, it := range items {
			seen[it.ID] = true
		}
		for _, it := range remoteItems {
			if seen[it.ID] {
				continue
			}
			items = append(items, it)
		}
	}
	return items, nil
}

// RestoreOptions controls restore confirmation and source selection.
type RestoreOptions struct {
	ID      string
	Confirm bool
}

// RestoreResult summarizes a restore.
type RestoreResult struct {
	ID               string `json:"id"`
	SafetySnapshotID string `json:"safetySnapshotId"`
	RestoredFiles    int    `json:"restoredFiles"`
}

// Restore decrypts a backup, writes a safety snapshot of the current DB, then
// replaces Postgres and portable state files. Confirm must be true.
func Restore(ctx context.Context, cfg Config, opts RestoreOptions) (RestoreResult, error) {
	const op = "backup.Restore"
	if !opts.Confirm && !cfg.ConfirmRestore {
		return RestoreResult{}, apperr.New(apperr.CodeInvalid, op, "restore requires --confirm (destructive)")
	}
	if err := requireNonEmpty(op, "id", opts.ID); err != nil {
		return RestoreResult{}, err
	}
	if err := cfg.EnsureDirs(); err != nil {
		return RestoreResult{}, err
	}
	identityPath := cfg.AgeIdentityFile()
	if !identityLooksPresent(identityPath) {
		return RestoreResult{}, apperr.New(apperr.CodeNotFound, op, "age identity missing; cannot decrypt").With("path", identityPath)
	}

	cipher, err := loadCiphertext(ctx, cfg, opts.ID)
	if err != nil {
		return RestoreResult{}, err
	}
	plain, err := Decrypt(identityPath, cipher)
	if err != nil {
		return RestoreResult{}, err
	}
	manifest, dump, state, err := ReadArchiveTar(plain)
	if err != nil {
		return RestoreResult{}, err
	}

	safety, err := createSafetySnapshot(ctx, cfg)
	if err != nil {
		return RestoreResult{}, apperr.Wrap(err, apperr.CodeFailed, op, "safety snapshot before restore failed")
	}

	if err := RestorePostgres(ctx, cfg.Postgres, dump); err != nil {
		return RestoreResult{}, err
	}
	n, err := writeStateFiles(cfg.Paths.ProviderData, state)
	if err != nil {
		return RestoreResult{}, err
	}

	_ = manifest
	return RestoreResult{
		ID:               opts.ID,
		SafetySnapshotID: safety,
		RestoredFiles:    n,
	}, nil
}

func loadCiphertext(ctx context.Context, cfg Config, id string) ([]byte, error) {
	const op = "backup.loadCiphertext"
	local := LocalStore{Dir: cfg.Paths.ObjectsDir}
	data, err := local.Get(ctx, id)
	if err == nil {
		return data, nil
	}
	if !S3Configured(cfg.S3) || cfg.LocalOnly {
		return nil, err
	}
	remote, s3err := NewS3Store(cfg.S3)
	if s3err != nil {
		return nil, err
	}
	data, s3err = remote.Get(ctx, id)
	if s3err != nil {
		return nil, apperr.Wrap(s3err, apperr.CodeNotFound, op, "backup not found locally or on S3").With("id", id)
	}
	return data, nil
}

func createSafetySnapshot(ctx context.Context, cfg Config) (string, error) {
	const op = "backup.createSafetySnapshot"
	dump, err := DumpPostgres(ctx, cfg.Postgres)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	id := "safety-" + NewBackupID(now)
	host, _ := os.Hostname()
	manifest := Manifest{
		Version:   ArchiveVersion,
		ID:        id,
		CreatedAt: now,
		Hostname:  host,
		Excludes:  append([]string{}, DefaultExcludes...),
	}
	manifest.Postgres.Format = "plain"
	manifest.Postgres.Database = cfg.Postgres.Database
	manifest.Postgres.User = cfg.Postgres.User
	plain, _, err := BuildArchiveTar(manifest, dump, cfg.Paths.ProviderData)
	if err != nil {
		return "", err
	}
	cipher, err := Encrypt(cfg.AgeIdentityFile(), plain)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(cfg.Paths.SafetyDir, 0o755); err != nil {
		return "", wrap(err, op, "mkdir safety")
	}
	store := LocalStore{Dir: cfg.Paths.SafetyDir}
	if _, err := store.Put(ctx, id, cipher); err != nil {
		return "", err
	}
	return id, nil
}

func writeStateFiles(providerData string, state map[string][]byte) (int, error) {
	const op = "backup.writeStateFiles"
	if providerData == "" {
		return 0, nil
	}
	n := 0
	for name, data := range state {
		rel := strings.TrimPrefix(name, "state/")
		path := filepath.Join(providerData, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return n, wrap(err, op, "mkdir state").With("path", path)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			return n, wrap(err, op, "write state file").With("path", path)
		}
		n++
	}
	return n, nil
}
