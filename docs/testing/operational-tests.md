# Operational test catalog

Manual-first checks for a real Wonderfeed household stack. Use this for Milestone 1 import/runtime exit, Milestone 3 control-plane smoke, and Milestone 4 trial prep. Stable IDs tie each case to future automation in full regression.

**Index:** [testing/README.md](README.md). **Setup:** [operator-ytzero.md](../operator-ytzero.md). **Scope:** [roadmap.md](../roadmap.md).

## Conventions

| Field | Meaning |
| --- | --- |
| **Tier** | `smoke` (minutes), `regression` (full pass before pin bump), `trial` (household week) |
| **Automation** | `yes` (script-friendly), `partial` (API only; session/UI still manual), `none` (device/browser judgment) |
| **Owner** | `host` (Wonderfeed CLI/API), `provider` (YT Zero UI/API), `device` (OS/browser/network) |

Record results in your run log: date, git SHA (`providers/ytzero` pin + host `HEAD`), `make provider-health` version line, pass/fail, notes.

### Baseline environment

1. `git submodule update --init --recursive`
2. `make build`
3. `make init` (or existing `~/.config/wonderfeed/config.yaml`)
4. `make provider-up` (or `make serve`)
5. `make provider-health` → `"database": "postgres"`
6. Provider UI: http://127.0.0.1:3001
7. Control plane (when testing host API): `bin/wonderfeed control migrate up`, then `bin/wonderfeed control serve` on `127.0.0.1:8080` with `database_url` pointing at the same Postgres as YT Zero (`postgresql://...@127.0.0.1:5432/ytzero`)

When YT Zero auth is enabled, set `YTZERO_SESSION_COOKIE` (primary profile) before policy sync and access-control tests. Sync fails closed without it when required ([operator-ytzero.md](../operator-ytzero.md)).

---

## A. Stack smoke (`OP-*`)

| ID | Tier | Auto | Owner | Procedure | Expected |
| --- | --- | --- | --- | --- | --- |
| OP-001 | smoke | yes | host | `make provider-status` | Submodule pin and compose state print; paths under host `data/` |
| OP-002 | smoke | yes | host | `make provider-health` | HTTP 200; JSON includes `"database": "postgres"` |
| OP-003 | smoke | partial | host | Compare health version/commit to `providers/ytzero` pin in roadmap / `provider status` | Running image matches accepted pin |
| OP-004 | regression | yes | host | `make test` && `make vet` | All pass (host unit; no live stack) |
| OP-005 | regression | partial | provider | In `providers/ytzero`, run `bun test` when Bun is installed | Upstream unit suite passes at pinned commit |

**Automation notes:** OP-001–003 map to a future `scripts/regression/smoke.sh` or CI job after `provider-up`. OP-004 already runs in CI via `make test`.

---

## B. Provider API probes (`OP-*`, no child session)

Default browser session is **not** a child profile unless noted.

| ID | Tier | Auto | Owner | Procedure | Expected |
| --- | --- | --- | --- | --- | --- |
| OP-010 | smoke | yes | provider | `GET /api/health` (via health CLI or curl) | OK + postgres backend |
| OP-011 | smoke | yes | provider | `GET /api/child/status` unauthenticated | `is_child: false` for default session |
| OP-012 | regression | yes | provider | `GET /api/search/youtube?q=test` as default/adult session | Live YouTube search results (proves open search exists for non-child) |
| OP-013 | regression | partial | provider | With child profile logged in and `child_local_only` enabled: library/local search vs YouTube search routes | YouTube open search empty or blocked; local library search still works per policy |

**Automation notes:** OP-010–012 are curl-friendly. OP-013 needs session cookie or UI login fixture.

---

## C. Control plane API (`CP-*`)

Requires `control serve` and migrations. Loopback bind does not require parent auth key; document any non-loopback trial separately.

| ID | Tier | Auto | Owner | Procedure | Expected |
| --- | --- | --- | --- | --- | --- |
| CP-001 | smoke | yes | host | `POST /api/v1/parent/children` with `{"name":"trial-child"}` | 201 + child id |
| CP-002 | regression | yes | host | `GET /api/v1/parent/children` | Lists created child |
| CP-003 | regression | yes | host | `PUT .../policy` with supported fields (`daily_minutes`, `local_only`, `hide_shorts`, `hide_live`, `downloads_only`) | 200; sync status eventually OK |
| CP-004 | regression | yes | host | `POST .../allowlist` with `provider: youtube`, valid `external_id` | Entry stored; version/export fields present |
| CP-005 | regression | partial | host | `POST .../sync` (force sync) | Reconcile runs; provider follows match allowlist when DSN shared |
| CP-006 | regression | partial | host | With auth enabled: sync without `YTZERO_SESSION_COOKIE` | Fail closed (error surfaced, policy not silently skipped) |
| CP-007 | regression | partial | provider | After sync on Wonderfeed-managed child: attempt widen follows (add channel / import) in YT Zero UI as child | Denied via access-control (`channels`, `imports`, `followed_playlists`) |

**Automation notes:** CP-001–004 overlap `internal/httpapi` tests; extend with integration tests against test Postgres. CP-005–007 need live stack + optional cookie.

---

## D. Trusted set and feed (`UI-*`)

Milestone 1 household import. Use a **small** OPML or Takeout export (3–5 known kid-safe channels).

| ID | Tier | Auto | Owner | Procedure | Expected |
| --- | --- | --- | --- | --- | --- |
| UI-001 | regression | none | provider | Import trusted channels (OPML/Takeout/NewPipe) on parent or primary profile | Channels appear in library; subscriptions durable across `provider-down` / `provider-up` |
| UI-002 | regression | none | provider | Create restricted child profile; assign/import follows appropriate to trial | Child sees only subscribed/main feed behavior per YT Zero settings |
| UI-003 | regression | none | provider | Confirm main feed shows new uploads from followed channels only | No open Home-style recommendations from unfollowed sources |
| UI-004 | regression | none | provider | Shorts: `child_hide_shorts` + `show_shorts` aligned with household policy | Shorts tab hidden or feed excludes Shorts per configuration |
| UI-005 | regression | none | provider | Live/upcoming: child profile with live hidden | Live items absent from feed and adjacent playback paths |
| UI-006 | regression | none | provider | Daily watch limit + time grant flow | Lock screen appears when limit hit; parent grant extends watch time |
| UI-007 | trial | none | provider | Autoplay settings (`feed_autoplay_*`) | Behavior matches configured defaults (typically off for trial) |

**Automation notes:** UI-* candidates for Playwright against local stack; keep fixtures in host repo, not in `providers/ytzero`.

---

## E. Backup and restore (`OP-*`)

Destructive. Use a disposable clone or accept safety snapshot behavior.

| ID | Tier | Auto | Owner | Procedure | Expected |
| --- | --- | --- | --- | --- | --- |
| OP-020 | regression | partial | host | `wonderfeed backup create --local-only` then `backup list` | Encrypted artifact under `data/backups/` |
| OP-021 | trial | none | host | `backup restore <id> --confirm` on test stack | Safety snapshot written; subscriptions/policy restored per runbook |

---

## F. Device and escape honesty (`DEV-*`)

Milestone 4. Application reduces hatches; **device** layer proves fail-closed claims ([architecture.md](../architecture.md) layer 5).

| ID | Tier | Auto | Owner | Procedure | Expected |
| --- | --- | --- | --- | --- | --- |
| DEV-001 | regression | none | device | Child profile: open watch page, inspect player chrome for outbound YouTube links | Document what is still reachable (wiki `Child-Lock.md` expectations) |
| DEV-002 | regression | none | device | Attempt `youtube.com` in same browser profile outside kiosk | Blocked or unavailable when lockdown applied |
| DEV-003 | trial | none | device | One-week daily use on target appliance (see [living-room-hardware.md](../living-room-hardware.md)) | Family does not rely on honor-system "do not open YouTube" |
| DEV-004 | trial | none | device | Premium/embed trial if applicable | Document cookie/session requirements; no false ad-free claims |

Platform-specific lockdown steps remain a separate doc task on the roadmap; link results from trial runs into that doc when it exists.

---

## G. Known gaps (do not treat as pass)

These roadmap items are **not** covered by passing the tables above. Failing them is expected until built.

| Gap | Roadmap owner | Test stance |
| --- | --- | --- |
| Main-feed diversity (max 1 channel / N) | Wonderfeed host | No OP/UI case yet; add `CP-*` / `UI-*` when implemented |
| Approve new channel/video requests (non-time) | Wonderfeed host | No case yet |
| Parent activity view (raw chronology) | Wonderfeed host | Deferred; superseded by [content-portfolio.md](../content-portfolio.md) vision |
| OPML export | Optional host | Import only today (UI-001) |

---

## Suggested run order

1. **Daily dev:** OP-001, OP-002, OP-004  
2. **Before submodule pin bump:** Full A + B + C + `make test` + OP-005 if Bun available  
3. **Household trial prep:** A–E, then DEV-001–002 on the child device profile  
4. **Milestone 4 exit:** DEV-003 for seven days, record incidents

---

## Full regression (future)

| Phase | Scope |
| --- | --- |
| CI (every PR) | OP-004; optional OP-002 when job can start compose |
| Nightly / pre-release | OP-001–003, OP-010–012, CP-001–005 against ephemeral Postgres |
| Manual gate | All `UI-*` and `DEV-*` until browser/device harness ships |

When adding automation, reference the same ID in test name or comment (example: `TestOperational_OP002_ProviderHealthPostgres`).
