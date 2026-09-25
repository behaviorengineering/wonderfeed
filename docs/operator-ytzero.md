# Operate YT Zero from Wonderfeed

Wonderfeed runs the pinned YT Zero provider through a **host-owned** Docker Compose overlay with **PostgreSQL**. Provider source stays at [`providers/ytzero`](../providers/ytzero); durable data and env live on the host.

Remote access design (Cloudflare Tunnel, school rollouts): [`cloudflare.md`](cloudflare.md).

## Prerequisites

- Docker Desktop (or another Docker Engine) running
- Go 1.22+ (to build the host CLI)
- [process-compose](https://github.com/F1bonacc1/process-compose) on `PATH` for `make serve` (optional if you only use detached `provider up`)
- Git submodule initialized: `git submodule update --init --recursive`

## Quick start

`make serve` / `make provider-up` build a **src-patch** image (pinned release layers + `deploy/ytzero/src-overlay` Postgres cold-start fixes) so empty Postgres boots cleanly. Upstream `2026.09.8` alone crash-loops (image_cache import race and table-level FOREIGN KEY translation).

Interactive (process-compose TUI):

```bash
make serve
```

Detached:

```bash
make provider-up
make provider-health
```

Open http://127.0.0.1:3001

Health should report `"database": "postgres"`.

Stop without wiping data:

```bash
make serve-down
# or: make provider-down
```

## What the host owns

| Path | Role |
| --- | --- |
| [`deploy/ytzero/compose.yaml`](../deploy/ytzero/compose.yaml) | Compose overlay (Postgres + YT Zero image, ports, mounts) |
| [`deploy/ytzero/.env.example`](../deploy/ytzero/.env.example) | Env template (`POSTGRES_PASSWORD` required) |
| `deploy/ytzero/.env` | Local overrides (gitignored; created on prepare/up) |
| `data/postgres/` | Parent for PostgreSQL; cluster lives in `data/postgres/pgdata/` (gitignored except `.gitkeep`) |
| `data/ytzero/` | YT Zero files: caches, logs, downloads, state marker (gitignored except `.gitkeep`) |
| `data/backups/` | Encrypted age backups + age identity (gitignored except `.gitkeep`) |
| [`process-compose.yaml`](../process-compose.yaml) | Interactive stack lifecycle |
| `bin/wonderfeed` | Host operator CLI |

## CLI

```text
wonderfeed version
wonderfeed help
wonderfeed provider status
wonderfeed provider prepare
wonderfeed provider up
wonderfeed provider down
wonderfeed provider health
wonderfeed provider open
wonderfeed backup create [--local-only]
wonderfeed backup list
wonderfeed backup restore <id> --confirm
```

`provider prepare` creates `data/ytzero/`, `data/postgres/`, and `.env` (with a generated `POSTGRES_PASSWORD` when copying from the example).

`provider up` runs prepare, then `docker compose up -d`.

`provider down` / `make serve-down` run compose `down` **without** `-v`.

## Encrypted state backups

```bash
make build
./bin/wonderfeed backup create --local-only
./bin/wonderfeed backup list
./bin/wonderfeed backup restore <id> --confirm
```

Create runs `pg_dump` in the Postgres container, packs small portable files under `data/ytzero/` (avatars, `database-state.json`, …), encrypts with [age](https://age-encryption.org/), and writes `data/backups/objects/<id>.age`. Downloads, image cache, logs, cookies, and SQLite dirs are excluded by default.

Restore is destructive: it requires `--confirm`, writes a safety snapshot under `data/backups/safety/`, then replaces the public schema and portable state files.

Optional S3-compatible upload (R2/B2/MinIO) uses `WONDERFEED_BACKUP_S3_*` in `.env` (see `.env.example`). Keep the age identity file offline-safe; without it you cannot decrypt.

## Customize

1. Edit `deploy/ytzero/.env` (never commit it).
2. Required: `POSTGRES_PASSWORD` (avoid `@ : / ? #` so `DATABASE_URL` stays valid).
3. Override images with `YTZERO_IMAGE` / `POSTGRES_IMAGE` when needed.
4. Child profiles and feed policy stay in the YT Zero UI; record product gaps in `docs/roadmap.md`.

## Control plane allowlist sync

When running `wonderfeed control serve`, set `YTZERO_SESSION_COOKIE` to a
household **admin** session cookie (`ytzero_session=...` from logging in as the
primary profile or another administrator). The adapter syncs host allowlists
through YT Zero `POST /api/channels/reconcile`; it does not impersonate the
child profile or require a child-lock unlock PIN.

Rebuild `wonderfeed-ytzero:src-patch` after bumping the `providers/ytzero`
gitlink so the running image includes the reconcile route
(`make provider-image`).

## Switching from SQLite

This overlay always sets `DATABASE_URL`. Upstream YT Zero will initialize a **new** PostgreSQL database on a clean install. If `data/ytzero` already has a populated SQLite file from an older host run, YT Zero refuses automatic switch; use YT Zero **Settings → Dangerous → Database** to migrate, or start fresh by stopping the stack and removing the old SQLite files under `data/ytzero/db/` (only if you accept data loss).

## Image vs git pin

| Pin | Meaning |
| --- | --- |
| Git submodule `providers/ytzero` | Source and docs reference for the accepted ADR pin (`2026.09.8` / matching SHA) |
| `deploy/ytzero/src-overlay/` | Host-owned Postgres cold-start patches applied by `Dockerfile.src-patch` until upstream ships them |
| `providers/ytzero` @ pinned SHA | `channelRoutes.ts` copied into src-patch image for admin allowlist reconcile until upstream image includes it |
| `make provider-image` → `wonderfeed-ytzero:src-patch` | Host default runtime: base `ghcr.io/pelski/ytzero:2026.09.8` + src-overlay |
| Compose `YTZERO_IMAGE` | Override; `.env.example` defaults to `wonderfeed-ytzero:src-patch` |
| Compose `POSTGRES_IMAGE` (default `postgres:17-alpine`) | Runtime database image |

After changing `deploy/ytzero/src-overlay/`, re-run `make provider-image` (or `make provider-up` / `make serve`, which rebuild).

## Boundary

- MUST NOT write Wonderfeed brand, family fixtures, or private paths into `providers/ytzero`.
- MUST NOT store runtime data under the submodule tree.
- MUST NOT publish PostgreSQL through Cloudflare Tunnel (app HTTPS only; see [`cloudflare.md`](cloudflare.md)).
- Prefer this process boundary (HTTP + Docker) over linking host Go into the Bun app (AGPL).

See also: [architecture.md](architecture.md), [decisions/0001-ytzero-provider.md](decisions/0001-ytzero-provider.md), `.cursor/skills/wonderfeed-ytzero-ops/`.
