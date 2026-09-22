# Wonderfeed product brief

## Problem

YouTube is excellent at surfacing more video and poor at staying out of the way for children. Parent tools that only filter the stock YouTube UI still leave Home, Shorts, search, autoplay, and end screens in the child's path. Pure allowlist apps often fail to deliver a living feed of new uploads from trusted channels.

Wonderfeed targets the gap: a **parent-owned recommender and control layer** that uses hosting platforms for playback, while the family owns channel selection, approval, queue rules, and session limits.

## Target users

| Role | Need |
| --- | --- |
| Parent / guardian | Curate trusted channels, approve exceptions, set time and format rules, see what was watched |
| Child | Receive a calm feed of fresh, allowed content without open-ended YouTube browsing |
| Operator (self-host) | Run a durable inbox/control plane on family infrastructure |

## Product principles

1. **Parent owns the algorithm.** Feed composition is explicit rules and approvals, not a black-box engagement model.
2. **Allowlist first.** Unknown channels and videos do not play until policy allows them.
3. **Push fresh trusted content.** The child surface should feel alive with new uploads from approved sources, not only a static library.
4. **Separate concerns.** Channel curation, parent approval, device/browser enforcement, and playback entitlement are different layers. Do not pretend one embed parameter solves all of them.
5. **Providers stay portable.** Upstream tools like YT Zero remain independent. Wonderfeed adapts; it does not spill host brand into provider trees.
6. **Fail closed for children.** Ambiguous policy means no playback, not "probably fine."

## Non-goals (near term)

- Replacing YouTube as a video host or CDN.
- Scraping or proxying Google account credentials for children through Wonderfeed.
- Building a general-purpose adult YouTube client.
- Training a large-scale open-web video safety classifier as the MVP.
- Committing Wonderfeed product policy into `providers/ytzero`.

## First-release success criteria

A first useful release is successful when:

1. Parents can define a trusted channel set and see new uploads land in a controlled inbox/feed.
2. Children can watch only from that approved set (or parent-approved exceptions) through a locked-down presentation path.
3. Shorts, live, and open search are policy-gated rather than default.
4. Watch-time or session limits can stop or pause further autoplay.
5. Provider state (subscriptions, tags, rules) remains inspectable and exportable rather than locked inside YouTube's account graph.

## Vocabulary

| Term | Meaning |
| --- | --- |
| Host | This repository: Wonderfeed product, docs, adapters, and agent skills |
| Provider | An upstream system mounted under `providers/` that supplies feed/inbox/player capabilities |
| Control plane | Parent-facing policy, approval, and session rules |
| Child surface | The locked-down presentation children use to watch |
| Allowlist | Explicit set of channels and/or videos permitted to play |
| Feed policy | Rules that shape order, diversity, format filters, and session length |

## Canonical references

- Architecture: [architecture.md](architecture.md)
- Roadmap: [roadmap.md](roadmap.md)
- Provider decision: [decisions/0001-ytzero-provider.md](decisions/0001-ytzero-provider.md)
- Upstream YT Zero: https://github.com/Pelski/ytzero
