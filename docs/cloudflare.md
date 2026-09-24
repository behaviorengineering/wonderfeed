**Plan:** use one product-controlled Cloudflare account, one Pro zone, one remotely managed tunnel per home, one first-level hostname per home, and your own app login for families; keep Cloudflare Access mainly for staff/admin unless you later decide its seat pricing is worth it. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/account-limits/)

## Shape

Use this naming and ownership model:

- Cloudflare account: owned by your company or the school program operator, not by families. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/self-hosted-public-app/)
- Zone: one product-owned zone such as `homes.wonderfeed.app`.
- Household hostname: `<household-id>.homes.wonderfeed.app`.
- Tunnel: one remotely managed Cloudflare Tunnel per household mini PC. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/do-more-with-tunnels/local-management/)

That keeps the hostname under one level below the zone, which matters because Universal SSL on a full Cloudflare setup covers the apex and first-level subdomains, but not deeper names. [developers.cloudflare](https://developers.cloudflare.com/ssl/edge-certificates/universal-ssl/limitations/)

## Why this layout

This is the cleanest low-cost pattern because:

| Part | Choice | Why |
|---|---|---|
| DNS/SSL | One Pro zone | Pro is US$25/month per zone and gives 3,500 DNS records, which is enough for 200–800 homes with headroom.  [cloudflare](https://www.cloudflare.com/plans/pro/) |
| Home connectivity | One tunnel per home | Cloudflare documents 1,000 tunnels and 1,000 routes per account, so one school cohort fits.  [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/account-limits/) |
| Public naming | One hostname per home | Simple routing, simple support, easy replacement if a box dies.  [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/get-started/create-remote-tunnel/) |
| Identity | Your app login for parents/kids | Avoids turning Cloudflare Access seat pricing into your main consumer auth bill. Cloudflare counts active users per authentication event.  [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/team-and-resources/users/seat-management/) |
| Staff protection | Cloudflare Access | Good for admin, support, internal tools and emergency paths.  [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/self-hosted-public-app/) |

## Naming rules

Use names like:

```text
42.homes.wonderfeed.app
b7f3.homes.wonderfeed.app
schoolA-0042.homes.wonderfeed.app
```

Avoid this unless `kids.school.edu` is separately delegated as its own Cloudflare zone:

```text
family-42.kids.school.edu
```

If `school.edu` is the zone, that hostname is too deep for normal Universal SSL coverage on a full setup zone. [developers.cloudflare](https://developers.cloudflare.com/ssl/edge-certificates/universal-ssl/limitations/)

## Provisioning flow

Provision each home like this:

1. Create a remotely managed tunnel through the API or Terraform. Cloudflare supports remote tunnel creation by API and stores remote tunnel config in Cloudflare. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/get-started/create-remote-tunnel-api/)
2. Fetch the tunnel token for that tunnel. Cloudflare documents token retrieval for remotely managed tunnels. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/configure-tunnels/remote-tunnel-permissions/)
3. Store the token in your provisioning system, linked to the household and appliance serial number.
4. Create the household DNS hostname in your zone, proxied through Cloudflare. Cloudflare supports proxied `CNAME` records, and only `A`, `AAAA`, and `CNAME` can be proxied. [developers.cloudflare](https://developers.cloudflare.com/dns/manage-dns-records/how-to/create-dns-records/)
5. Add the tunnel’s published application route to the local app, usually `http://localhost:<port>`. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/get-started/create-remote-tunnel/)
6. Install `cloudflared` on the mini PC and run it with that household’s token. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/downloads/)

In practice, your home installer can do something like:

```text
cloudflared service install <tunnel-token>
```

and then your control plane marks the device as online once the connector appears in Cloudflare. The family never sees Cloudflare at all. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/configure-tunnels/tunnel-availability/deploy-replicas/)

## Household box design

Each home box should run:

- Your local HTTP app.
- PostgreSQL.
- `cloudflared`.
- A small local agent for updates, health, config sync and safe restart.

Recommended local network model:

- Wonderfeed control plane listens on `127.0.0.1:8080` by default (`wonderfeed control serve`).
- YT Zero app listens on `127.0.0.1:3001` (provider UI/API; not the public product contract).
- PostgreSQL is published to host loopback only (`127.0.0.1:5432`), not to the LAN.
- `cloudflared` forwards the public household hostname **only** to the Wonderfeed control-plane port (`http://localhost:8080`), never to port `5432` or port `3001`.
- No inbound router port forwarding at all.

That aligns with Tunnel’s purpose: Cloudflare Tunnel connects the origin outward to Cloudflare without a public IP on the home side. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/)

### Tunnel ingress (home box)

Publish only the authenticated Wonderfeed control API:

```text
hostname: <household-id>.homes.wonderfeed.app
service:  http://localhost:8080
```

Do not add ingress rules for PostgreSQL or raw YT Zero. Parent calls require `Authorization: Bearer <WONDERFEED_PARENT_AUTH_KEY>` when the control plane is reachable beyond loopback.

## Control plane

Build a small central control plane outside the home devices. It should own:

- School.
- Household.
- Device serial / activation code.
- Cloudflare tunnel ID.
- Tunnel token issuance and rotation state.
- Hostname.
- Device status, last seen, version.
- Recovery and replacement workflow.

Suggested tables:

| Table | Key fields |
|---|---|
| `schools` | `id`, `name`, `district_id`, `cloudflare_account_shard` |
| `households` | `id`, `school_id`, `slug`, `hostname`, `status` |
| `devices` | `id`, `household_id`, `serial`, `activation_code`, `last_seen_at`, `software_version` |
| `cf_tunnels` | `household_id`, `cf_tunnel_id`, `cf_token_ref`, `cf_route_hostname`, `created_at`, `rotated_at` |
| `users` | `id`, `household_id`, `role`, `email`, `password_hash_or_oidc_ref` |

Keep the Cloudflare API token only in the control plane. Never ship it to the mini PCs. Ship only the home-specific tunnel token. Cloudflare’s tunnel token is the secret that allows that tunnel to run. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/configure-tunnels/remote-tunnel-permissions/)

## Authentication design

For families, use your own auth inside Wonderfeed.

Why:

- Cloudflare Access charges by active users, not by apps. A user consumes a seat after an authentication event. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/team-and-resources/users/seat-management/)
- Parents and children are customer identities, not internal workforce identities.
- Your app already needs household-level roles, permissions and parental controls.

So use this split:

| User type | Authentication path |
|---|---|
| Parents/kids | Wonderfeed login in the app |
| School operator/admin/support | Cloudflare Access in front of admin/support endpoints |
| Emergency maintenance endpoints | Cloudflare Access |

You can still put a wildcard Access app in front of `*.homes.wonderfeed.app`, but if you do, treat it as a coarse gate only. Cloudflare allows wildcard hostnames for self-hosted/public Access apps. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/self-hosted-public-app/index.md)

The catch is that one wildcard app applies the same Access policy shape across those hostnames, so it is not your real tenant isolation layer. The app must still verify who belongs to which household. Cloudflare explicitly says to validate the Access token at the origin. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/configure-apps/self-hosted-public-app/)

For your use case, the simpler version is often better:

- Family hostnames: app login only.
- Admin hostnames like `admin.wonderfeed.app` and `support.wonderfeed.app`: protected by Access.

## Hostname layout

Use separate hostnames for separate concerns:

```text
<home-id>.homes.wonderfeed.app         # family-facing app
admin.wonderfeed.app                   # internal admin UI
support.wonderfeed.app                 # support tooling
provision.wonderfeed.app               # device activation API
telemetry.wonderfeed.app               # device heartbeat API
```

That lets you keep Access focused where it adds the most value. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/)

## Cloudflare account setup

Set up the account like this:

1. Add the product domain/zone to Cloudflare.
2. Upgrade that zone to Pro.
3. Enable Universal SSL.
4. Create a least-privilege API token for tunnel and DNS automation.
5. Create remote tunnel automation through API or Terraform.
6. Add admin/support Access apps.
7. Turn on “require Access protection” for sensitive internal hostnames if you use those paths. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/self-hosted-public-app/)

Important limits to design around:

| Limit | Current documented value |
|---|---:|
| Tunnels per account | 1,000  [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/account-limits/) |
| Routes per account | 1,000  [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/account-limits/) |
| Access applications | 500  [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/account-limits/) |
| DNS records, Free zone created after 2024-09-01 | 200  [developers.cloudflare](https://developers.cloudflare.com/dns/manage-dns-records/) |
| DNS records, Pro zone | 3,500  [developers.cloudflare](https://developers.cloudflare.com/dns/manage-dns-records/) |

That means one Pro zone is fine for one school, and one account is fine for roughly one school cohort, but not for indefinite scaling. [developers.cloudflare](https://developers.cloudflare.com/dns/manage-dns-records/)

## Remote install flow

A practical appliance onboarding flow:

1. Factory-image or USB-image the mini PC with:
- your app,
- Postgres,
- `cloudflared`,
- device bootstrap service.

2. On first boot, device shows:
- QR code,
- 8–12 character activation code,
- local LAN URL for fallback setup.

3. Parent or school enters activation code into your provisioning portal.

4. Control plane:
- creates tunnel,
- creates DNS hostname,
- returns tunnel token + config payload.

5. Device downloads:
- tunnel token,
- app config,
- school/household assignment,
- update channel.

6. Device starts `cloudflared`, checks in, and marks itself healthy.

This minimizes support because the Cloudflare work is fully centralized. Cloudflare documents remote tunnel creation and token-based connector install for remotely managed tunnels. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/get-started/create-remote-tunnel-api/)

## Replacement and rotation

You need clear lifecycle rules:

### Replace dead box
- Create new device record.
- Reuse same household hostname.
- Either reuse tunnel if safe, or create a fresh tunnel and repoint the hostname route.
- Old token is revoked/rotated.

### Rotate token
- Control plane requests a new token or replaces the tunnel.
- Device fetches updated secret via authenticated check-in.
- Device restarts `cloudflared`.
- Mark old token invalid in your records.

### Rename household
- Change your internal label only if possible.
- Keep public hostname stable unless there is a strong reason to change it.

Stable hostnames reduce support burden.

## Monitoring

Track both device health and public reachability.

Minimum telemetry:

- `cloudflared` connected/disconnected.
- Last successful heartbeat.
- App health endpoint.
- DB health.
- Disk usage.
- Local queue depth if syncing data.
- Version drift.

Useful operational states:

| State | Meaning |
|---|---|
| `pending` | Ordered but not yet activated |
| `activating` | Booted, waiting for tunnel/connectivity |
| `healthy` | Tunnel and app healthy |
| `degraded` | Tunnel up, app failing; or app up, sync failing |
| `offline` | No heartbeat |
| `replaced` | Device retired |

Cloudflare also exposes tunnel status and supports multiple replicas per tunnel, but for a home appliance the normal pattern is one replica on one box. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/account-limits/)

## Security boundaries

Use these boundaries:

- Cloudflare account is operator-owned only.
- Devices hold tunnel token, not Cloudflare API token. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/configure-tunnels/remote-tunnel-permissions/)
- App auth is household-scoped.
- School admin roles are separate from family roles.
- Support access is time-limited and audited.
- Device never accepts inbound public traffic directly.
- Local admin/debug ports are bound to localhost unless explicitly needed.

Also assume the mini PC is semi-trusted, not fully trusted. Parents physically possess it. So:

- Encrypt sensitive local secrets at rest where practical.
- Avoid long-lived master secrets on the box.
- Support remote revoke/re-enrol.
- Sign software updates.

## School rollout plan

### Phase 1: pilot, 20–50 homes

Use:
- one Pro zone,
- one Cloudflare account,
- one tunnel per home,
- no Cloudflare Access for family users,
- Access only for admin/support.

Goal:
- prove appliance activation,
- prove home networking reliability,
- prove support workflow.

### Phase 2: first real school, 200–800 homes

Same model, but add:

- batch provisioning jobs,
- device inventory,
- automated rotation,
- alerting,
- spare hostname/tunnel headroom,
- staged rollout by class or cohort.

Operational rule:
- stop onboarding into a shard when it gets near 700–800 homes.
- leave buffer below the 1,000 tunnel/route ceiling for test devices, replacements and incidents. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/account-limits/)

## Later multi-district scale

Shard by district or regional cohort.

Pattern:

```text
Account A / Zone A -> Districts 1-3
Account B / Zone B -> Districts 4-6
Account C / Zone C -> Districts 7-9
```

Reasons:

- limits reset per account. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/account-limits/)
- incidents stay contained.
- DNS and tunnel churn stay smaller.
- migrations become easier.

At that stage, consider:

- Enterprise Organizations for central management of multiple accounts.
- Cloudflare for SaaS only if districts want vanity domains like `homeapp.district.edu`. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/self-hosted-public-app/)

Do **not** start with Cloudflare for SaaS unless vanity domains are a real requirement. Your product-owned zone is simpler and cheaper.

## DNS and SSL details

Use proxied `CNAME` records for household hostnames where appropriate. Cloudflare supports proxied `CNAME` records, and proxied hostnames receive Cloudflare edge behavior. [developers.cloudflare](https://developers.cloudflare.com/ssl/edge-certificates/universal-ssl/enable-universal-ssl/)

For SSL:

- `homes.wonderfeed.app` as the zone means `42.homes.wonderfeed.app` is first-level and covered.
- `wonderfeed.app` as the zone means `42.homes.wonderfeed.app` is deeper and not covered by normal Universal SSL on full setup. [developers.cloudflare](https://developers.cloudflare.com/ssl/edge-certificates/universal-ssl/)

So the best naming trick is:

```text
Create zone: homes.wonderfeed.app
Then use:   42.homes.wonderfeed.app
```

That is the cheapest clean way to keep everything certificate-safe.

## Cost view

Base networking cost is very low:

| Item | Likely cost |
|---|---:|
| One Pro zone | US$25/month or US$20/month effective annual billing  [cloudflare](https://www.cloudflare.com/plans/) |
| Tunnel per home | covered under the normal Tunnel model, subject to account limits  [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/account-limits/) |
| Family subdomains | no extra charge per subdomain; paid plans are per domain/zone  [cloudflare](https://www.cloudflare.com/plans/faq/) |

The real future cost decision is Access seats if you use Access for every family user. Since Cloudflare counts active users for seats, that can grow much faster than the zone cost. [cloudflare](https://www.cloudflare.com/plans/zero-trust-services/)

## Recommended exact design

For your case, I would use this:

| Layer | Recommendation |
|---|---|
| Public domain | `homes.wonderfeed.app` as its own Cloudflare zone |
| Zone plan | Pro |
| Remote access | one remotely managed tunnel per home |
| Family hostname | `<household-id>.homes.wonderfeed.app` |
| Family auth | Wonderfeed login |
| Staff/admin auth | Cloudflare Access |
| Provisioning | API-driven from central control plane |
| Growth boundary | one Cloudflare account/zone shard per ~700–800 homes |

## Do this next

1. Buy or choose the product domain and decide whether `homes.wonderfeed.app` will be the Cloudflare zone.
2. Create one Pro zone in Cloudflare. [cloudflare](https://www.cloudflare.com/plans/pro/)
3. Build a tiny provisioning service that can:
- create tunnel,
- fetch token,
- create DNS record,
- register household/device mapping. [developers.cloudflare](https://developers.cloudflare.com/dns/manage-dns-records/how-to/create-dns-records/)
4. Build the mini PC bootstrap agent.
5. Protect `admin` and `support` with Access, but keep family auth inside the app. [developers.cloudflare](https://developers.cloudflare.com/cloudflare-one/team-and-resources/users/seat-management/)
6. Pilot with 5–10 homes before designing district sharding.

Would you like the next step as a concrete implementation blueprint with API resources, database schema, bootstrap protocol, and ops runbooks?