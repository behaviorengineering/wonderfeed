-- Provider-scoped allowlist identity (YouTube is the first provider).
ALTER TABLE wonderfeed.wf_child_allowlist_channels
    ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'youtube';

ALTER TABLE wonderfeed.wf_child_allowlist_channels
    ADD COLUMN IF NOT EXISTS external_id TEXT;

UPDATE wonderfeed.wf_child_allowlist_channels
SET external_id = channel_id
WHERE external_id IS NULL OR external_id = '';

ALTER TABLE wonderfeed.wf_child_allowlist_channels
    ALTER COLUMN external_id SET NOT NULL;

ALTER TABLE wonderfeed.wf_child_allowlist_channels
    DROP CONSTRAINT IF EXISTS wf_child_allowlist_channels_pkey;

ALTER TABLE wonderfeed.wf_child_allowlist_channels
    DROP COLUMN IF EXISTS channel_id;

ALTER TABLE wonderfeed.wf_child_allowlist_channels
    ADD PRIMARY KEY (child_profile_id, provider, external_id);
