# ADR 0002: Wrap the provider now, replace the child surface later

**Status:** Accepted  
**Date:** 2026-09-25

## Context

Wonderfeed needs a working household product before it needs a custom child
surface. YT Zero already provides the feed, profile, playback, and child-policy
surfaces needed for the first release, while Wonderfeed owns parent policy and
the trusted channel allowlist.

The long-term child surface may be rebuilt in Wonderfeed. That future choice
must not delay the first usable vertical slice or move Wonderfeed policy into
the provider repository.

## Decision

Wonderfeed will **wrap YT Zero now**:

- Wonderfeed remains the parent control plane and source of truth for child
  policy and channel allowlists.
- YT Zero remains the initial child surface and provider feed implementation.
- The host adapter synchronizes the host-owned allowlist and supported policy
  fields to the selected YT Zero child profile.
- Future Wonderfeed child UI will consume the same host APIs and may replace the
  provider UI without changing the policy or persistence model.

The first shippable vertical slice is parent-only allowlist management,
provider synchronization, and child playback through the existing provider
surface. Activity views, approval requests, and a custom child surface remain
follow-on work.

## Rejected alternatives

### Compose now

This would make Wonderfeed own both parent and child UI before the policy and
allowlist seams have been exercised in working software. It adds presentation
scope without improving the first control-plane release.

### Replace the child surface now

This would duplicate YT Zero feed and playback behavior and delay the
allowlist-first product path. It remains a valid future direction after the
host API has real consumers.

## Licensing and boundary

YT Zero remains a separate AGPL-3.0 provider process and repository. Wonderfeed
uses its HTTP API through a host adapter and does not modify or brand the
provider tree. Any future distribution of a combined product still requires a
separate AGPL compliance review.

## Consequences

- The first release can ship with a working provider UI.
- Parent allowlist state is portable and independent of provider storage.
- The host syncs allowlist membership through an experimental PostgreSQL writer
  against YT Zero tables in the shared household database (no provider source
  changes). Policy continues through the HTTP adapter. Sync must fail closed when
  incomplete.
- A future child surface can replace YT Zero presentation behind the existing
  host control-plane API.
