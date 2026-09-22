# 0001. YT Zero as the first Wonderfeed provider

- **Status:** Accepted
- **Date:** 2026-09-22
- **Host path:** `providers/ytzero`
- **Upstream:** https://github.com/Pelski/ytzero
- **Pinned revision at acceptance:** `19b5b61fa2fe7967fa1270e66cfabe0fad18636e` (update via submodule bump; do not silently float `main` in production docs)

## Context

Wonderfeed needs a durable, self-hosted foundation for channel subscriptions, chronological inbox behavior, tags/rules, profiles, and intentional watching without depending on YouTube's recommendation home feed. Research compared extension-based organizers (PocketTube), privacy clients (FreeTube), approval-gate apps (BrainRotGuard), and commercial allowlist products. YT Zero matched the "manage your own algorithm" shape most closely: public RSS ingestion, local database ownership, triage states, and household/child profile features.

## Decision

Mount YT Zero as a **Git submodule** at `providers/ytzero` and treat it as the first **provider foundation**, not as the finished Wonderfeed product and not as a place to commit Wonderfeed brand or private policy.

## Consequences

### Positive

- Immediate access to subscription inbox, import formats, tags/rules, and household-oriented features.
- Clear repository boundary: provider stays upgradeable via pinned commits.
- Host docs and skills can describe Wonderfeed policy without forking upstream for messaging.

### Negative / follow-ups

- YT Zero alone is not full device enforcement; Wonderfeed must document OS/browser lockdown.
- AGPL-3.0 license requires care before distributing combined services or derivative works.
- Integration shape (wrap, compose, or replace UI) remains an open Milestone 2 decision.
- Host agents MUST NOT edit provider trees for product spill; use host adapters instead.

## Update policy

1. Pin a specific commit in the host gitlink (via submodule).
2. Review upstream release notes and license before bumping.
3. Record material capability changes in roadmap notes or a new ADR when behavior assumptions break.
4. After bump, run `git submodule update --init --recursive` in clean clones and re-verify child/parent flows before claiming compatibility.

## Out of scope for this decision

- Choosing Wonderfeed's runtime language or deployment topology.
- Choosing whether children use YT Zero UI directly.
- Implementing ranking beyond provider rules.
- Modifying upstream YT Zero source for Wonderfeed branding.
