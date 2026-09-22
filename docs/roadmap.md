# Wonderfeed roadmap

## Milestone 0: Scaffold (this repository state)

- [x] Public host repository `behaviorengineering/wonderfeed`
- [x] Provider submodule `providers/ytzero`
- [x] Product brief, architecture, roadmap, and provider decision record
- [x] Host Cursor skills for product context, provider integration, and content policy
- [ ] Merge scaffold PR and verify clean-clone with recursive submodules

**Exit gate:** a new contributor can clone, init submodules, and explain what Wonderfeed owns versus YT Zero.

## Milestone 1: Operate the provider as-is

- Run YT Zero locally (Docker Compose or documented upstream path).
- Import a small trusted channel set for a trial household.
- Exercise child profile, subscribed-content-only mode, Shorts/live gating, and watch-time limits.
- Record gaps relative to Wonderfeed principles (approval requests, diversity rules, child UI lock strength).

**Exit gate:** written gap list that separates "configure YT Zero" from "must build in Wonderfeed."

## Milestone 2: Choose the integration shape

Pick one primary path (decision record required):

| Option | Meaning |
| --- | --- |
| A. Wrap | Wonderfeed control plane configures and fronts YT Zero; child may still use provider UI under lockdown |
| B. Compose | Wonderfeed owns child/parent UI; reads/writes provider feed state through a thin adapter |
| C. Replace surface | Keep YT Zero for ingestion only; Wonderfeed replaces presentation entirely |

**Exit gate:** ADR chooses A/B/C with rejected alternatives and license implications (YT Zero is AGPL-3.0).

## Milestone 3: Parent control plane MVP

Minimum host-owned capabilities (may be adapters over provider features):

- Channel allowlist management with exportable source of truth.
- Child session policy (duration, max autoplay count, format filters).
- Parent activity view (what played, what was rejected, pending requests if any).
- Fail-closed defaults for unknown content.

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

## Decision log

| ID | Title | Status |
| --- | --- | --- |
| 0001 | YT Zero as first provider | Accepted |

Future ADRs belong in `docs/decisions/` with sequential numbers.
