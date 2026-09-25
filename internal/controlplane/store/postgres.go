package store

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
	"github.com/behaviorengineering/wonderfeed/internal/controlplane"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Clock returns the current time; inject for tests.
type Clock func() time.Time

// PostgresStore is the PostgreSQL implementation of controlplane.Store.
type PostgresStore struct {
	pool  *pgxpool.Pool
	clock Clock
}

// NewPostgresStore builds a store. Panics on nil pool or clock.
func NewPostgresStore(pool *pgxpool.Pool, clock Clock) *PostgresStore {
	if pool == nil {
		panic("store.NewPostgresStore: pool is nil")
	}
	if clock == nil {
		panic("store.NewPostgresStore: clock is nil")
	}
	return &PostgresStore{pool: pool, clock: clock}
}

var _ controlplane.Store = (*PostgresStore)(nil)

// Create inserts a child profile with desired policy and sync_pending status.
func (s *PostgresStore) Create(ctx context.Context, name, avatarColor string, policy controlplane.ChildPolicy) (controlplane.ChildProfile, error) {
	const op = "store.PostgresStore.Create"
	if err := requireCtx(ctx, op); err != nil {
		return controlplane.ChildProfile{}, err
	}
	if err := controlplane.ValidateCreateName(name); err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeInvalid, op, "validate name")
	}
	if err := controlplane.ValidatePolicy(policy); err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeInvalid, op, "validate policy")
	}
	if avatarColor == "" {
		avatarColor = "#7c5cff"
	}
	now := s.clock().UTC()
	id, err := newProfileID()
	if err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "generate id")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeUnavailable, op, "begin")
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
INSERT INTO wonderfeed.wf_child_profiles (id, name, avatar_color, provider_profile_id, version, created_at, updated_at)
VALUES ($1, $2, $3, '', 1, $4, $4)`,
		id, name, avatarColor, now,
	)
	if err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "insert profile")
	}
	_, err = tx.Exec(ctx, `
INSERT INTO wonderfeed.wf_child_policies (
  child_profile_id, daily_minutes, local_only, hide_shorts, hide_live, downloads_only,
  bedtime_start, bedtime_end, version, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 1, $9)`,
		id,
		policy.DailyMinutes,
		policy.LocalOnly,
		policy.HideShorts,
		policy.HideLive,
		policy.DownloadsOnly,
		policy.BedtimeStart,
		policy.BedtimeEnd,
		now,
	)
	if err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "insert policy")
	}
	_, err = tx.Exec(ctx, `
INSERT INTO wonderfeed.wf_policy_sync_events (child_profile_id, status, sync_error, created_at)
VALUES ($1, $2, '', $3)`,
		id, string(controlplane.SyncPending), now,
	)
	if err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "insert sync event")
	}
	if err := tx.Commit(ctx); err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "commit")
	}
	return s.Get(ctx, id)
}

// List returns all child profiles ordered by created_at.
func (s *PostgresStore) List(ctx context.Context) ([]controlplane.ChildProfile, error) {
	const op = "store.PostgresStore.List"
	if err := requireCtx(ctx, op); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, profileSelect+" ORDER BY p.created_at ASC")
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeFailed, op, "query")
	}
	defer rows.Close()
	var out []controlplane.ChildProfile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, apperr.Wrap(err, apperr.CodeFailed, op, "scan")
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Wrap(err, apperr.CodeFailed, op, "rows")
	}
	if out == nil {
		out = []controlplane.ChildProfile{}
	}
	return out, nil
}

// Get loads one child profile by host ID.
func (s *PostgresStore) Get(ctx context.Context, id string) (controlplane.ChildProfile, error) {
	const op = "store.PostgresStore.Get"
	if err := requireCtx(ctx, op); err != nil {
		return controlplane.ChildProfile{}, err
	}
	row := s.pool.QueryRow(ctx, profileSelect+` WHERE p.id = $1`, id)
	p, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return controlplane.ChildProfile{}, apperr.New(apperr.CodeNotFound, op, "child profile not found").With("id", id)
	}
	if err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "query").With("id", id)
	}
	return p, nil
}

// ListAllowlist returns the host-owned approved channels for a child.
func (s *PostgresStore) ListAllowlist(ctx context.Context, id string) ([]controlplane.AllowlistChannel, error) {
	const op = "store.PostgresStore.ListAllowlist"
	if err := requireCtx(ctx, op); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
SELECT channel_id, title, url, added_at
FROM wonderfeed.wf_child_allowlist_channels
WHERE child_profile_id = $1
ORDER BY added_at ASC, channel_id ASC`, id)
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeFailed, op, "query").With("id", id)
	}
	defer rows.Close()
	out := []controlplane.AllowlistChannel{}
	for rows.Next() {
		var channel controlplane.AllowlistChannel
		if err := rows.Scan(&channel.ChannelID, &channel.Title, &channel.URL, &channel.AddedAt); err != nil {
			return nil, apperr.Wrap(err, apperr.CodeFailed, op, "scan").With("id", id)
		}
		out = append(out, channel)
	}
	if err := rows.Err(); err != nil {
		return nil, apperr.Wrap(err, apperr.CodeFailed, op, "rows").With("id", id)
	}
	return out, nil
}

// ReplaceAllowlist replaces a child's host-owned channels with optimistic locking.
func (s *PostgresStore) ReplaceAllowlist(ctx context.Context, id string, expectedVersion int64, channels []controlplane.AllowlistChannel) (controlplane.ChildProfile, error) {
	const op = "store.PostgresStore.ReplaceAllowlist"
	if err := requireCtx(ctx, op); err != nil {
		return controlplane.ChildProfile{}, err
	}
	normalized, err := controlplane.NormalizeAllowlist(channels)
	if err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeInvalid, op, "validate allowlist")
	}
	now := s.clock().UTC()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeUnavailable, op, "begin")
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
UPDATE wonderfeed.wf_child_profiles
SET allowlist_version = allowlist_version + 1, updated_at = $1
WHERE id = $2 AND allowlist_version = $3`,
		now, id, expectedVersion,
	)
	if err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "update version").With("id", id)
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM wonderfeed.wf_child_profiles WHERE id = $1)`, id).Scan(&exists); err != nil {
			return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "check exists").With("id", id)
		}
		if !exists {
			return controlplane.ChildProfile{}, apperr.New(apperr.CodeNotFound, op, "child profile not found").With("id", id)
		}
		return controlplane.ChildProfile{}, apperr.New(apperr.CodeConflict, op, "allowlist version conflict").With("id", id)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM wonderfeed.wf_child_allowlist_channels WHERE child_profile_id = $1`, id); err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "clear allowlist").With("id", id)
	}
	for _, channel := range normalized {
		if _, err := tx.Exec(ctx, `
INSERT INTO wonderfeed.wf_child_allowlist_channels (child_profile_id, channel_id, title, url, added_at)
VALUES ($1, $2, $3, $4, $5)`,
			id, channel.ChannelID, channel.Title, channel.URL, now,
		); err != nil {
			return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "insert allowlist channel").
				With("id", id).With("channel_id", channel.ChannelID)
		}
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO wonderfeed.wf_policy_sync_events (child_profile_id, status, sync_error, created_at)
VALUES ($1, $2, '', $3)`,
		id, string(controlplane.SyncPending), now,
	); err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "insert sync event").With("id", id)
	}
	if err := tx.Commit(ctx); err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "commit").With("id", id)
	}
	return s.Get(ctx, id)
}

// UpdatePolicy applies an optimistic-lock policy update and marks sync_pending.
func (s *PostgresStore) UpdatePolicy(ctx context.Context, id string, expectedVersion int64, policy controlplane.ChildPolicy) (controlplane.ChildProfile, error) {
	const op = "store.PostgresStore.UpdatePolicy"
	if err := requireCtx(ctx, op); err != nil {
		return controlplane.ChildProfile{}, err
	}
	if err := controlplane.ValidatePolicy(policy); err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeInvalid, op, "validate policy")
	}
	now := s.clock().UTC()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeUnavailable, op, "begin")
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `
UPDATE wonderfeed.wf_child_policies SET
  daily_minutes = $1,
  local_only = $2,
  hide_shorts = $3,
  hide_live = $4,
  downloads_only = $5,
  bedtime_start = $6,
  bedtime_end = $7,
  version = version + 1,
  updated_at = $8
WHERE child_profile_id = $9 AND version = $10`,
		policy.DailyMinutes,
		policy.LocalOnly,
		policy.HideShorts,
		policy.HideLive,
		policy.DownloadsOnly,
		policy.BedtimeStart,
		policy.BedtimeEnd,
		now,
		id,
		expectedVersion,
	)
	if err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "update policy").With("id", id)
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM wonderfeed.wf_child_profiles WHERE id = $1)`, id).Scan(&exists); err != nil {
			return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "check exists").With("id", id)
		}
		if !exists {
			return controlplane.ChildProfile{}, apperr.New(apperr.CodeNotFound, op, "child profile not found").With("id", id)
		}
		return controlplane.ChildProfile{}, apperr.New(apperr.CodeConflict, op, "policy version conflict").
			With("id", id)
	}
	_, err = tx.Exec(ctx, `
UPDATE wonderfeed.wf_child_profiles SET version = version + 1, updated_at = $1 WHERE id = $2`,
		now, id,
	)
	if err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "bump profile version").With("id", id)
	}
	_, err = tx.Exec(ctx, `
INSERT INTO wonderfeed.wf_policy_sync_events (child_profile_id, status, sync_error, created_at)
VALUES ($1, $2, '', $3)`,
		id, string(controlplane.SyncPending), now,
	)
	if err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "insert sync event").With("id", id)
	}
	if err := tx.Commit(ctx); err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "commit").With("id", id)
	}
	return s.Get(ctx, id)
}

// SetProviderProfileID stores the provider-side profile identifier.
func (s *PostgresStore) SetProviderProfileID(ctx context.Context, id, providerProfileID string) (controlplane.ChildProfile, error) {
	const op = "store.PostgresStore.SetProviderProfileID"
	if err := requireCtx(ctx, op); err != nil {
		return controlplane.ChildProfile{}, err
	}
	now := s.clock().UTC()
	tag, err := s.pool.Exec(ctx, `
UPDATE wonderfeed.wf_child_profiles
SET provider_profile_id = $1, updated_at = $2
WHERE id = $3`,
		providerProfileID, now, id,
	)
	if err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "update provider id").With("id", id)
	}
	if tag.RowsAffected() == 0 {
		return controlplane.ChildProfile{}, apperr.New(apperr.CodeNotFound, op, "child profile not found").With("id", id)
	}
	return s.Get(ctx, id)
}

// RecordSyncStatus appends a sync event for the child profile.
func (s *PostgresStore) RecordSyncStatus(ctx context.Context, id string, status controlplane.SyncStatus, syncErr string) (controlplane.ChildProfile, error) {
	const op = "store.PostgresStore.RecordSyncStatus"
	if err := requireCtx(ctx, op); err != nil {
		return controlplane.ChildProfile{}, err
	}
	switch status {
	case controlplane.SyncPending, controlplane.SyncSynced, controlplane.SyncFailed:
	default:
		return controlplane.ChildProfile{}, apperr.New(apperr.CodeInvalid, op, "invalid sync status").
			With("status", string(status))
	}
	now := s.clock().UTC()
	tag, err := s.pool.Exec(ctx, `
INSERT INTO wonderfeed.wf_policy_sync_events (child_profile_id, status, sync_error, created_at)
SELECT $1, $2, $3, $4
WHERE EXISTS (SELECT 1 FROM wonderfeed.wf_child_profiles WHERE id = $1)`,
		id, string(status), syncErr, now,
	)
	if err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "insert sync event").With("id", id)
	}
	if tag.RowsAffected() == 0 {
		return controlplane.ChildProfile{}, apperr.New(apperr.CodeNotFound, op, "child profile not found").With("id", id)
	}
	if _, err := s.pool.Exec(ctx, `UPDATE wonderfeed.wf_child_profiles SET updated_at = $1 WHERE id = $2`, now, id); err != nil {
		return controlplane.ChildProfile{}, apperr.Wrap(err, apperr.CodeFailed, op, "touch profile").With("id", id)
	}
	return s.Get(ctx, id)
}

const profileSelect = `
SELECT
  p.id::text,
  p.name,
  p.avatar_color,
  p.provider_profile_id,
  p.version,
  p.allowlist_version,
  p.created_at,
  p.updated_at,
  pol.daily_minutes,
  pol.local_only,
  pol.hide_shorts,
  pol.hide_live,
  pol.downloads_only,
  pol.bedtime_start,
  pol.bedtime_end,
  pol.version,
  pol.updated_at,
  COALESCE(ev.status, 'sync_pending'),
  COALESCE(ev.sync_error, '')
FROM wonderfeed.wf_child_profiles p
JOIN wonderfeed.wf_child_policies pol ON pol.child_profile_id = p.id
LEFT JOIN LATERAL (
  SELECT status, sync_error
  FROM wonderfeed.wf_policy_sync_events
  WHERE child_profile_id = p.id
  ORDER BY created_at DESC, id DESC
  LIMIT 1
) ev ON TRUE`

type scannable interface {
	Scan(dest ...any) error
}

func scanProfile(row scannable) (controlplane.ChildProfile, error) {
	var p controlplane.ChildProfile
	var status string
	err := row.Scan(
		&p.ID,
		&p.Name,
		&p.AvatarColor,
		&p.ProviderProfileID,
		&p.Version,
		&p.AllowlistVersion,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.Policy.DailyMinutes,
		&p.Policy.LocalOnly,
		&p.Policy.HideShorts,
		&p.Policy.HideLive,
		&p.Policy.DownloadsOnly,
		&p.Policy.BedtimeStart,
		&p.Policy.BedtimeEnd,
		&p.Policy.Version,
		&p.Policy.UpdatedAt,
		&status,
		&p.SyncError,
	)
	if err != nil {
		return controlplane.ChildProfile{}, err
	}
	p.SyncStatus = controlplane.SyncStatus(status)
	return p, nil
}

func requireCtx(ctx context.Context, op string) error {
	if ctx == nil {
		return apperr.New(apperr.CodeInvalid, op, "context is required")
	}
	if _, ok := ctx.Deadline(); !ok {
		return apperr.New(apperr.CodeInvalid, op, "outbound: missing deadline")
	}
	return nil
}

func newProfileID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	hexed := hex.EncodeToString(b[:])
	return hexed[0:8] + "-" + hexed[8:12] + "-" + hexed[12:16] + "-" + hexed[16:20] + "-" + hexed[20:32], nil
}
