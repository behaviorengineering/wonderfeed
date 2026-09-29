# Testing (Wonderfeed host)

Wonderfeed verification spans three layers. Keep them separate so CI stays fast and household checks stay honest about UI and device limits.

| Layer | What it proves | Where it lives | Typical trigger |
| --- | --- | --- | --- |
| Unit / package | Host Go logic, HTTP handlers with fakes, adapter contracts | `make test`, `make vet` | Every PR |
| Provider unit | YT Zero app behavior in the pinned submodule | `providers/ytzero` with `bun test` (when Bun is on PATH) | Provider pin bumps, upstream fixes |
| Operational | Live stack, real Postgres, browser sessions, device lockdown | [operational-tests.md](operational-tests.md) | Release prep, household trial, Milestone 1/4 exit |

## Operational catalog

**[operational-tests.md](operational-tests.md)** is the checklist for Milestone 1 runtime verification, control-plane household smoke, and Milestone 4 device hardening. Cases use stable IDs (`OP-*`, `CP-*`, `UI-*`, `DEV-*`) so automation can adopt them into a future **full regression** suite without renaming prose.

Manual execution is the default today. Each case notes an **automation target** (`none`, `partial`, `yes`) for when we wire scripted probes (curl, Playwright, or Go integration tests against a test stack).

## Related docs

- Stack setup: [operator-ytzero.md](../operator-ytzero.md)
- Open scope and exit gates: [roadmap.md](../roadmap.md)
- Device/kiosk expectations: [living-room-hardware.md](../living-room-hardware.md)
- Policy assumptions for child playback: [product-brief.md](../product-brief.md), `.cursor/skills/wonderfeed-content-policy/`

## Future regression (not implemented)

Goal: run a **smoke subset** of `OP-*` / `CP-*` in CI when Docker is available (health, pin, selected API gates), and keep `UI-*` / `DEV-*` in scheduled or pre-trial manual runs until browser/device harness exists. Do not fold operational UI walks into `make test` without an isolated compose fixture; see automation notes in the catalog.
