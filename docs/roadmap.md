# Wonderfeed roadmap

## Milestone 0: Scaffold (this repository state)

- [x] Public host repository `behaviorengineering/wonderfeed`
- [x] Provider submodule `providers/ytzero`
- [x] Product brief, architecture, roadmap, and provider decision record
- [x] Host Cursor skills for product context, provider integration, and content policy
- [x] Merge scaffold PR and verify clean-clone with recursive submodules

**Exit gate:** a new contributor can clone, init submodules, and explain what Wonderfeed owns versus YT Zero.

## Milestone 1: Operate the provider as-is

- [x] Host compose overlay and Go CLI to run YT Zero without writing into the submodule
- [x] Host PostgreSQL + process-compose serve path (`make serve` / `make provider-up`)
- [x] Source-backed capability audit of pinned YT Zero (`providers/ytzero` @ `19b5b61`)
- [ ] Import a small trusted channel set for a trial household
- [ ] Exercise remaining runtime-only items (child lock UX, embed escape on a real device/browser)
- [x] Record gaps relative to Wonderfeed principles (approval requests, diversity rules, child UI lock strength)

### Capability matrix (source audit)

Classifications: **implemented** (wired end-to-end), **configurable** (depends on profile/settings), **not found**, **runtime-only** (needs device/UI verification).

| Requirement | Classification | Evidence | Owner |
| --- | --- | --- | --- |
| Trusted channel import (OPML, Takeout, NewPipe) | implemented | `app/src/routes.ts` `/channels/import`; `youtube.ts` `parseOpml`; `takeout.ts` | Configure YT Zero |
| Followed-only main feed | implemented | `feedQuery.ts` `feedVisibilityWhere` + `feedSourceExists` | Configure YT Zero |
| Child profiles + restricted defaults | configurable | `profilePermissions.ts` Restricted still allows `channels`; Child Lock PIN can unlock channel add | Configure YT Zero; Wonderfeed for hard no-widen |
| Shorts gating | configurable | `child_hide_shorts` is UI/nav only; feed uses `show_shorts` in `feedQuery.ts` | Configure both settings; Wonderfeed if one fail-closed policy |
| Live / upcoming gating | implemented | `childHidesLive` enforced in feed, adjacent playback, video routes | Configure YT Zero |
| Daily watch-time lock | implemented (cooperative) | `childTime.ts` `childStatus.locked`; UI `ChildLockScreen.tsx` | Configure YT Zero; runtime UX check |
| Parent time grants | implemented | `childRoutes.ts` time-request + `applyGrant` | Configure YT Zero |
| Autoplay controls | configurable | `feed_autoplay_*` defaults off in `db.ts` | Configure YT Zero |
| Open YouTube search blocked | configurable | `GET /search/youtube` empty when `child_local_only`; local library search remains | Keep Subscribed content only |
| Main-feed diversity (max 1 channel / N) | not found | Discovery `per_channel_limit` only (`recommendationRanking.ts`); not main feed | Wonderfeed host |
| Approve new channel/video requests | not found | Only `child_time_requests` exist | Wonderfeed host |
| Downloads-only / local-only modes | implemented | `child_local_only`, `child_downloads_only` | Configure YT Zero (+ yt-dlp for downloads-only) |
| Inspectable / exportable state | implemented (backup); not found (OPML export) | `backupRoutes.ts` / `portableBackup.ts`; OPML import only | Configure backup; optional Wonderfeed OPML-out |
| Embed / youtube.com escape | runtime-only | Wiki `Child-Lock.md`: cross-origin player may still expose YouTube links | Device/browser layer |

### Gap list (ownership)

| Gap | Configure in YT Zero? | Must build on Wonderfeed host? | Notes |
| --- | --- | --- | --- |
| Trusted set + fresh followed feed | Yes | No | Import + follow on the child profile |
| Child time limits + time grants | Yes | No | Cooperative UI lock; server reports `locked` |
| Shorts / live gating | Mostly yes | Maybe | Live is server-enforced; set `show_shorts` with `child_hide_shorts` |
| Subscribed-only / no open YT search | Yes (`child_local_only`) | No for basic gate | Local library search still available |
| Child cannot widen allowlist | Partial (deny `channels`, Child Lock) | Yes for hard guarantee | PIN-unlocked child can still follow |
| Diversity (1 channel / N on main feed) | No | Yes | Discovery plugin is not the main child feed |
| Approve new channel/video requests | No | Yes | Time requests only |
| OPML export | No | Optional | Portable backup includes subscriptions |
| Embed / youtube.com escape | Downloads-only helps | Presentation + docs | Device/browser lockdown required for honesty |

**Exit gate:** written gap list that separates "configure YT Zero" from "must build in Wonderfeed." (met by the tables above; household import + runtime UX still open.)

### Runtime verification (focused)

Verified against the live host stack (`make provider-health`, pin `19b5b61`, `"database": "postgres"`):

| Check | Result |
| --- | --- |
| Provider health / pin | OK; version `2026.09.8`, commit `19b5b61` |
| Unauthenticated `GET /api/child/status` | `is_child: false` (default session is not a child profile) |
| Unauthenticated `GET /api/search/youtube` | Returns live YouTube results (open search for the default/non-child session) |
| Child `local_only` search gate | Confirmed in source (`libraryRoutes.ts`); needs a logged-in child session to exercise on the live UI |
| Embed escape | Confirmed in upstream wiki `Child-Lock.md` and `WatchPage.tsx` (`showYouTube={!isChildProfile}`); hard block is device/browser, not app-only |
| Provider unit tests | Not run here: host has no `bun` on PATH; re-run in the YT Zero tree with `bun test` when available |

Still deferred to a household trial: import a small trusted set, create a child profile, and walk lock UX / Shorts tab / live block in the browser.

Operator runbook: [operator-ytzero.md](operator-ytzero.md).

## Milestone 2: Choose the integration shape

**Accepted path:** [ADR 0002](decisions/0002-wrap-provider-now.md), wrap the
provider now and replace the child surface later.

| Option | Meaning |
| --- | --- |
| A. Wrap | **Selected.** Wonderfeed control plane configures and fronts YT Zero; child may still use provider UI under lockdown |
| B. Compose | Wonderfeed owns child/parent UI; reads/writes provider feed state through a thin adapter |
| C. Replace surface | Keep YT Zero for ingestion only; Wonderfeed replaces presentation entirely |

**Exit gate:** met by ADR 0002, including rejected alternatives and AGPL-3.0 licensing implications.

## Provider scope (product, not schedule)

Wonderfeed is **YouTube-first in implementation** because YT Zero is the first mounted provider. The control plane is **not YouTube-exclusive**: allowlist entries are provider-scoped (`provider` + `external_id`), and the same parent-owned curation model applies to any future algorithmic catalog (for example Netflix-style feeds). A well-known brand does not replace allowlist-first policy or fail-closed playback.

## Milestone 3: Parent control plane MVP

Minimum host-owned capabilities (may be adapters over provider features):

- [x] Host-owned child profile + policy API (`wonderfeed control serve`) with provider adapter seam
- [x] Durable desired policy in host PostgreSQL (`wonderfeed.*`) with sync status
- [x] Fail-closed bind: non-loopback requires `WONDERFEED_PARENT_AUTH_KEY`
- [x] Provider-scoped channel allowlist CRUD with a versioned, exportable JSON source of truth
- [ ] Parent activity view (what played, what was rejected, pending requests if any)
- [ ] Child cannot widen allowlists from the child surface (hard guarantee beyond YT Zero PIN)

**Exit gate:** a parent can change policy without editing provider source, and children cannot widen allowlists from the child surface.

## Milestone 4: Hardening for real devices

- Document recommended device/browser lockdown for macOS, iOS, Android, and living-room browsers.
- Reduce escape hatches (outbound YouTube browsing, stock apps, unrestricted search).
- Clarify Premium family seating and cookie requirements when using embeds.

**Exit gate:** a trial family runs for one week without relying on "please don't open YouTube."

## Open risks

| Risk | Why it matters | Mitigation direction |
| --- | --- | --- |
| Embed escape | Child can leave curated flow via YouTube chrome | Device lockdown + UI choices that avoid outbound links |
| AGPL coupling | Host services that link/modify YT Zero may inherit obligations | Prefer process boundary / clear licensing review before distribution |
| Upstream churn | Provider APIs and UI change | Pin submodule commits; adapt on host; avoid deep forks without ADR |
| False safety | Docs imply YT Zero alone is full parental control | Keep architecture language honest about layers |
| Scope creep | Building adult YouTube power-user features | Hold to product brief non-goals |

## Later direction (not scheduled)

Hosted curated library: central discovery and evaluation service that grows a shared catalog of channels and videos worth following, then drives parent/kid search suggestions from that store instead of YouTube Home. Draft: [curated-library.md](curated-library.md).

## Decision log

| ID | Title | Status |
| --- | --- | --- |
| 0001 | YT Zero as first provider | Accepted |
| 0002 | Wrap provider now, replace child surface later | Accepted |

Future ADRs belong in `docs/decisions/` with sequential numbers.
