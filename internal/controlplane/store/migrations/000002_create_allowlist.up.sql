-- Host-owned parent allowlist, separate from YT Zero's per-profile follows.
ALTER TABLE wonderfeed.wf_child_profiles
    ADD COLUMN IF NOT EXISTS allowlist_version BIGINT NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS wonderfeed.wf_child_allowlist_channels (
    child_profile_id UUID NOT NULL REFERENCES wonderfeed.wf_child_profiles(id) ON DELETE CASCADE,
    channel_id TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL,
    added_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (child_profile_id, channel_id)
);
