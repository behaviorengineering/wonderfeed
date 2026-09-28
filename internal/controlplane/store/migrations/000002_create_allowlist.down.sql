DROP TABLE IF EXISTS wonderfeed.wf_child_allowlist_channels;
ALTER TABLE wonderfeed.wf_child_profiles
    DROP COLUMN IF EXISTS allowlist_version;
