# Wonderfeed architecture

## Current shape

Wonderfeed is a **host repository** with provider submodules. A small host Go CLI and Docker Compose overlay can run the pinned YT Zero provider locally. There is not yet a Wonderfeed control-plane database or child-facing product UI. The architecture docs define ownership and seams so later implementation does not blur them.

```text
Parent policy and product docs
  -> Wonderfeed host CLI / deploy overlay (ops)
  -> Wonderfeed host-owned adapter and future control plane
    -> YT Zero provider boundary at providers/ytzero
    -> provider-owned feed state and playback links
    -> future child-facing presentation
```

## Ownership boundary

| Concern | Owner today | Notes |
| --- | --- | --- |
| Product principles, roadmap, parent UX intent | Wonderfeed host | Docs and host skills under this repo |
| Local provider run (compose, process-compose, env, health) | Wonderfeed host | `deploy/ytzero/`, `data/postgres/`, `data/ytzero/`, `cmd/wonderfeed` |
| Subscription inbox, tags, rules, profiles | YT Zero provider | Source at `providers/ytzero`; DB in host Postgres; files under `data/ytzero` |
| Remote HTTPS to homes (school/product) | Wonderfeed + Cloudflare | Design in [cloudflare.md](cloudflare.md); not local serve |
| Device lockdown / kiosk / DNS blocks | Outside app (OS, browser profile, network) | Required for real child enforcement |
| YouTube embed chrome and related videos | YouTube platform | Not fully removable via embed params |
| Premium / ad-free entitlement | Signed-in YouTube session in the playback browser | Separate from Wonderfeed curation |

**Rule:** do not write Wonderfeed brand, private paths, or family policy into `providers/ytzero`. Prefer host adapters and docs. See `.cursor/rules/repository-boundaries.mdc`.

## Layers (conceptual)

### 1. Curation and feed state

Trusted channels, tags, automatic rules, archive/reject flows, and chronological inbox behavior. YT Zero already implements much of this as a self-hosted subscription inbox without Google login or YouTube Data API keys.

### 2. Parent approval and policy

Who may watch what, daily limits, Shorts/live gating, request-to-approve flows, and diversity rules (for example max one video per channel in N slots). Some of this exists in YT Zero child profiles; Wonderfeed may wrap or extend it later without forking product spill into upstream.

### 3. Child presentation

The surface children actually use. Options deliberately deferred:

- Use YT Zero UI directly in a locked browser profile.
- Wrap YT Zero behind a thinner Wonderfeed child UI.
- Consume provider APIs/data and render a separate host UI.

### 4. Playback

Video bytes and player chrome remain with YouTube (embed) or optional local download plugins inside the provider. Wonderfeed must treat playback entitlement (cookies, Premium family seats) as a browser-session concern, not as something a custom proxy can honestly invent.

### 5. Device and network enforcement

The real boundary against "open youtube.com and escape the feed." Managed child profiles, kiosk mode, app allowlists, and DNS filtering sit here. Application design can reduce escape hatches; it cannot replace OS-level controls.

## Integration seams (likely)

These are the places a future host adapter will touch first:

1. **Channel set import/export** — OPML, Takeout CSV, NewPipe JSON already understood by YT Zero.
2. **Feed listing** — chronological uploads from allowlisted channels, with format filters.
3. **Triage states** — watch / schedule / archive / reject as durable host or provider state.
4. **Profile and limit APIs** — map Wonderfeed parent policy onto provider profiles when wrapping.
5. **Playback handoff** — approved `videoId` into an embed or provider player, never unrestricted search.

No registry, DI container, or host control-plane database exists yet. Host ops for the provider are documented in [operator-ytzero.md](operator-ytzero.md). When control-plane registration arrives, document order in a plan that follows `.cursor/skills/plan-scaffold/`.

## Explicit limitations

- Embedded YouTube players can still expose platform UI (logo, related content behavior, fullscreen). Policy must assume partial escape unless the device blocks it.
- Privacy-enhanced (`youtube-nocookie`) embeds conflict with reliable Premium session recognition. Prefer ordinary `youtube.com` embeds when Premium is required.
- Third-party frontends that avoid Google login generally cannot inherit YouTube Premium entitlements.
- YT Zero is a strong foundation for "manage your own algorithm," not a complete substitute for OS lockdown.

## Deferred decisions

Documented in [roadmap.md](roadmap.md): wrap vs extend vs replace YT Zero UI, runtime language, auth model, telemetry policy, and ranking sophistication beyond rules.
