---
name: wonderfeed-ytzero-ops
description: >-
  Run and customize the pinned YT Zero provider from the Wonderfeed host:
  make serve, wonderfeed provider verbs, deploy/ytzero overlay, data/ytzero
  volumes, env files, health checks, and pin/image alignment. Use when starting
  YT Zero, fixing provider up/down, or changing host ops around providers/ytzero.
---

# Wonderfeed YT Zero ops

## When to load

Load when an operator or agent needs to start, stop, health-check, or configure the YT Zero provider **from the Wonderfeed repository root**, or when someone is about to `cd providers/ytzero` to run Docker by hand.

Related:

- [docs/operator-ytzero.md](../../../docs/operator-ytzero.md)
- [docs/decisions/0001-ytzero-provider.md](../../../docs/decisions/0001-ytzero-provider.md)
- `.cursor/skills/wonderfeed-provider-integration/SKILL.md`
- `.cursor/skills/process-compose-docker/SKILL.md` (timed Docker probe; no volume wipe on down)

## Core constraints

**CONSTRAINT:** Provider runtime data and env MUST live on the host under `data/ytzero/` and `deploy/ytzero/`. MUST NOT create or bind durable DB/cache volumes under `providers/ytzero/`.

- Enforcement: Inspect compose volume mounts and `.gitignore` before changing ops paths.
- Violation: STOP, move data/env to host paths, re-verify.

CORRECT:
```text
deploy/ytzero/compose.yaml mounts ../../data/ytzero:/data
```

PROHIBITED:
```text
cd providers/ytzero && docker compose up
# writes ./data inside the submodule
```

**CONSTRAINT:** Agents MUST operate the provider with the host CLI or Make verbs (`wonderfeed provider up|down|status|health`, `make serve` / `make serve-down`). MUST NOT document bare submodule compose as the primary Wonderfeed path.

- Enforcement: Operator docs and skill examples name host verbs first.
- Violation: STOP, rewrite steps to host CLI/Make, re-verify.

CORRECT:
```bash
make serve
make provider-health
```

PROHIBITED:
```bash
docker compose -f providers/ytzero/docker-compose.yml up -d
```

**CONSTRAINT:** `provider down` / `make serve-down` MUST stop containers without removing volumes (`docker compose down` without `-v`). MUST NOT wipe `data/ytzero` as part of a normal stop.

- Enforcement: Read down argv; confirm no `-v` / `volume rm` on the default path.
- Violation: STOP, remove volume destruction from the default down path.

**CONSTRAINT:** Docker preflight on up MUST use a bounded timeout. MUST NOT call bare unbounded `docker info` as the only probe.

- Enforcement: Code review of `ProbeDocker` / up path.
- Violation: STOP, add a timed probe, re-verify.

**CONSTRAINT:** Wonderfeed brand, family policy, and private layout MUST stay out of `providers/ytzero`. Customization is host `.env`, host docs, and in-app YT Zero settings; product gaps go to host roadmap notes.

- Enforcement: Diff review; no host spill in submodule.
- Violation: STOP, revert spill, apply host-only change.

CORRECT:
```text
Set YTZERO_AUTH_METHOD in deploy/ytzero/.env
Record "need parent approval queue" in docs/roadmap.md Milestone 1 gaps
```

PROHIBITED:
```text
Edit providers/ytzero UI copy to say Wonderfeed
```

**CONSTRAINT:** On image or git pin bumps, MUST record both the submodule SHA and the `YTZERO_IMAGE` value used for runtime. MUST NOT silently float `:latest` in host docs as the production default without noting drift risk.

- Enforcement: Check compose default and ADR/operator note on bump PRs.
- Violation: STOP, pin an explicit tag or digest, document alignment.

## Steps

1. **Init submodules** — `git submodule update --init --recursive`.
2. **Build CLI** — `make build`.
3. **Configure** — edit `deploy/ytzero/.env` after first `provider up` creates it from `.env.example`.
4. **Start** — `make serve` (timed Docker probe, then compose up).
5. **Verify** — `make provider-health` and open http://127.0.0.1:3001.
6. **Stop** — `make serve-down` (volumes kept).
7. **Gap notes** — after a household trial, write configure-vs-build gaps under Milestone 1 in `docs/roadmap.md`.

## Pre-completion checklist

- [ ] **Host paths:** Compose mounts `data/ytzero`; env under `deploy/ytzero`.
      Method: Read `deploy/ytzero/compose.yaml` and `.gitignore`.
      Pass: No durable volume under `providers/ytzero`.
      Fail: Move mounts → re-verify.
- [ ] **CLI surface:** Bare `wonderfeed` prints usage and does not start Docker.
      Method: Run `bin/wonderfeed` with no args.
      Pass: Usage on stderr, non-zero exit.
      Fail: Fix root command → re-test.
- [ ] **Down safety:** Default down does not pass `-v`.
      Method: Read `ytzero.Down` argv.
      Pass: No volume wipe.
      Fail: Remove `-v` → re-verify.
- [ ] **No submodule spill:** Ops change does not edit provider source for Wonderfeed branding.
      Method: `git -C providers/ytzero status`.
      Pass: Clean or unrelated intentional upstream work only.
      Fail: Revert spill.
