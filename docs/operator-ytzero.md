# Operate YT Zero from Wonderfeed

Wonderfeed runs the pinned YT Zero provider through a **host-owned** Docker Compose overlay. Provider source stays at [`providers/ytzero`](../providers/ytzero); durable data and env live on the host.

## Prerequisites

- Docker Desktop (or another Docker Engine) running
- Go 1.22+ (to build the host CLI)
- Git submodule initialized: `git submodule update --init --recursive`

## Quick start

```bash
make build
make serve
make provider-health
```

Open http://127.0.0.1:3001

Stop without wiping data:

```bash
make serve-down
```

## What the host owns

| Path | Role |
| --- | --- |
| [`deploy/ytzero/compose.yaml`](../deploy/ytzero/compose.yaml) | Compose overlay (image, ports, volume mount) |
| [`deploy/ytzero/.env.example`](../deploy/ytzero/.env.example) | Env template (copy to `.env`) |
| `deploy/ytzero/.env` | Local overrides (gitignored; created on first `provider up`) |
| `data/ytzero/` | SQLite DB, caches, downloads (gitignored except `.gitkeep`) |
| `bin/wonderfeed` | Host operator CLI |

## CLI

```text
wonderfeed version
wonderfeed help
wonderfeed provider status
wonderfeed provider up
wonderfeed provider down
wonderfeed provider health
wonderfeed provider open
```

`provider up` copies `.env.example` to `.env` when missing, creates `data/ytzero/`, probes Docker with a short timeout, then runs `docker compose -p wonderfeed-ytzero -f deploy/ytzero/compose.yaml up -d`.

`provider down` runs compose `down` **without** `-v`, so volumes and `data/ytzero` stay.

## Customize

1. Edit `deploy/ytzero/.env` (never commit it).
2. Useful keys are documented in `.env.example` and upstream [Configuration](https://github.com/Pelski/ytzero/wiki/Configuration).
3. Override the image with `YTZERO_IMAGE` when the default tag is wrong for your pin.
4. Child profiles, tags, Shorts/live gates, and watch limits are configured in the running YT Zero UI. Record product gaps in `docs/roadmap.md` Milestone 1 notes; do not patch Wonderfeed policy into the submodule.

## Image vs git pin

| Pin | Meaning |
| --- | --- |
| Git submodule `providers/ytzero` | Source and docs reference for the accepted ADR pin |
| Compose `YTZERO_IMAGE` (default `ghcr.io/pelski/ytzero:2026.09.8`) | Runtime container image |

Keep them aligned on bump PRs. Prefer changing the host compose default (or `.env`) when upstream publishes a matching tag; bump the gitlink in a separate reviewed change when you need submodule docs or API source at that revision.

## Boundary

- MUST NOT write Wonderfeed brand, family fixtures, or private paths into `providers/ytzero`.
- MUST NOT store runtime data under the submodule tree.
- Prefer this process boundary (HTTP + Docker) over linking host Go into the Bun app (AGPL).

See also: [architecture.md](architecture.md), [decisions/0001-ytzero-provider.md](decisions/0001-ytzero-provider.md), `.cursor/skills/wonderfeed-ytzero-ops/`.
