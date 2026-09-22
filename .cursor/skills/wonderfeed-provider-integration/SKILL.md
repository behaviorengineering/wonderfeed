---
name: wonderfeed-provider-integration
description: >-
  Enforces Wonderfeed provider adapter rules for YT Zero and future providers:
  submodule boundaries, no product spill, pin updates, and portable upstream
  trees. Use when editing providers/, adding submodules, wrapping YT Zero, or
  syncing provider pins.
---

# Wonderfeed provider integration

## When to load

Load when touching `providers/`, `.gitmodules`, provider pins, host adapters around YT Zero, or any change that might edit upstream provider source for Wonderfeed product needs.

Related:

- [docs/architecture.md](../../../docs/architecture.md)
- [docs/decisions/0001-ytzero-provider.md](../../../docs/decisions/0001-ytzero-provider.md)
- `.cursor/rules/repository-boundaries.mdc`
- `.cursor/skills/sync-submodules-after-merge/`

## Core constraints

**CONSTRAINT:** Before editing any path under `providers/`, agents MUST resolve the owning Git repository and treat it as a nested repository, not host files.

- MUST: run `git -C <path> rev-parse --show-toplevel` (or equivalent) before the first write in that tree.
- MUST NOT: stage provider file bytes as ordinary host-owned content from the host root.

Enforcement: Ownership check before write; `git status` in both host and provider before claiming done.
Violation: STOP, resolve ownership, continue only under the correct boundary.

CORRECT:
```text
Need a Wonderfeed parent policy field
→ implement in host adapter/docs
→ leave providers/ytzero unchanged
```

PROHIBITED:
```text
Need a Wonderfeed parent policy field
→ edit providers/ytzero UI copy to say "Wonderfeed"
```

**CONSTRAINT:** Wonderfeed brand, private layout paths, host-only skills, and family policy MUST stay in the host. Provider trees MUST remain portable for any consumer of that upstream project.

- MUST: put product-specific behavior in host adapters, host docs, or host skills.
- MUST NOT: commit Wonderfeed naming, household fixtures, or private paths into `providers/ytzero`.

Enforcement: Review the planned diff for host brand and private paths inside provider trees.
Violation: STOP, move spill to the host, re-verify.

**CONSTRAINT:** YT Zero updates MUST be explicit submodule pin bumps with review, not silent floats to floating branch tips in production assumptions.

- MUST: record material pin changes and re-check roadmap assumptions when upstream behavior breaks docs.
- MUST NOT: document "always track upstream main" as the production pin policy without an ADR.

Enforcement: `git submodule status` and decision/roadmap notes on bump PRs.
Violation: STOP, pin a commit, document the bump, re-verify.

**CONSTRAINT:** Prefer wrap/compose over forking provider internals for product features.

- MUST: propose a host adapter or process boundary when Wonderfeed needs behavior YT Zero does not expose cleanly.
- MUST NOT: "just this once" patch upstream for host-only UX unless an ADR chooses a maintained fork.

Enforcement: Design review against Milestone 2 options in `docs/roadmap.md`.
Violation: STOP, write or update an ADR, re-verify.

## Steps

1. **Resolve ownership** — confirm whether the path is host or provider.
2. **Refuse spill** — if the request is product-specific, keep changes on the host.
3. **Pin consciously** — when updating YT Zero, bump the gitlink and note license/behavior impact.
4. **Sync cleanly** — after merge or pin bump, follow recursive submodule update practice so clones are not falsely dirty.

## Pre-completion checklist

- [ ] **Ownership:** Provider edits (if any) are intentional upstream work, not host spill.
      Method: `git -C providers/ytzero status` and diff review.
      Pass: No Wonderfeed brand/policy in provider tree.
      Fail: Revert spill → host-only change.
- [ ] **Gitlink:** `.gitmodules` and submodule SHA match the intended pin.
      Method: `git submodule status` and `cat .gitmodules`.
      Pass: Paths and URLs correct; SHA intentional.
      Fail: Fix metadata → re-verify.
- [ ] **Clone path:** Fresh recursive init still works after the change.
      Method: Documented clone command or disposable clone test.
      Pass: Provider tree populates without host files inside it.
      Fail: Fix submodule metadata → re-test.
