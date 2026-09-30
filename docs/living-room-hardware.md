# Living-room hardware (candidates)

**Status:** research notes for Milestone 4 device hardening. Not a purchase mandate. Specs and AU prices change; re-check the retailer listing before buying.

Wonderfeed’s living-room goal is a **dedicated viewing appliance**: HDMI (or USB-C display) into the TV, browser kiosk on the child surface, not a full desktop chrome experience. A variant keeps a **small onboard screen** for menu and AI face while the TV stays for playback (see HEIGAOLA below). Chromecast / stock smart-TV browsers are weaker for fail-closed kids use; see the presentation notes below.

Companion design for remote HTTPS to homes: [cloudflare.md](cloudflare.md). Device lockdown still sits outside the app ([architecture.md](architecture.md) layer 5).

## What runs on the box

Always-on residents (order of typical RAM cost while watching):

| Resident | Role | Rough RAM (order of magnitude) |
| --- | --- | --- |
| Chromium (or similar) kiosk | Child surface + YouTube embed playback | ~1–2.5GB; can grow over long sessions |
| PostgreSQL | Provider + host control-plane data | ~100–150MB when tuned for a family DB |
| Wonderfeed / YT Zero app processes | Local HTTP UI and control plane | ~50–200MB idle |
| Optional ontology / graph store | Content and policy relationships | hundreds of MB if kept local and in-memory |
| Optional OTel collector | Span export only (not full trace UI) | ~50–150MB |
| Optional remote support agent (for example RustDesk) | Parent/operator rescue only | tens of MB idle; spikes when connected |

**Agentic / LLM pipelines** (curation, evaluation) are mostly **network-bound** when models stay remote: short local CPU bursts for orchestration and parse, not continuous local inference.

**Video decode:** YouTube delivers a compressed stream. The **local browser** still decodes and composites on the appliance. Hardware decode helps keep the CPU cool during long sessions; it is not “done entirely in the cloud.” Prefer a box with competent AV1 / HEVC / VP9 / H.264 decode when choosing between SBCs and mini PCs.

### Suggested split

| On the TV appliance | Elsewhere (parent laptop, cloud, second server) |
| --- | --- |
| Kiosk browser, Postgres, Wonderfeed + provider UI | Full tracing storage / query UIs (long retention) |
| Lean OTel collector that samples and forwards | Heavy admin / bulk curation review UIs |
| Optional lean local graph if the product needs it | Neo4j-style heavy JVM graphs; local LLM inference |
| Autologin + kiosk lockdown | CI, build, and operator tooling |
| Optional remote-support agent (disabled or gated) | Self-hosted RustDesk / hbbs-hbbr relay, or operator console |

**RAM tiers (guidance):** 8GB is minimum viable for kiosk + Postgres + light apps (skip local graph and full tracing). **16GB** is the comfortable multi-service target. 32GB mainly if you later run local models on the same box.

## Flagged form factor: built-in screen + TV out

A mini PC with its **own small panel** plus HDMI/USB-C to the TV can split roles: TV plays the curated feed; the onboard screen shows a calm menu and an **AI face / concierge presence** without stealing the big display. That is a different product shape than “hide a stick behind the TV.”

**Kiosk is non-negotiable** on whatever computes: autologin into a locked browser (or dual locked surfaces), no desktop chrome, no unrestricted YouTube. The screen or robot only changes *where* the face lives, not the lockdown rules.

**Optional embodiment:** instead of (or in addition to) a built-in panel face, the household can attach [Reachy Mini](https://pollen-robotics.com/reachy-mini/) (Pollen Robotics / Hugging Face) as an expressive companion head. That is an **option**, not the MVP path. Published kit pricing (check the site; shipping/duties extra, lead times apply):

| Variant | Approx. list | Brain | Notes |
| --- | --- | --- | --- |
| Reachy Mini | US$499 | On-board Raspberry Pi CM4, Wi‑Fi + USB | Wireless / standalone-ish |
| Reachy Mini Lite | US$399 | Your Mac/PC over USB | Same motion; budget tethered |

Both ship as DIY kits (~2h assembly). For other regions, Pollen points buyers to Seeed Studio. See [Reachy Mini product page](https://pollen-robotics.com/reachy-mini/).

**Price intuition (operator note):** a screened N100 box (for example HEIGAOLA-class ~16GB with panel) can land in a similar *total* spend band to a headless N100 mini PC **plus** Reachy Mini Lite or Mini, once you add AU mini-PC street price, robot kit, shipping, and import. Prefer a live spreadsheet before locking a BOM; the robot buys motion and presence, the screened PC buys a second locked UI without assembly.

### Candidate: HEIGAOLA / HEIGAOLAPC (N100, 7" screen, 16GB)

Leads the shortlist for the “appliance with a face” idea **without** a separate robot: stock **16GB** (comfortable multi-service tier), N100, 7" integrated display (touch noted in listing metadata), HDMI + full-function USB-C for TV/monitor out, dual 2.5GbE. Fan-cooled. Optional 4G data path on the listing (IoT-oriented; not a phone).

| Field | Value (listing-dependent) |
| --- | --- |
| Model | HEIGAOLAPC mini PC with screen (`7" 16GB+256GB` style) |
| CPU | Intel Processor N100 (up to 3.4GHz) |
| Memory | **16GB DDR4** (listing max 16GB; treat as fixed) |
| Storage | 256GB + microSD up to 512GB (listing) |
| On-device display | **7" screen** (touch / human-interface per listing) |
| External display | HDMI + USB-C (PD / 4K claimed); triple/dual display marketing; use TV for playback |
| Network | Dual **2.5GbE**, Wi‑Fi 5, BT 5.0; 4G data module claimed (no calling) |
| Cooling | Active fan (8W TDP class N100) |
| Size | ~6.81 × 4.57 × 1.02 in (listing) |
| AU listing (example) | [Amazon AU: HEIGAOLA N100 7" 16GB 256GB](https://www.amazon.com.au/HEIGAOLA-Windows-Computer-Display-Desktop/dp/B0DB1TKXRW) |

**Why it may be the ideal Wonderfeed appliance**

- Local screen can host menu + AI face while the TV stays a clean viewing surface.
- **16GB** matches the RAM guidance without a SODIMM kit.
- Dual 2.5G helps if the box also runs local services or a soft-router role.
- One SKU vs “headless N100 + Reachy Mini” can win on simplicity at a similar spend band (see price intuition above).

**Caveats**

- Brand/support is thinner than MINIX / Beelink / MeLE; treat as a prototype appliance until field-proven.
- Built-in panel + Windows still needs **kiosk** lockdown; the small screen must not become an escape hatch to the full desktop.
- Dual-UI product work (TV child surface vs local concierge UI) is host software, not free with the hardware.
- Fan noise and 7" placement (coffee table vs shelf vs under-TV) need a living-room trial.
- Related HEIGAOLA SKUs use 5.5" / 10.1" panels and older N5095 chips; stick to the N100 16GB listing if that is the target.
- Reachy remains available later if you want physical motion; the panel does not replace that product option.

## Candidate: MeLE PCG02 (N100 stick)

Almost ideal form factor for “hide behind the TV”: fanless PC stick, HDMI path, quiet 24/7 intent, N100 decode block.

| Field | Value (listing-dependent) |
| --- | --- |
| Model | MeLE PCG02 (`PCG02-N100`) |
| CPU | Intel Processor N100 (4C/4T, up to 3.4GHz) |
| Memory | 8GB LPDDR4 (soldered; not a drop-in 16GB upgrade on this SKU) |
| Storage | 256GB (variant: 128GB); microSD expand up to ~1TB (listing claim) |
| Display | HDMI; 4K support (listing); HDMI extender in box |
| Network | Dual-band Wi‑Fi (2.4/5GHz AC), Gigabit Ethernet, Bluetooth 4.2 |
| Ports | 2× USB-A 10Gbps, 1× USB-C 10Gbps; USB-C PD noted on related MeLE stick SKUs |
| Cooling | Fanless (case can run warm, listing cites ~55–70°C surface) |
| Extras | VESA mount, Wake on LAN / PXE / auto power / RTC wake, Kensington slot |
| OS (typical retail) | Windows 11 Pro (replace or dual-boot Linux for kiosk if preferred) |
| AU listing (example) | [Amazon AU: MeLE PCG02 N100 8GB 256GB](https://www.amazon.com.au/MeLE-PCG02-Computer-Functional-Industrial/dp/B0DK4XX23P) |

**Why it fits**

- Stick + HDMI matches the living-room appliance story better than a full desktop tower.
- Fanless and signage-oriented power features suit an always-on kiosk.
- N100 is a sane class for Chromium + embeds without needing a gaming APU.

**Caveats**

- **8GB RAM** matches “minimum viable,” not the comfortable multi-service tier. Fine for kiosk + Postgres + light Go/Node; defer local graph and heavy observability, or ship spans off-box.
- Soldered RAM: if you later want 16GB residents on-box, pick a different SKU (see alternates), not a memory upgrade on this stick.
- Fanless heat: leave airflow; do not bury under soft furnishings.
- Windows stick out of the box still shows desktop chrome until you install a kiosk autostart (or Linux + Chromium `--kiosk`).
- Retail links and configurations change; treat the Amazon URL as a pointer, not a pin.

## AU shortlist: mini PC boxes

Same living-room job as the stick. Prefer **16GB-class RAM** (stock or after a SO-DIMM kit) if ontology + tracing stay on-box. Several SKUs ship as **8GB**; budget a RAM upgrade where the chassis allows it. Beelink ME and DreamQuest land in the **12GB** band (closer to comfortable, still short of 16GB). MINIX Z100-0dB is the fanless **box** alternative to the MeLE stick.

### Candidate: GMKtec NucBox G3 Pro (i3-10110U)

Box form, VESA-friendly, dual HDMI, stronger single-thread than N100 per the listing’s own comparison language. Active cooling (fan), not silent like the MeLE stick.

| Field | Value (listing-dependent) |
| --- | --- |
| Model | GMKtec NucBox G3 Pro |
| CPU | Intel Core i3-10110U (2C/4T, up to 4.1GHz) |
| Memory | 8GB DDR4 SO-DIMM (listing: up to 64GB) |
| Storage | 256GB M.2; dual M.2 expansion claimed |
| Display | Dual HDMI (listing mixes HDMI 1.4 / 4K notes; verify before buy) |
| Network | Wi‑Fi 6, BT 5.2, **2.5GbE** LAN |
| Cooling | Active fan + heat pipe; ~114×106×44mm |
| Extras | VESA mount, HDMI cable in box |
| AU listing (example) | [Amazon AU: GMKtec G3 Pro i3-10110U 8GB 256GB](https://www.amazon.com.au/GMKtec-i3-10110U-Channel-Computer-Business/dp/B0GGH84CXK) |

**Fits when:** you want expandability and wired 2.5G behind a TV/monitor, and accept fan noise. **Caveat:** still 8GB out of the box; 10th-gen UHD decode is fine for common YouTube codecs, but check AV1 hardware support if that matters for your embeds.

### Candidate: C4 SE (Ryzen 5 3500U)

Vega 8 iGPU is a step up for multi-monitor / light media vs N-series sticks. Dual LAN and triple display (HDMI + DP + USB-C) are office-oriented extras.

| Field | Value (listing-dependent) |
| --- | --- |
| Model | C4 SE mini PC |
| CPU | AMD Ryzen 5 3500U (4C/8T, up to 3.7GHz, ~15W TDP) |
| iGPU | Radeon Vega 8 |
| Memory | 8GB DDR4 SO-DIMM (listing: up to 32GB dual-channel) |
| Storage | 256GB NVMe; dual M.2 2280 PCIe 3.0 claimed |
| Display | HDMI 2.0 4K@60, DP 4K@60, USB-C video (listing: 1080p on Type-C) |
| Network | Dual Gigabit LAN, Wi‑Fi 5, BT 5.0 |
| AU listing (example) | [Amazon AU: C4 SE Ryzen 5 3500U 8GB 256GB](https://www.amazon.com.au/C4-SE-Ryzen-3500U-Windows/dp/B0DHGFS8NC) |

**Fits when:** you want AMD graphics headroom and easy RAM/SSD upgrades. **Caveat:** Wi‑Fi 5 only; fan-cooled box, not a stick.

### Candidate: ORIGIMAGIC C5 (Ryzen 5 3500U)

Same CPU class as C4 SE. Retail URL slug may say “C4 Neo”; the live AU title presents as **C5**. Dual HDMI 4K@60 + USB-C PD/video; under 1kg; VESA-friendly per listing.

| Field | Value (listing-dependent) |
| --- | --- |
| Model | ORIGIMAGIC C5 (listing slug may still say C4 Neo) |
| CPU | AMD Ryzen 5 3500U (4C/8T, up to 3.7GHz) |
| iGPU | Radeon Vega 8 |
| Memory | 8GB DDR4 SODIMM (listing: up to 32GB) |
| Storage | 256GB NVMe; extra M.2 2242 slot claimed |
| Display | Dual HDMI 2.0 4K@60 + USB-C DP / PD 3.0 |
| Network | Wi‑Fi 5, BT 5.0, Gigabit LAN |
| AU listing (example) | [Amazon AU: ORIGIMAGIC C5 / C4 Neo path 8GB 256GB](https://www.amazon.com.au/ORIGIMAGIC-C4-Neo-Windows-Computer/dp/B0DM1T1G15) |

**Fits when:** similar to C4 SE; prefer dual HDMI over HDMI+DP. **Caveat:** confirm the exact chassis/SKU on the page you buy; brand naming across C4/C5/Neo variants is messy on marketplaces.

### Candidate: Beelink ME (N150, 12GB)

Closer to a **NAS / soft-router / HTPC** chassis than a pure HDMI stick: N150 (newer than N100), **12GB LPDDR5** soldered (no SO-DIMM bump), tiny onboard eMMC, and up to **six** M.2 SSD slots. Dual 2.5GbE + Wi‑Fi 6. Stock storage is not appliance-ready until you add at least one NVMe.

| Field | Value (listing-dependent) |
| --- | --- |
| Model | Beelink ME mini |
| CPU | Intel Twin Lake N150 (4C/4T, up to 3.6GHz); listing claims ~10%+ vs N100 |
| Memory | **12GB LPDDR5** (soldered; max 12GB on this SKU) |
| Storage | 64GB eMMC + **6×** M.2 PCIe 3.0 slots (2230/2242/2280 claimed; buy SSDs separately) |
| Display | Dual HDMI (4K claimed) |
| Network | Dual **2.5GbE**, Wi‑Fi 6, BT 5.2 |
| Cooling | Active vertical airflow; listing cites quiet-ish fan under 4K load |
| Power | Built-in PSU (AC cable; fewer brick clutter claims) |
| Extras | WOL / PXE / auto power-on (BIOS); HDMI cable in box |
| AU listing (example) | [Amazon AU: Beelink ME N150 12GB 64GB eMMC](https://www.amazon.com.au/Beelink-ME-Computer-Support-Virtual/dp/B0FJRZTCVB) |

**Fits when:** you want N-series efficiency, more RAM than the 8GB sticks/boxes without shopping a SODIMM kit, and room for local disk (Postgres, downloads, maybe a lean graph on SSD). **Caveats:** 12GB is better than 8GB but still below the comfortable **16GB** multi-service target; RAM is not upgradeable; plan budget and install time for at least one M.2 SSD (64GB eMMC alone is a poor root for Docker + DB + OS); fan + NAS form factor is larger/louder than the MeLE stick; overkill if you only need a kiosk with cloud-backed storage.

### Candidate: MINIX Z100-0dB (N100, fanless box)

Fanless **mini PC** (not a stick): N100, real **256GB NVMe** in the box, dual 4K display, **2.5GbE**, USB-C, VESA, auto power-on. Listing claims no thermal underclocking and up to **32GB** RAM max (verify whether this SKU is user-upgradeable before buying a kit). Sibling **NEO Z300-0dB** (N300, **16GB**/512GB) appears on the same AU storefront if you want stock 16GB and still fanless.

| Field | Value (listing-dependent) |
| --- | --- |
| Model | MINIX NEO Z100-0dB (256GB) |
| CPU | Intel Processor N100 (Alder Lake-N, 4C/4T, up to ~3.4GHz) |
| Memory | 8GB DDR4 (listing max 32GB; confirm upgrade path on the unit you buy) |
| Storage | 256GB M.2 PCIe Gen3 x4 NVMe |
| Display | Dual HDMI, 4K dual display claimed |
| Network | **2.5GbE**, Wi‑Fi (dual-band antennas in box), BT as shipped |
| Cooling | **Fanless** (0dB marketing); passive + RAM cooling claims |
| Extras | VESA mount, auto power-on (BIOS), HDMI cable, multi-region PSU plugs |
| AU listing (example) | [Amazon AU: MINIX Z100-0dB N100 8GB 256GB](https://www.amazon.com.au/MINIX-Z100-0dB-Fanless-256GB-Computer/dp/B0CPLPX78C) |

**Fits when:** you want MeLE-like silence without the stick form factor, plus proper NVMe and 2.5G Ethernet out of the box. **Caveats:** still **8GB** stock (minimum viable); fanless cases run warm (give airflow); confirm RAM upgradeability vs buying the 16GB Z300 sibling instead.

### Candidate: DreamQuest Mini Plus (N95, 12GB / 1TB)

N95 (same Alder Lake-N family as N100; listing claims higher CPU/GPU than N100). Stock **12GB DDR5** (listing max **12GB**, so treat as non-upgradeable) and a large **1TB** M.2 SSD (replaceable up to 2TB per listing). Dual HDMI + dual full-function USB-C for multi-display 4K@60; Wi‑Fi 6 and BT 5.3. A related DreamQuest N95 **16GB**/512GB SKU appears on the same storefront.

| Field | Value (listing-dependent) |
| --- | --- |
| Model | DreamQuest Mini Plus |
| CPU | Intel N95 (up to 3.4GHz) |
| Memory | **12GB DDR5** (listing max 12GB) |
| Storage | **1TB** M.2 2280 SSD (listing: replace up to 2TB) |
| Display | 2× HDMI + 2× USB-C video; triple-screen / 4K@60 claimed |
| Network | LAN, Wi‑Fi 6, Bluetooth 5.3 |
| Cooling | Not marketed as fanless; expect a small fan unless the unit you receive says otherwise |
| Extras | HDMI cable in box; Windows 11 Pro |
| AU listing (example) | [Amazon AU: DreamQuest N95 12GB 1TB](https://www.amazon.com.au/DreamQuest-Windows-Desktop-Computers-Bluetooth5-3/dp/B0FSRG65W3) |

**Fits when:** you want more stock RAM and disk than the 8GB/256GB crowd without opening the chassis for SODIMMs or SSDs on day one. **Caveats:** still under the **16GB** comfort line; RAM ceiling is 12GB; brand/support longevity is less proven than MINIX/Beelink/MeLE; verify fan noise in a quiet living room before committing.

### Short comparison (AU finds)

| Candidate | Form | Stock RAM | Upgrade RAM? | Quiet? | Standout |
| --- | --- | --- | --- | --- | --- |
| **HEIGAOLA 7"** | Mini PC **with screen** | **16GB** | No (max 16GB) | Fan | On-device menu/AI face + TV out; price-competes with N100+Reachy band |
| MeLE PCG02 | Stick | 8GB soldered | No | Fanless | Hides on the HDMI port |
| MINIX Z100-0dB | Mini box | 8GB | Maybe (listing max 32GB) | Fanless | NVMe + 2.5GbE + VESA, silent box |
| GMKtec G3 Pro | Mini box | 8GB | Yes (high ceiling) | Fan | 2.5GbE, Wi‑Fi 6, dual HDMI |
| C4 SE | Mini box | 8GB | Yes (to 32GB) | Fan | Vega 8, dual LAN, HDMI+DP+USB-C |
| ORIGIMAGIC C5 | Mini box | 8GB | Yes (to 32GB) | Fan | Vega 8, dual HDMI + USB-C PD |
| Beelink ME | Mini box (NAS-ish) | **12GB** soldered | No | Fan | N150, 6× M.2, dual 2.5GbE; buy SSDs |
| DreamQuest Mini Plus | Mini box | **12GB** DDR5 | No (max 12GB) | Fan (typical) | N95, **1TB** SSD, dual HDMI + dual USB-C |

## Other candidates (same class)

| Option | Notes |
| --- | --- |
| MeLE Quieter 4C (N100, **16GB** / 512GB) | Better RAM headroom without DIY SODIMM; box form, not stick. Related retail SKUs appear next to PCG02 on AU listings. |
| MINIX NEO Z300-0dB (N300, **16GB** / 512GB) | Fanless sibling of Z100 with stock 16GB; check AU availability/price vs upgrading a Z100. |
| DreamQuest N95 **16GB** / 512GB (related SKU) | Same brand family with stock 16GB if the 12GB/1TB unit is short on RAM. |
| MeLE Quieter 4C N150 16GB | Newer N-series; same “comfortable RAM” idea. |
| Generic N100/N150 mini PC (16GB, NVMe) | Any reputable 16GB N-series mini PC with HDMI; prefer NVMe SSD over eMMC-only for Postgres + logs. |
| Raspberry Pi 5 | Cheaper appliance experiments; weaker / less complete hardware video decode story than N100 for long YouTube sessions. Acceptable for UI-only trials. |
| Android TV box + Fully Kiosk (device owner) | Viable if you control the launcher; stock Google TV sticks often escape via Home. |
| Chromecast / AirPlay / stock smart-TV browser | Not preferred primary path: high escape risk and weak control of Premium session + curated UI. |
| [Reachy Mini](https://pollen-robotics.com/reachy-mini/) (optional) | Expressive companion head; US$399 Lite (tethered to your N100/PC) or US$499 wireless (CM4 on-board). Pair with a **kiosk** mini PC + TV; does not replace device lockdown. |

## Presentation path (software on the box)

Recommended primary path (headless or stick → TV only):

1. Appliance boots to a dedicated non-admin account.
2. Chromium (or Edge) starts in **kiosk** mode on the child-surface URL only.
3. Hide or remove desktop/dock; restrict key chords where the OS allows.
4. Optional nightly kiosk restart to limit Chromium memory growth on long sessions.

Recommended path when the box has an **on-device screen** (for example HEIGAOLA 7"):

1. Same **kiosk** lockdown and fail-closed child policy as above.
2. **TV / HDMI output:** curated playback / child surface only.
3. **Local panel:** menu, session status, and AI face / concierge UI (parent-facing or ambient), not a second open YouTube window.
4. Do not let the small screen expose a full Windows desktop, browser chrome, or unrestricted search.

Optional path with **Reachy Mini**:

1. Keep the mini PC + TV on the kiosk child surface (playback never depends on the robot).
2. Run Reachy as a separate presence channel (Lite USB to the appliance, or wireless Mini with its own CM4).
3. Drive face/motion from Wonderfeed’s concierge/agent APIs later; do not let robot apps open unrestricted YouTube or desktop control for the child profile.
4. Treat robot failure as non-blocking: TV viewing still works if Reachy is offline.

Do not rely on YouTube embed parameters alone for allowlist or “no related video” enforcement. Device lockdown and Wonderfeed policy remain separate layers ([product-brief.md](product-brief.md), [architecture.md](architecture.md)).

## Dual-output kiosk pin (TV vs local panel)

Goal: one physical output (usually **HDMI to the TV**) shows **only** the child-surface Chromium/Edge kiosk; the other output (built-in 7" panel or second monitor) shows menu / AI face. Research snapshot (Perplexity, 2026): [pin kiosk to one HDMI](https://www.perplexity.ai/search/3af07ce4-a2d1-4242-8d47-0bbcad97b275).

**Must use extended desktop, not mirrored.** Duplicate/mirror shares one framebuffer on both screens, so the TV cannot be “kiosk only.”

### Recommended: Linux + Cage (fail-closed)

1. Install a minimal Linux image with **no** full desktop environment (nothing to fall back to if a browser dies).
2. Treat each output as its own sealed surface: run [Cage](https://github.com/cage-kiosk/cage) (or Sway with workspace pinned to an output) **once per display**, each launching Chromium `--kiosk` at a different URL and `--user-data-dir`.
3. Manage both with systemd `Restart=always` so a crash returns to kiosk, not a shell.
4. On HDMI unplug/replug, expect a brief black re-attach, not Explorer; still use a cable retention clip so hotplug is rare.
5. Modern N100 boards generally drive **two independent** outputs (HDMI + panel / USB-C DP-alt). On triple-port boards, confirm the two you need are both active pipes.

### Fallback: Windows 11 Shell Launcher

Stock **Assigned Access** single-app kiosk and Intune multi-app kiosk are a poor fit for “two different apps on two monitors.” Prefer:

1. [Shell Launcher](https://learn.microsoft.com/en-us/windows/configuration/shell-launcher/) (replace Explorer) or a thin custom shell that starts two Edge/Chrome processes.
2. Pin the TV instance with `--kiosk` plus `--window-position` / size matched to that monitor’s coordinates (separate profile per display).
3. Group Policy / lockdown: disable Win+P, Task View, and other ways to switch to Duplicate or show the desktop.
4. Optional: commercial multi-monitor helpers (DisplayFusion-class) only if Explorer is already gone; they are not a substitute for Shell Launcher.
5. Add a watchdog on display-change events to reassert fullscreen placement after HDMI hotplug (Windows is likelier to flash desktop briefly than Cage).

FancyZones and casual “drag the window to the TV” setups are **not** fail-closed for kids.

### Windows dual-monitor smoke test (layout only)

You can prove **placement** on a normal Windows laptop/desktop with two monitors **before** buying appliance hardware or enabling Shell Launcher. This is not a kids lockdown test.

1. Connect two displays. Settings → System → Display → **Extend these displays** (not Duplicate). Click **Identify** and note which is “1” / “2”.
2. Decide which monitor stands in for the TV. Make a note of its origin: if it is the leftmost / primary, `--window-position=0,0` often works; if the other panel is primary, the TV’s X is usually that panel’s width in pixels (for example `1920,0`).
3. Open a terminal and launch two separate browser profiles (Edge or Chrome). Adjust the URL and position values:

```bat
msedge --kiosk https://127.0.0.1:3001 --user-data-dir=%TEMP%\wf-tv-kiosk --window-position=0,0
msedge --app=https://127.0.0.1:8080 --user-data-dir=%TEMP%\wf-panel-ui --window-position=1920,0
```

4. Confirm: TV stand-in shows only the first URL full screen; second monitor shows the other UI; displays are not mirrored.
5. Expect failures that **do not** mean the layout idea is wrong: Alt+Tab, Win+P, taskbar, and dragging windows across monitors still work on stock Windows. Those are closed later with Shell Launcher + policy (or Linux + Cage), not in this smoke test.

Pass criteria for the smoke test: two URLs, two monitors, extended mode, correct placement. Fail criteria to ignore for now: child can still escape via OS chrome.

### Product rule

Playback entitlement and allowlist stay in the **TV kiosk** profile. The local panel (or Reachy) must not open a second unrestricted YouTube session. If the panel stack fails, the TV kiosk must keep running on its own.

## Optional remote support (RustDesk)

Living-room appliances will break (kiosk crash, display mode flipped, disk full). Shipping a remote desktop path lets a parent or Wonderfeed operator fix issues without a truck roll. [RustDesk](https://rustdesk.com/) is the leading candidate: open source, self-hostable relay (`hbbs` / `hbbr`), works on Windows and Linux, light enough for an N100-class box.

**Role:** operator/parent rescue tool. **Not** a child feature and **not** a substitute for kiosk lockdown.

### Placement

| Piece | Where | Notes |
| --- | --- | --- |
| RustDesk client / agent | On the appliance | Starts after boot; may sit behind the kiosk shell |
| ID + one-time / permanent password | Parent vault or operator provisioning | Never printed on the child-facing panel |
| Relay / rendezvous (`hbbs`/`hbbr`) | Operator-controlled VPS or household tunnel path | Prefer self-host over public free servers for family devices |
| Viewer | Parent laptop or support desk | Connect only when troubleshooting |

Aligns with household networking in [cloudflare.md](cloudflare.md): the appliance already reaches out; remote support should not require inbound router port forwards if the relay is reachable outbound.

### Product constraints

1. **Off or gated by default** for child profiles: permanent unattended access only after parent enable, or require approval / time-limited session.
2. **Do not weaken kiosk:** remote session may need a way to exit Shell Launcher / Cage for repair, then return to kiosk; document that as an operator procedure, not a child gesture.
3. **Auth:** strong password or key; rotate after support; no shared “default” password across households.
4. **Scope:** support sees the appliance desktop to fix display/kiosk/OS issues; it must not become a path for the child to browse arbitrary sites through the support viewer’s machine.
5. **Alternatives** if RustDesk is a poor fit later: Tailscale SSH + limited GUI, Meshcentral, commercial RMM. Same constraints apply.

### Open product questions

- Always-installed agent vs install-on-demand when the parent opens a “get help” flow.
- Self-hosted relay per deployment vs one Wonderfeed-operated relay for school/product fleets.
- Whether remote control is allowed while a child session is active, or only when the box is in parent/maintenance mode.

## Open decisions

- Face on a screened appliance (HEIGAOLA-class) vs face on optional [Reachy Mini](https://pollen-robotics.com/reachy-mini/) vs both (panel for menu, Reachy for motion) vs neither (TV-only kiosk).
- Dual-UI OS: **Linux + Cage** (preferred for pin-to-HDMI) vs Windows Shell Launcher (fleet/skill constraint).
- Stick (PCG02) vs fanless box (MINIX Z100 / Z300) vs expandable mini PC (GMKtec / C4 SE / C5) with a planned **16GB** kit vs 12GB fixed boxes (Beelink ME, DreamQuest) vs buying a stock-16GB SKU outright.
- Prefer fanless living-room silence vs slightly louder active cooling for upgradeability or NAS-style disk.
- Whether graph and trace backends ever live on the TV box or only on a parent/ops host.
- Whether the local panel / Reachy is child-visible, parent-only, or ambient (face without controls).
- Reachy Mini vs Mini Lite (on-board CM4 vs tether to the Wonderfeed box).
- Optional RustDesk (or equivalent) for remote rescue: always-on gated agent vs on-demand; self-hosted relay ownership.

## Related

- Windows appliance CD (two-repo): [windows-appliance.md](windows-appliance.md) and private **wonderfeed-local** (`docs/kiosk.md` there)
- Roadmap Milestone 4: [roadmap.md](roadmap.md)
- Household box / tunnel shape: [cloudflare.md](cloudflare.md)
- Provider local ops: [operator-ytzero.md](operator-ytzero.md)
- Reachy Mini (optional embodiment): [pollen-robotics.com/reachy-mini](https://pollen-robotics.com/reachy-mini/)
- Dual-output kiosk research: [Perplexity thread](https://www.perplexity.ai/search/3af07ce4-a2d1-4242-8d47-0bbcad97b275)
- RustDesk (optional remote support): [rustdesk.com](https://rustdesk.com/)
