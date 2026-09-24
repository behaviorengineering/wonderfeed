-- Host-owned Wonderfeed control-plane tables (separate from YT Zero app schema).
CREATE SCHEMA IF NOT EXISTS wonderfeed;

CREATE TABLE IF NOT EXISTS wonderfeed.schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wonderfeed.wf_child_profiles (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    avatar_color TEXT NOT NULL DEFAULT '#7c5cff',
    provider_profile_id TEXT NOT NULL DEFAULT '',
    version BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS wonderfeed.wf_child_policies (
    child_profile_id UUID PRIMARY KEY REFERENCES wonderfeed.wf_child_profiles(id) ON DELETE CASCADE,
    daily_minutes INTEGER NOT NULL DEFAULT 0 CHECK (daily_minutes >= 0 AND daily_minutes <= 1440),
    local_only BOOLEAN NOT NULL DEFAULT TRUE,
    hide_shorts BOOLEAN NOT NULL DEFAULT TRUE,
    hide_live BOOLEAN NOT NULL DEFAULT TRUE,
    downloads_only BOOLEAN NOT NULL DEFAULT FALSE,
    bedtime_start TEXT NOT NULL DEFAULT '',
    bedtime_end TEXT NOT NULL DEFAULT '',
    version BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS wonderfeed.wf_policy_sync_events (
    id BIGSERIAL PRIMARY KEY,
    child_profile_id UUID NOT NULL REFERENCES wonderfeed.wf_child_profiles(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('synced', 'sync_pending', 'sync_failed')),
    sync_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS wf_policy_sync_events_child_created_idx
    ON wonderfeed.wf_policy_sync_events (child_profile_id, created_at DESC);
