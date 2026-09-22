# Wonderfeed

Wonderfeed is a **parent-owned control plane for kids' video watching**. It aims to replace YouTube's recommendation loop with a rules-based feed of content parents deliberately allow, while still using YouTube (or similar hosts) for actual playback.

This repository is the **host product**. Provider integrations live under `providers/`. The first provider foundation is [YT Zero](https://github.com/Pelski/ytzero), mounted at [`providers/ytzero`](providers/ytzero).

## Product thesis

Parents should be able to:

- Choose trusted channels and approved videos.
- Push a calm, chronological, policy-shaped feed to children.
- Keep discovery, ranking, approval, and session rules under family control.
- Avoid treating YouTube Home, Shorts, or search as the child's default surface.

Wonderfeed owns product policy, parent workflows, and future adapters. YT Zero owns subscription-inbox and rules-based feed machinery today. Wonderfeed must not invent host branding or private layout inside the provider submodule.

Canonical docs:

| Doc | Purpose |
| --- | --- |
| [docs/product-brief.md](docs/product-brief.md) | Problem, users, principles, non-goals, success criteria |
| [docs/architecture.md](docs/architecture.md) | Host vs provider boundary, feed/playback model, seams |
| [docs/roadmap.md](docs/roadmap.md) | Milestones, risks, decision gates |
| [docs/operator-ytzero.md](docs/operator-ytzero.md) | Run and customize YT Zero from this repo |
| [docs/decisions/0001-ytzero-provider.md](docs/decisions/0001-ytzero-provider.md) | Why YT Zero is the first provider pin |

## Repository layout

```text
wonderfeed/
  cmd/wonderfeed/    # host operator CLI
  internal/          # host libraries (provider ops, errors)
  deploy/ytzero/     # host compose overlay and env template
  data/ytzero/       # local provider data (gitignored)
  providers/
    ytzero/          # git submodule: Pelski/ytzero (do not spill host policy here)
  docs/              # product and architecture documentation
  .cursor/
    packs/shared/    # git submodule: shared Cursor packs
    skills/          # pack symlinks + host-owned Wonderfeed skills
```

## Clone with submodules

```bash
git clone --recurse-submodules https://github.com/behaviorengineering/wonderfeed.git
cd wonderfeed
```

If you already cloned without submodules:

```bash
git submodule update --init --recursive
```

After updating the packs pin, refresh Cursor links:

```bash
bash .cursor/packs/shared/scripts/link-into-project.sh --project .
```

## Run the provider (Milestone 1)

```bash
make build
make serve
make provider-health
```

Open http://127.0.0.1:3001. Stop with `make serve-down` (data kept). Full operator notes: [docs/operator-ytzero.md](docs/operator-ytzero.md).

## Provider boundary (critical)

- Treat `providers/ytzero` as an independent repository.
- Put Wonderfeed-specific skills, brand, policy, and private paths in this host only.
- Prefer wrapping, adapting, or composing the provider over editing upstream for product needs.
- See repository-boundary rules in `.cursor/rules/repository-boundaries.mdc` and the host skills `.cursor/skills/wonderfeed-provider-integration/` and `.cursor/skills/wonderfeed-ytzero-ops/`.

## Current status

Milestone 1 ops scaffold:

- Public host repository and product docs.
- YT Zero pinned as a provider submodule.
- Host Cursor skills for product context, provider integration, content policy, and YT Zero ops.
- Host Go CLI (`wonderfeed`) and compose overlay to run the provider without writing into the submodule.

Runtime control plane, authentication product model, and the exact child-facing UI are **not** decided yet. See the [roadmap](docs/roadmap.md).

## License note

Host documentation, skills, and Go tooling in this repository are authored for Wonderfeed. The `providers/ytzero` tree remains under its upstream license (AGPL-3.0). Treat provider license obligations carefully before shipping derived binaries or services. Prefer the Docker/HTTP process boundary documented in [docs/operator-ytzero.md](docs/operator-ytzero.md).

