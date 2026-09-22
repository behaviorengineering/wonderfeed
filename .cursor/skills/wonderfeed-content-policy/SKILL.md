---
name: wonderfeed-content-policy
description: >-
  Applies Wonderfeed child-safety and parent-control policy assumptions:
  allowlist-first, fail-closed playback, approval boundaries, and honest limits
  of YouTube embeds. Use for child surface, session limits, approval flows,
  Shorts/live gating, or parental control claims.
---

# Wonderfeed content policy

## When to load

Load when designing or implementing child-facing playback, parent approval, allowlists, session limits, format gates (Shorts/live), or any claim that Wonderfeed "keeps kids safe" on YouTube-hosted video.

Related:

- [docs/product-brief.md](../../../docs/product-brief.md)
- [docs/architecture.md](../../../docs/architecture.md)
- `.cursor/skills/wonderfeed-product-context/`

## Core constraints

**CONSTRAINT:** Child playback MUST be allowlist-first and fail-closed.

- MUST: unknown channels/videos remain non-playable until policy or parent approval allows them.
- MUST NOT: default the child surface to open YouTube search, Home, Shorts shelf, or unrestricted related playback.

Enforcement: Trace the child play path; confirm a positive allow decision exists before play.
Violation: STOP, remove open discovery defaults, re-verify.

CORRECT:
```text
Child taps a card from the approved feed → play that videoId.
Child searches → creates a parent review request, not playback.
```

PROHIBITED:
```text
Child search box queries YouTube and plays the first result immediately.
```

**CONSTRAINT:** Agents MUST treat channel curation, parent approval, device/browser enforcement, and playback entitlement as separate layers.

- MUST: name which layer a change actually improves.
- MUST NOT: claim embed parameters or provider feed rules alone equal full parental control.

Enforcement: Architecture layer check in PR/design notes.
Violation: STOP, rewrite the claim per layer, re-verify.

CORRECT:
```text
Feed rules exclude Shorts; device profile blocks youtube.com outside the app.
```

PROHIBITED:
```text
rel=0 on the iframe means children cannot leave curated playback.
```

**CONSTRAINT:** Documentation and UI copy MUST state known YouTube embed and Premium limitations honestly.

- MUST: note that platform chrome, related content behavior, and fullscreen can create escape hatches.
- MUST: note that Premium ad-free playback depends on a signed-in browser session with cookies, and that `youtube-nocookie` embeds are usually the wrong choice when Premium recognition matters.
- MUST NOT: promise anonymous, cookieless, Premium, and fully chrome-free playback at once.

Enforcement: Scan new safety claims against `docs/architecture.md` limitations.
Violation: STOP, correct the docs/UI copy, re-verify.

**CONSTRAINT:** Parent controls MUST NOT be editable from the child surface.

- MUST: keep allowlist and limit administration on the parent/control-plane side.
- MUST NOT: expose settings that let a child widen the allowlist, disable limits, or unlock open YouTube browsing.

Enforcement: Permission/role review of settings routes.
Violation: STOP, move control to parent surface, re-verify.

## Steps

1. **Identify the actor** — parent, child, or operator.
2. **Require a positive allow** — channel/video/session rule must permit play.
3. **Gate formats** — Shorts, live, and upcoming streams follow explicit policy, not defaults.
4. **State enforcement gaps** — if device lockdown is required, say so instead of implying the app is enough.
5. **Keep Premium separate** — document session/cookie needs; do not invent proxy entitlements.

## Pre-completion checklist

- [ ] **Fail-closed play path:** No child play without allowlist/approval.
      Method: Walk UI/API from intent to player load.
      Pass: Missing allow → no play.
      Fail: Open path exists → close it.
- [ ] **Layer honesty:** Claims name curation vs approval vs device vs entitlement.
      Method: Review prose against architecture layers.
      Pass: No single-layer overclaim.
      Fail: Rewrite → re-verify.
- [ ] **Child cannot escalate:** Settings that widen policy are parent-only.
      Method: Role check on settings endpoints/screens.
      Pass: Child cannot mutate allowlist/limits.
      Fail: Restrict → re-verify.
