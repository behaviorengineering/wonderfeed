ALTER TABLE wonderfeed.wf_child_allowlist_channels
    DROP CONSTRAINT IF EXISTS wf_child_allowlist_channels_pkey;

ALTER TABLE wonderfeed.wf_child_allowlist_channels
    ADD COLUMN IF NOT EXISTS channel_id TEXT;

UPDATE wonderfeed.wf_child_allowlist_channels
SET channel_id = external_id
WHERE channel_id IS NULL OR channel_id = '';

ALTER TABLE wonderfeed.wf_child_allowlist_channels
    ALTER COLUMN channel_id SET NOT NULL;

ALTER TABLE wonderfeed.wf_child_allowlist_channels
    DROP COLUMN IF EXISTS provider,
    DROP COLUMN IF EXISTS external_id;

ALTER TABLE wonderfeed.wf_child_allowlist_channels
    ADD PRIMARY KEY (child_profile_id, channel_id);
