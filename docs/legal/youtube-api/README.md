# YouTube API terms (tracking)

**Purpose:** Keep Wonderfeed aligned with the official YouTube API Services documents when designing API-backed features (for example the curated library vision in [../../curated-library.md](../../curated-library.md)).

**Copyright:** Google owns these texts. This repository does **not** commit full verbatim copies. Local page snapshots live under `snapshots/*.md` (gitignored) for offline review. Git tracks only this index and [manifest.json](manifest.json) (canonical URLs, fetch timestamps, content hashes).

## Canonical documents

| Id | Document | URL |
| --- | --- | --- |
| `api-services-terms-of-service` | YouTube API Services Terms of Service | https://developers.google.com/youtube/terms/api-services-terms-of-service |
| `developer-policies` | YouTube API Services Developer Policies | https://developers.google.com/youtube/terms/developer-policies |
| `developer-policies-guide` | Developer Policies Guide | https://developers.google.com/youtube/terms/developer-policies-guide |
| `required-minimum-functionality` | Required Minimum Functionality | https://developers.google.com/youtube/terms/required-minimum-functionality |
| `revision-history` | Terms revision history (watch for changes) | https://developers.google.com/youtube/terms/revision-history |
| `determine-quota-cost` | Quota costs | https://developers.google.com/youtube/v3/determine_quota_cost |

Revision history RSS (optional subscribe in a reader):  
https://developers.google.com/youtube/terms/revision-history/rss.xml

## Refresh local snapshots

From the repo root (needs network):

```bash
make youtube-api-terms-refresh
```

That runs [../../../scripts/fetch-youtube-api-terms.sh](../../../scripts/fetch-youtube-api-terms.sh), which:

1. Downloads each URL into `docs/legal/youtube-api/snapshots/<id>.md`
2. Writes SHA-256 hashes and `fetched_at` into `manifest.json`
3. Prints whether any hash changed since the last committed manifest

**Keep up to date:** re-run the target periodically (or when designing API features). If hashes change, skim the revision history, update product docs if needed, then commit the updated `manifest.json` (still not the page snapshots).

## Operator notes (not legal advice)

- Most public API metadata you store should be deleted or refreshed within about 30 days (see Developer Policies storage sections).
- Do not scrape youtube.com; use the official API only for API-backed services.
- Do not download or cache audiovisual content without YouTube’s prior written approval.
- Label Wonderfeed judgments as your own; do not present them as YouTube metrics.
- The Developer Policies Guide says not to use the API to make claims that a video or channel is “safe or suitable to watch”; keep Wonderfeed verdicts clearly product-owned and counsel-reviewed.
- Child-directed clients have extra COPPA/GDPR and Made for Kids duties.

Have counsel review before shipping a multi-tenant Data API product.
