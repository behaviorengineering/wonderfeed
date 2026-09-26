package ytzero

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
	"github.com/behaviorengineering/wonderfeed/internal/provider"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgAllowlistSync writes YT Zero subscription state directly in PostgreSQL.
// Experimental: couples to the provider schema in the shared household database.
type PgAllowlistSync struct {
	pool *pgxpool.Pool
}

// NewPgAllowlistSync builds a sync writer. Panics on nil pool.
func NewPgAllowlistSync(pool *pgxpool.Pool) *PgAllowlistSync {
	if pool == nil {
		panic("ytzero.NewPgAllowlistSync: pool is nil")
	}
	return &PgAllowlistSync{pool: pool}
}

var _ provider.AllowlistSynchronizer = (*PgAllowlistSync)(nil)

// AddMembership ensures the provider profile follows one YouTube channel.
func (s *PgAllowlistSync) AddMembership(ctx context.Context, providerProfileID string, channel provider.ScopedChannel) error {
	const op = "ytzero.PgAllowlistSync.AddMembership"
	if err := requireAllowlistCtx(ctx, op); err != nil {
		return err
	}
	uid, channelID, err := parseYouTubeMembership(op, providerProfileID, channel)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return apperr.Wrap(err, apperr.CodeUnavailable, op, "begin transaction")
	}
	defer tx.Rollback(ctx)
	if err := ensureProviderProfile(ctx, tx, op, uid); err != nil {
		return err
	}
	title := strings.TrimSpace(channel.Title)
	if title == "" {
		title = channelID
	}
	url := strings.TrimSpace(channel.URL)
	if url == "" {
		url = fmt.Sprintf("https://www.youtube.com/channel/%s", channelID)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO channels (channel_id, title, url, thumbnail)
VALUES ($1, $2, $3, '')
ON CONFLICT (channel_id) DO NOTHING`,
		channelID, title, url,
	); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "ensure channel row").With("channel_id", channelID)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO user_channels (user_id, channel_id, followed)
VALUES ($1, $2, 1)
ON CONFLICT (user_id, channel_id) DO UPDATE SET followed = 1`,
		uid, channelID,
	); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "upsert user channel").With("user_id", strconv.Itoa(uid))
	}
	if err := tx.Commit(ctx); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "commit transaction")
	}
	return nil
}

// RemoveMembership marks one YouTube channel unfollowed for the provider profile.
func (s *PgAllowlistSync) RemoveMembership(ctx context.Context, providerProfileID string, channel provider.ScopedChannel) error {
	const op = "ytzero.PgAllowlistSync.RemoveMembership"
	if err := requireAllowlistCtx(ctx, op); err != nil {
		return err
	}
	uid, channelID, err := parseYouTubeMembership(op, providerProfileID, channel)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
UPDATE user_channels SET followed = 0 WHERE user_id = $1 AND channel_id = $2`,
		uid, channelID,
	)
	if err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "unfollow channel").With("user_id", strconv.Itoa(uid))
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	return nil
}

// ReconcileAll replaces the provider profile follows with the host allowlist.
func (s *PgAllowlistSync) ReconcileAll(ctx context.Context, providerProfileID string, channels []provider.ScopedChannel) error {
	const op = "ytzero.PgAllowlistSync.ReconcileAll"
	if err := requireAllowlistCtx(ctx, op); err != nil {
		return err
	}
	uid, err := parseProviderProfileID(op, providerProfileID)
	if err != nil {
		return err
	}
	desired := make(map[string]provider.ScopedChannel, len(channels))
	for _, channel := range channels {
		_, channelID, err := parseYouTubeMembership(op, providerProfileID, channel)
		if err != nil {
			return err
		}
		desired[channelID] = channel
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return apperr.Wrap(err, apperr.CodeUnavailable, op, "begin transaction")
	}
	defer tx.Rollback(ctx)
	if err := ensureProviderProfile(ctx, tx, op, uid); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `
SELECT channel_id FROM user_channels WHERE user_id = $1 AND followed = 1`, uid)
	if err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "list followed channels")
	}
	defer rows.Close()
	current := map[string]struct{}{}
	for rows.Next() {
		var channelID string
		if err := rows.Scan(&channelID); err != nil {
			return apperr.Wrap(err, apperr.CodeFailed, op, "scan followed channel")
		}
		current[channelID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "read followed channels")
	}
	for channelID, channel := range desired {
		title := strings.TrimSpace(channel.Title)
		if title == "" {
			title = channelID
		}
		url := strings.TrimSpace(channel.URL)
		if url == "" {
			url = fmt.Sprintf("https://www.youtube.com/channel/%s", channelID)
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO channels (channel_id, title, url, thumbnail)
VALUES ($1, $2, $3, '')
ON CONFLICT (channel_id) DO NOTHING`,
			channelID, title, url,
		); err != nil {
			return apperr.Wrap(err, apperr.CodeFailed, op, "ensure channel row").With("channel_id", channelID)
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO user_channels (user_id, channel_id, followed)
VALUES ($1, $2, 1)
ON CONFLICT (user_id, channel_id) DO UPDATE SET followed = 1`,
			uid, channelID,
		); err != nil {
			return apperr.Wrap(err, apperr.CodeFailed, op, "follow channel").With("channel_id", channelID)
		}
	}
	for channelID := range current {
		if _, ok := desired[channelID]; ok {
			continue
		}
		if _, err := tx.Exec(ctx, `
UPDATE user_channels SET followed = 0 WHERE user_id = $1 AND channel_id = $2`,
			uid, channelID,
		); err != nil {
			return apperr.Wrap(err, apperr.CodeFailed, op, "unfollow channel").With("channel_id", channelID)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "commit transaction")
	}
	return nil
}

func parseYouTubeMembership(op, providerProfileID string, channel provider.ScopedChannel) (int, string, error) {
	if strings.ToLower(strings.TrimSpace(channel.Provider)) != provider.ProviderYouTube {
		return 0, "", apperr.New(apperr.CodeInvalid, op, "unsupported provider").With("provider", channel.Provider)
	}
	externalID := strings.TrimSpace(channel.ExternalID)
	if externalID == "" {
		return 0, "", apperr.New(apperr.CodeInvalid, op, "external_id is required")
	}
	uid, err := parseProviderProfileID(op, providerProfileID)
	if err != nil {
		return 0, "", err
	}
	return uid, externalID, nil
}

func parseProviderProfileID(op, providerProfileID string) (int, error) {
	uid, err := strconv.Atoi(strings.TrimSpace(providerProfileID))
	if err != nil || uid < 1 {
		return 0, apperr.New(apperr.CodeInvalid, op, "provider profile id must be numeric").
			With("provider_profile_id", providerProfileID)
	}
	return uid, nil
}

func ensureProviderProfile(ctx context.Context, tx pgx.Tx, op string, uid int) error {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, uid).Scan(&exists); err != nil {
		return apperr.Wrap(err, apperr.CodeFailed, op, "check provider profile")
	}
	if !exists {
		return apperr.New(apperr.CodeNotFound, op, "provider profile not found").With("user_id", strconv.Itoa(uid))
	}
	return nil
}

func requireAllowlistCtx(ctx context.Context, op string) error {
	if ctx == nil {
		return apperr.New(apperr.CodeInvalid, op, "context is required")
	}
	if _, ok := ctx.Deadline(); !ok {
		return apperr.New(apperr.CodeInvalid, op, "outbound: missing deadline")
	}
	return nil
}
