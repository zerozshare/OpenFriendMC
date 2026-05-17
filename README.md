# OpenFriendMC — OpenFriend release hub
<img width="1920" height="1080" alt="openfriend-all-version" src="https://github.com/user-attachments/assets/162c92c3-74a9-48fd-80e2-2f63dbb5f110" />


Bridge Minecraft Java Edition's Friends List (snapshot 26.2+) to any TCP Minecraft server.

This repository (**[zerozshare/OpenFriendMC](https://github.com/zerozshare/OpenFriendMC)**) is the central distribution point. **All release assets — Core binaries, OpenFriend plugin jars, OpenFriendBypass plugin jars, OpenFriend mod jars (Fabric / Forge / Forge-Legacy) — are uploaded here under [Releases](https://github.com/zerozshare/OpenFriendMC/releases).** The Go binary's auto-update feature pulls from this repo by name. Source code for each component lives in its own repository (see the table below).

Official site: **https://openfriend.net/**
Team: **ZSHARE** ([zpw.jp](https://zpw.jp))

[![Join our Discord](https://img.shields.io/badge/Discord-Join%20the%20community-5865F2?style=for-the-badge&logo=discord&logoColor=white)](https://discord.gg/YRTyXEwVsE)

Come chat, ask questions, share builds, and shape the roadmap.

> ## ⚠️ Unofficial — not affiliated with Microsoft, Mojang, or the Xbox brand
>
> OpenFriend is an **independent, community-built** project. It is **not** developed, endorsed, supported, sponsored, certified, or otherwise officially connected to Microsoft Corporation, Mojang AB, Mojang Studios, or the Xbox brand. "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks of their respective owners and are used here for descriptive interoperability purposes only.
>
> OpenFriend speaks the same network protocols the official Minecraft Java Edition client uses. **Use it on accounts you control, on servers you operate or have permission to operate on. You assume all risk associated with running this software.**

> ## 🚧 Current scope: offline-mode servers only
>
> Right now, OpenFriend bridges Friends-List joins **only to offline-mode Minecraft servers**. The online-mode bypass (Floodgate-style auth skip) is **implemented but not yet verified end-to-end** because Paper / Spigot have not released a build matching snapshot 26.2 — once they do, the bypass plugin will be smoke-tested and certified for online-mode use. Until then: set `online-mode=false` on the backend server you bridge to.

## What's new in v1.0.7 (current)

Reliability release focused on the **WebRTC tunnel quality**. If you tried OpenFriend earlier and hit "Connecting…" timeouts or laggy joins on heavy modpacks, this release fixes those:

- **No more silent SCTP drops on heavy login handshakes.** Backpressure now propagates from the WebRTC send queue back into the local TCP socket via TCP flow control — Forge mod-registry sync for 90+ mod packs (Hexxit-Updated, large horror packs, kitchen-sink) no longer corrupts mid-handshake.
- **ICE candidate path visibility.** Every successful join logs whether it went `LAN-direct`, `P2P (NAT-traversed, full bandwidth)`, or `TURN-relay (bandwidth limited by relay)` — so you immediately know if slowness is a relay-bandwidth issue.
- **Explicit STUN server + relaxed ICE timeouts** for higher P2P success rate vs. relay fallback on tricky NATs.
- **First-class 1.8.9 Forge support** via standalone ForgeGradle 2.1.6 (the toolchain the 1.8.9 community originally used). Now ships alongside 1.7.10 and 1.12.2 in the Legacy bundle.
- **Multi-session bug fixed.** Same friend joining twice in quick succession (reconnect, two clients, race conditions) no longer kills the first session.

See per-component release notes: [Core](RELEASE_NOTES_v1.0.7_core.md) · [Fabric](RELEASE_NOTES_v1.0.7_fabric.md) · [Forge](RELEASE_NOTES_v1.0.7_forge.md) · [Plugin](RELEASE_NOTES_v1.0.7_plugin.md)

## What it does

OpenFriend lets your friends join your Minecraft server through the in-game **Friends List** (introduced in Java Edition snapshot 26.2). Six independently-shippable components, mix and match. Source code lives in separate repositories — pre-built artifacts ship here under [Releases](https://github.com/zerozshare/OpenFriendMC/releases).

| Component | Source repository | What it does |
|---|---|---|
| **OpenFriend Core** (CLI / Go binary) | [zerozshare/OpenFriendCore](https://github.com/zerozshare/OpenFriendCore) | Authenticates with your Microsoft account, broadcasts presence, accepts incoming Friends-List joins, and bridges the WebRTC data channel to a real Minecraft server. 5 prebuilt platform binaries (macOS arm64/amd64, Linux arm64/amd64, Windows amd64) |
| **OpenFriend Mod** (Fabric + modern Forge + NeoForge) | [zerozshare/OpenFriendMod](https://github.com/zerozshare/OpenFriendMod) | Brings the snapshot 26.2 Friends List UI to **MC 1.16.5 – 1.21.11** across Fabric (26 builds), Forge (18 builds), and NeoForge (in progress). Built via Architectury Loom + an in-house preprocessor (OpenProcess) that lets one source tree target 30+ MC API generations. Embeds Core as a subprocess |
| **OpenFriend Mod — Legacy** (Forge 1.7.10 / 1.8.9 / 1.12.2) | [zerozshare/OpenFriendModLegacy](https://github.com/zerozshare/OpenFriendModLegacy) | Same Friends UI ported to pre-Loom legacy Forge. 1.7.10 / 1.12.2 use RetroFuturaGradle 1.4.9; **1.8.9** uses standalone ForgeGradle 2.1.6 (the only FG version that supports it). 1.6.4 and older are out of scope — their pre-2014 username-based auth predates Mojang's OAuth, so the Friends List API isn't reachable from those clients |
| **OpenFriend Plugin** (Spigot / Paper / Velocity) | [zerozshare/OpenFriendPlugin](https://github.com/zerozshare/OpenFriendPlugin) | Drops the Core binary into your server, starts it as a managed subprocess, surfaces status in chat for OPs. 73 Spigot jars (1.8 → 1.21.11) + 1 Velocity jar |
| **OpenFriend Bypass** (Spigot / Paper) | [zerozshare/OpenFriendBypass](https://github.com/zerozshare/OpenFriendBypass) | Optional. Skips encryption auth on **online-mode** servers for Friends-List-routed connections (Floodgate-style). 30 builds covering modern MC versions on Java 17+ |
| **OpenMix** (UI toolkit library) | [zerozshare/OpenMix](https://github.com/zerozshare/OpenMix) | Renderer-agnostic Java UI toolkit extracted from the mod. Powers the mod's overlay across all MC versions; reusable in other projects |

## Quick start

### CLI (standalone)

```
./openfriend --target 127.0.0.1:25565
```

First run prints a Microsoft device code. Authenticate once, the token is encrypted to your machine and reused.

### Plugin (Spigot / Paper / Velocity)

1. Drop these into `plugins/`:
   - `OpenFriend-spigot-<MCver>.jar` (bridge plugin, or `OpenFriend-velocity-*.jar` for Velocity)
   - `OpenFriendBypass-<MCver>.jar` (optional, for online-mode servers)
   - `packetevents.jar` (required when using the Bypass plugin)
2. Start the server. The plugin extracts the Core binary, generates `bypass.pem`, and prompts you to restart.
3. Restart. OPs see a status report in chat on login.

### Mod (Fabric / Forge / Forge-Legacy)

1. Install the loader for your target Minecraft version:
   - **Fabric**: https://fabricmc.net/use/installer/  (MC 1.16.5 – 1.21.11)
   - **Forge modern**: install the matching Forge installer  (MC 1.16.5 – 1.21.8)
   - **Forge legacy**: install Forge for 1.7.10 / 1.8.9 / 1.12.2
2. Drop the matching jar into `mods/`:
   - Fabric → `OpenFriend-fabric-<MCver>.jar`
   - Forge modern → `OpenFriend-forge-<MCver>.jar`
   - Forge legacy → `OpenFriend-forge-<MCver>.jar` (same naming, from the legacy bundle)
3. Launch Minecraft. A new **Friends** button appears on Title / Pause / Multiplayer screens.
4. Click it → first run prompts a Microsoft device code (URL + code shown in `latest.log`). After signing in once, your full Friends overlay opens: list / requests / search / host / blocked.

No Fabric API dependency required. The Core binary for every platform is bundled inside each jar (~22 MB jar) and auto-launches on first use via JSON-RPC stdio. **47 mod builds total** (26 Fabric + 18 Forge + 3 Forge-Legacy).

## Status

OpenFriend is in **test mode** (v1.0.7). Feature matrix:

| | Standalone CLI | Plugin (Spigot/Velocity) | Mod (Fabric) | Mod (Forge) | Mod (Forge-Legacy) |
|---|:---:|:---:|:---:|:---:|:---:|
| Microsoft authentication | ✓ | ✓ | ✓ | ✓ | ✓ |
| Presence broadcasting | ✓ | ✓ | ✓ | ✓ | ✓ |
| Friends list management | ✓ | ✓ | ✓ | ✓ | ✓ |
| Skin upload | ✓ | ✓ | — | — | — |
| Host mode (accept joins) | ✓ | ✓ | ✓ | ✓ | ✓ |
| Join mode (join a friend) | ✓ | — | ✓ | ✓ | ✓ |
| Offline-mode backend | ✓ | ✓ | ✓ | ✓ | ✓ |
| Online-mode backend | ⚠ via Bypass | ⚠ via Bypass | ✓ native (Mod ↔ Mod) | ✓ native | ✓ native |
| Machine-bound credential | ✓ | ✓ | ✓ (via Core) | ✓ (via Core) | ✓ (via Core) |
| Auto-update (Core) | ✓ | notify-only | bundled | bundled | bundled |
| In-game HUD toasts | — | — | ✓ | ✓ | ✓ (1.7.10: in-game only) |
| WebRTC backpressure (v1.0.7) | ✓ | ✓ | ✓ | ✓ | ✓ |
| ICE path diagnostics (v1.0.7) | ✓ | ✓ | ✓ | ✓ | ✓ |
| NeoForge build | — | — | — | — | — |
| Bedrock / Java protocol bridge | ✗ | ✗ | ✗ | ✗ | ✗ |

✓ implemented · ⚠ implemented but not yet end-to-end verified · ⏳ planned · ✗ not planned · — not applicable

### Coming soon

- **NeoForge mod builds** — same Architectury setup as Forge, mostly a matter of validating per-version mappings against Mojang's new artifact layout.
- **Snapshot 26.x / 1.21.9-11 Forge builds** — blocked on Forge upstream publishing Gradle artifacts for those MC versions (Fabric builds already shipped).
- **Self-hosted TURN server option** — for users whose NAT keeps falling back to Mojang's relay and hitting bandwidth limits. v1.0.7 added the diagnostic logging that tells you when you're stuck on relay; v1.1 adds the way out.
- **Real player skin heads on legacy MC versions (1.16.5 – 1.19.x)** — currently shows a hashed-color placeholder.
- **Geyser-style protocol translation** so a 1.20.x server can accept a 26.2-snapshot client through OpenFriend.
- **In-game `/openfriend` commands** for OPs.

## Open source dependencies — Thanks

- [pion/webrtc](https://github.com/pion/webrtc) — pure Go WebRTC
- [coder/websocket](https://github.com/coder/websocket) — Go WebSocket client
- [google/uuid](https://github.com/google/uuid) — UUID handling
- [denisbrodbeck/machineid](https://github.com/denisbrodbeck/machineid) — cross-platform machine identification
- [golang.org/x/image](https://pkg.go.dev/golang.org/x/image) — Catmull-Rom image scaling
- [retrooper/packetevents](https://github.com/retrooper/packetevents) — cross-version Bukkit packet manipulation
- [ForgeGradle 2.1.6](https://github.com/MinecraftForge/ForgeGradle) — the only FG version that natively supports MC 1.8.9, used by our Legacy bundle
- [Architectury Loom](https://github.com/architectury/architectury-loom) — multi-loader gradle plugin powering the 18-version Forge matrix
- [RetroFuturaGradle](https://github.com/GTNewHorizons/RetroFuturaGradle) — modern-Gradle support for legacy MC versions (1.7.10 / 1.12.2)
- Mojang's protocol design — referenced for compatibility (no Mojang code is shipped here)

## License

MIT. See [LICENSE](LICENSE).

"Minecraft", "Xbox", "Xbox Live", and related marks are trademarks of Microsoft Corporation and Mojang AB. OpenFriend is **not affiliated with, endorsed by, or sponsored by** Microsoft Corporation, Mojang AB, or the Xbox brand.
