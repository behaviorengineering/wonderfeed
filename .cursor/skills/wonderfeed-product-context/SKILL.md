---
name: wonderfeed-product-context
description: >-
  Applies Wonderfeed product vocabulary, principles, scope, and non-goals when
  planning or implementing host features. Use for product brief alignment,
  parent-controlled feed work, child surface scope, roadmap checks, or when an
  agent might invent adult YouTube-client features.
---

# Wonderfeed product context

## When to load

Load when planning, implementing, or reviewing Wonderfeed host work that touches product scope, naming, parent/child roles, feed policy, or roadmap claims.

Canonical sources (do not fork a second truth):

- [docs/product-brief.md](../../../docs/product-brief.md)
- [docs/architecture.md](../../../docs/architecture.md)
- [docs/roadmap.md](../../../docs/roadmap.md)

## Core constraints

**CONSTRAINT:** Agents MUST treat Wonderfeed as a parent-owned control plane for kids' video watching, not as a general YouTube power-user client.

- MUST: preserve allowlist-first, fail-closed, and parent-owns-the-algorithm principles from the product brief.
- MUST NOT: expand scope into adult recommendation clients, unrestricted search as default, or "just use YouTube Home with filters" as the product thesis.

Enforcement: Compare the proposed change to `docs/product-brief.md` principles and non-goals.
Violation: STOP, rewrite the proposal against the brief, re-verify.

CORRECT:
```text
Add parent-configurable max autoplay count for the child session.
```

PROHIBITED:
```text
Add a For You tab that mirrors YouTube Home recommendations for kids.
```

**CONSTRAINT:** Vocabulary MUST stay consistent with the product brief table (host, provider, control plane, child surface, allowlist, feed policy).

- MUST: use those terms in docs, skills, and code identifiers when introducing new concepts.
- MUST NOT: invent parallel names for the same concept (for example "kid mode portal" vs "child surface" without an ADR).

Enforcement: Grep new docs/PRs for conflicting role names.
Violation: STOP, rename to brief vocabulary, re-verify.

**CONSTRAINT:** Agents MUST NOT claim a feature is shipped or enforced when only documentation or provider config exists.

- MUST: distinguish Milestone 0 scaffold, provider-as-is operation, and host-built control plane work using `docs/roadmap.md`.
- MUST NOT: imply device lockdown or Premium entitlement is solved by Wonderfeed docs alone.

Enforcement: Checklist against roadmap milestones before status language in README/PR.
Violation: STOP, downgrade claims to the matching milestone, re-verify.

## Steps

1. **Read the brief** — confirm problem, principles, and non-goals still apply to the request.
2. **Name the layer** — curation/feed, parent policy, child presentation, playback, or device enforcement.
3. **Check the roadmap** — place the work in a milestone or mark it as a new decision that needs an ADR.
4. **Refuse scope creep** — if the request matches a non-goal, say so and propose the nearest in-scope alternative.

## Pre-completion checklist

- [ ] **Brief alignment:** Change matches principles and avoids non-goals.
      Method: Diff intent against `docs/product-brief.md`.
      Pass: Explicit match; no new adult-client surface.
      Fail: Conflicts with brief → STOP, revise.
- [ ] **Vocabulary:** Uses host/provider/control plane/child surface/allowlist/feed policy consistently.
      Method: Scan new prose and identifiers.
      Pass: No synonym forks for the same role.
      Fail: Rename → re-verify.
- [ ] **Status honesty:** Does not overclaim enforcement or Milestone completion.
      Method: Compare claims to `docs/roadmap.md`.
      Pass: Claims match current milestone evidence.
      Fail: Soften claims → re-verify.
