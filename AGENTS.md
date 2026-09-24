# Agents — Wonderfeed host

Humans read [README.md](README.md) (elevator pitch only). This file is the operator entry for AI copilots working in this repository.

Wonderfeed is the **host product**. Provider code lives under `providers/` as separate repositories. Do not spill Wonderfeed brand, private paths, or family policy into provider trees.

## Load order (before coding)

1. [docs/product-brief.md](docs/product-brief.md) — principles, non-goals, vocabulary
2. [docs/architecture.md](docs/architecture.md) — host vs provider seams, target product flow
3. [docs/roadmap.md](docs/roadmap.md) — what is done vs deferred
4. Matching host skill(s) below for the task at hand

## Host skills (Cursor)

| When | Skill |
| --- | --- |
| Product scope, naming, parent/child roles | [.cursor/skills/wonderfeed-product-context/SKILL.md](.cursor/skills/wonderfeed-product-context/SKILL.md) |
| Allowlist, fail-closed, child surface, embeds honesty | [.cursor/skills/wonderfeed-content-policy/SKILL.md](.cursor/skills/wonderfeed-content-policy/SKILL.md) |
| Editing `providers/`, pins, no product spill | [.cursor/skills/wonderfeed-provider-integration/SKILL.md](.cursor/skills/wonderfeed-provider-integration/SKILL.md) |
| `make serve`, Postgres, compose, backups, health | [.cursor/skills/wonderfeed-ytzero-ops/SKILL.md](.cursor/skills/wonderfeed-ytzero-ops/SKILL.md) |

Shared pack skills (golang-quality, process-compose-docker, and so on) live under `.cursor/packs/shared/` and are soft-linked into `.cursor/skills/`. Edit packs only via [edit-cursor-packs](.cursor/skills/edit-cursor-packs/SKILL.md).

## Setup and local stack

```bash
git submodule update --init --recursive
make build
make provider-up          # or: make serve
make provider-health      # expect "database": "postgres"
```

Open http://127.0.0.1:3001. Stop with `make serve-down` or `make provider-down` (volumes kept).

Full runbook: [docs/operator-ytzero.md](docs/operator-ytzero.md).

Encrypted backups: `make backup-create` / `wonderfeed backup list`.

YouTube API terms refresh (local snapshots, gitignored): `make youtube-api-terms-refresh` — see [docs/legal/youtube-api/README.md](docs/legal/youtube-api/README.md).

## Hard boundaries

- **MUST** treat `providers/ytzero` as its own git repo (`git -C providers/ytzero rev-parse --show-toplevel` before edits). Commit there first; host only bumps the gitlink.
- **MUST NOT** write Wonderfeed product policy, brand, or private layout into `providers/ytzero`.
- **MUST NOT** invent adult YouTube-client features that contradict the product brief (Home feed clones, unrestricted child search, and so on).
- **MUST** fail closed for children: unknown content does not play until policy or parent approval allows it.
- Prefer host adapters and docs over forking provider internals for product needs.

## Vision docs (not Milestone 1 ops)

| Doc | Purpose |
| --- | --- |
| [docs/curated-library.md](docs/curated-library.md) | Hosted curated library, voice concierge/librarian flow, API compliance notes |
| [docs/cloudflare.md](docs/cloudflare.md) | Remote HTTPS / tunnel design for homes |
| [docs/decisions/0001-ytzero-provider.md](docs/decisions/0001-ytzero-provider.md) | Why YT Zero is the first provider |

Parent concierge, child librarian, and central evaluation are **vision**. Do not claim they are shipped. Implement only after an explicit request and roadmap placement.

## Verification

```bash
make build
make test
make vet
make provider-health   # when the stack is up
```

Prefer focused checks over inventing new mega-scripts. Use existing Makefile verbs from `make help`.

## Git practice for agents

- Do not edit on `main` / `master` / `develop` / `trunk`; create or stay on a feature branch.
- If the current branch already has an open PR, ask whether to continue that PR or merge first before starting a sibling branch.
- Commit only when the human asks. Do not push unless asked.
