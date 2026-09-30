# Windows living-room appliance

**Status:** deploy via private **wonderfeed-local** (GitLab), same two-repo pattern as Polypus + polypus-local.

| Repo | Role |
| --- | --- |
| [behaviorengineering/wonderfeed](https://github.com/behaviorengineering/wonderfeed) (this tree) | Product: CLI, control plane, `deploy/ytzero` overlay sources, docs |
| **wonderfeed-local** ([gitlab.com/mindhoc/wonderfeed-local](https://gitlab.com/mindhoc/wonderfeed-local), private) | Appliance CD: `images.env`, compose, `go run ./cmd/wonderfeed-local deploy`, Edge kiosk runbook |

Develop on Mac with `make serve` / `make provider-up` ([operator-ytzero.md](operator-ytzero.md)). Ship to the Windows PC through wonderfeed-local pipelines (`deploy:windows`, runner tag `windows-home`).

## Secrets and config (not in git)

Household `config.yaml`, `.env.deploy`, Postgres passwords, and session cookies stay on the appliance host (OS credential store + `%USERPROFILE%\.config\wonderfeed\`). Wonderfeed-local documents setup in its `docs/PROVE-CD.md`. **Do not** commit or paste real config into either repository.

## Day-1 appliance shape

1. `go run ./cmd/wonderfeed-local deploy` (Postgres + YT Zero src-patch image, health on `http://127.0.0.1:3001`).
2. `wonderfeed control migrate up` and `wonderfeed control serve` on the host (product CLI).
3. Edge kiosk on `http://127.0.0.1:3001` (wonderfeed-local `docs/kiosk.md`).

Image pins and on-runner src-patch build: wonderfeed-local `images.env` (`POSTGRES_IMAGE`, `YTZERO_BASE_IMAGE`, `YTZERO_SRC_PATCH_IMAGE`). Set `WONDERFEED_PRODUCT_DIR` to a product git checkout on the Windows host.

## Validation

[testing/operational-tests.md](testing/operational-tests.md): `OP-001`–`OP-002`, `DEV-001`–`DEV-002`, then trusted-set import `UI-001` when ready.

## Related

- [living-room-hardware.md](living-room-hardware.md) (kiosk ladder, Shell Launcher)
- [roadmap.md](roadmap.md) Milestone 4
- Product operator runbook: [operator-ytzero.md](operator-ytzero.md)
