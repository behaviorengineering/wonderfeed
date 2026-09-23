---
name: wonderfeed-ytzero-ops
description: >-
  Run and customize the pinned YT Zero provider from the Wonderfeed host:
  make serve (process-compose), wonderfeed provider verbs, PostgreSQL + YT Zero
  compose overlay, data/postgres and data/ytzero volumes, env files, health
  checks, and pin/image alignment. Use when starting YT Zero, fixing provider
  up/down, or changing host ops around providers/ytzero.
---

# Wonderfeed YT Zero ops

## When to load

Load when an operator or agent needs to start, stop, health-check, or configure the YT Zero provider **from the Wonderfeed repository root**, or when someone is about to `cd providers/ytzero` to run Docker by hand.

Related:

- [docs/operator-ytzero.md](../../../docs/operator-ytzero.md)
- [docs/cloudflare.md](../../../docs/cloudflare.md) (remote HTTPS via Tunnel; not part of local serve)
- [docs/decisions/0001-ytzero-provider.md](../../../docs/decisions/0001-ytzero-provider.md)
- `.cursor/skills/wonderfeed-provider-integration/SKILL.md`
- `.cursor/skills/process-compose-docker/SKILL.md` (timed Docker probe; no volume wipe on down)

## Core constraints

**CONSTRAINT:** Provider runtime data and env MUST live on the host under `data/ytzero/`, `data/postgres/`, and `deploy/ytzero/`. MUST NOT create or bind durable DB/cache volumes under `providers/ytzero/`.

- Enforcement: Inspect compose volume mounts and `.gitignore` before changing ops paths.
- Violation: STOP, move data/env to host paths, re-verify.

CORRECT:
```text
deploy/ytzero/compose.yaml mounts ../../data/ytzero:/data and ../../data/postgres for PostgreSQL
```

PROHIBITED:
```text
cd providers/ytzero && docker compose up
# writes ./data inside the submodule
```

**CONSTRAINT:** The durable database MUST be PostgreSQL via `DATABASE_URL` in the host overlay. MUST NOT document SQLite as the shared/home-server default.

- Enforcement: Read `deploy/ytzero/compose.yaml` for `postgres` service and `DATABASE_URL`.
- Violation: STOP, restore Postgres wiring, re-verify.

**CONSTRAINT:** Agents MUST operate the provider with host Make/CLI verbs (`make serve` / `make serve-down`, `wonderfeed provider prepare|up|down|status|health`). MUST NOT document bare submodule compose as the primary Wonderfeed path.

- Enforcement: Operator docs and skill examples name host verbs first.
- Violation: STOP, rewrite steps to host CLI/Make, re-verify.

CORRECT:
```bash
make serve
make provider-health
# expect database: postgres
```

PROHIBITED:
```bash
docker compose -f providers/ytzero/docker-compose.yml up -d
```

**CONSTRAINT:** `provider down` / `make serve-down` MUST stop containers without removing volumes (`docker compose down` without `-v`). MUST NOT wipe `data/ytzero` or `data/postgres` as part of a normal stop.

- Enforcement: Read down argv and `scripts/pc-down.sh`; confirm no `-v` / `volume rm` on the default path.
- Violation: STOP, remove volume destruction from the default down path.

**CONSTRAINT:** Docker preflight on up MUST use a bounded timeout. MUST NOT call bare unbounded `docker info` as the only probe.

- Enforcement: Code review of `ProbeDocker` / `scripts/pc-up.sh`.
- Violation: STOP, add a timed probe, re-verify.

**CONSTRAINT:** PostgreSQL MUST NOT be published through Cloudflare Tunnel or other public ingress. Only the app HTTPS port is eligible for remote exposure (see `docs/cloudflare.md`).

- Enforcement: Compose publishes YT Zero port only; no Postgres host port mapping by default.
- Violation: STOP, remove public Postgres publish, re-verify.

**CONSTRAINT:** Wonderfeed brand, family policy, and private layout MUST stay out of `providers/ytzero`. Customization is host `.env`, host docs, and in-app YT Zero settings; product gaps go to host roadmap notes.

- Enforcement: Diff review; no host spill in submodule.
- Violation: STOP, revert spill, apply host-only change.

**CONSTRAINT:** On image or git pin bumps, MUST record the submodule SHA and the `YTZERO_IMAGE` / `POSTGRES_IMAGE` values used for runtime. MUST NOT silently float `:latest` in host docs as the production default without noting drift risk.

- Enforcement: Check compose defaults and ADR/operator note on bump PRs.
- Violation: STOP, pin an explicit tag or digest, document alignment.

## Steps

1. **Init submodules** — `git submodule update --init --recursive`.
2. **Build CLI** — `make build`.
3. **Configure** — ensure `deploy/ytzero/.env` has `POSTGRES_PASSWORD` (`provider prepare` / first up generates one from the example). Prefer `YTZERO_IMAGE=wonderfeed-ytzero:src-patch` (see `.env.example`).
4. **Start** — `make serve` or `make provider-up` (both run `provider-image` first).
5. **Verify** — `make provider-health` shows `"database": "postgres"`; open http://127.0.0.1:3001.
6. **Backup (optional)** — `wonderfeed backup create --local-only`, then `backup list`. Restore only with `--confirm`.
7. **Stop** — `make serve-down` (volumes kept).
8. **Remote access** — follow `docs/cloudflare.md` separately; do not expose Postgres.
9. **Gap notes** — after a household trial, write configure-vs-build gaps under Milestone 1 in `docs/roadmap.md`.

**Cold start:** Upstream image `2026.09.8` alone must not be the default for empty Postgres. Host uses `deploy/ytzero/Dockerfile.src-patch` until an upstream tag includes the migration fixes.

**Backups:** Encrypt Postgres dumps + portable `data/ytzero` state with age. MUST exclude downloads, imgcache, logs, and cookie dirs. MUST require `--confirm` on restore and write a safety snapshot first.

## Pre-completion checklist

- [ ] **Host paths:** Compose mounts `data/ytzero` and `data/postgres`; env under `deploy/ytzero`.
      Method: Read `deploy/ytzero/compose.yaml` and `.gitignore`.
      Pass: No durable volume under `providers/ytzero`.
      Fail: Move mounts → re-verify.
- [ ] **PostgreSQL:** Health reports `database: postgres`.
      Method: `wonderfeed provider health`.
      Pass: JSON field `database` is `postgres`.
      Fail: Fix `DATABASE_URL` / Postgres service → re-test.
- [ ] **CLI surface:** Bare `wonderfeed` prints usage and does not start Docker.
      Method: Run `bin/wonderfeed` with no args.
      Pass: Usage on stderr, non-zero exit.
      Fail: Fix root command → re-test.
- [ ] **Down safety:** Default down does not pass `-v`.
      Method: Read `ytzero.Down` and `scripts/pc-down.sh`.
      Pass: No volume wipe.
      Fail: Remove `-v` → re-verify.
- [ ] **No submodule spill:** Ops change does not edit provider source for Wonderfeed branding.
      Method: `git -C providers/ytzero status`.
      Pass: Clean or unrelated intentional upstream work only.
      Fail: Revert spill.
