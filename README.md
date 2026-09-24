# Wonderfeed

YouTube is built to keep people watching. For children, that means Shorts, open search, and rabbit holes. Wonderfeed turns YouTube into a library you control. It is what a smart home YouTube integration should be: voice-first, parent-aligned, and focused on your chosen channels rather than engagement.

### Why Wonderfeed?

- **Better than YouTube Kids:** You pick the channels. There is no mystery algorithm trying to keep them watching.
- **Better than timers:** You control the content, not just the clock.
- **Beyond YT Zero:** [YT Zero](https://github.com/Pelski/ytzero) is the technical engine that fetches trusted-channel feeds. Wonderfeed is the family product that owns the rules, the conversation, and fail-closed playback.

### How it works

1. **You curate the library.** Only channels you approve can appear.
2. **Distractions are gated.** Shorts, live, and open search are policy-controlled, not the default child surface.
3. **You own the session.** You decide what plays and for how long.

### Current status

This is early. Milestone 1 is a local foundation, not a polished parent app yet.

**Ready now**

- Host docs, ops, and a pinned YT Zero provider you can run locally (`make serve` / `make provider-up`)
- Allowlist-style channel feeds, with Shorts/live/search gated through provider policy
- No Wonderfeed parent app or child-facing product UI yet

**Building next**

- Parent voice/chat policy
- Child push-to-talk librarian
- Parent review for new requests
- Hosted channel evaluation

Details: [curated-library.md](docs/curated-library.md), [roadmap.md](docs/roadmap.md).

---

## Documentation

- [Product brief](docs/product-brief.md)
- [Architecture](docs/architecture.md)
- [Roadmap](docs/roadmap.md)
- [Curated library (vision)](docs/curated-library.md)
- [Operator runbook](docs/operator-ytzero.md)
- [Remote access](docs/cloudflare.md)
- [Copilot guide](AGENTS.md)
