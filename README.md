# OpenFriendMC — OpenFriend release hub

Bridge Minecraft Java Edition's Friends List (snapshot 26.2+) to any TCP Minecraft server.

This repository (**[zerozshare/OpenFriendMC](https://github.com/zerozshare/OpenFriendMC)**) is the central distribution point. **All release assets — Core binaries, OpenFriend plugin jars, OpenFriendBypass plugin jars, OpenFriend mod jars — are uploaded here under [Releases](https://github.com/zerozshare/OpenFriendMC/releases).** The Go binary's auto-update feature pulls from this repo by name. Source code for each component lives in its own repository (see the table below).

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

## What it does

OpenFriend lets your friends join your Minecraft server through the in-game **Friends List** (introduced in Java Edition snapshot 26.2). Five independently-shippable components, mix and match. Source code lives in separate repositories — pre-built artifacts ship here under [Releases](https://github.com/zerozshare/OpenFriendMC/releases).

| Component | Source repository | What it does |
|---|---|---|
| **OpenFriend Core** (CLI / Go binary) | [zerozshare/OpenFriendCore](https://github.com/zerozshare/OpenFriendCore) | Authenticates with your Microsoft account, broadcasts presence, accepts incoming Friends-List joins, and bridges the WebRTC data channel to a real Minecraft server |
| **OpenFriend Mod** (Fabric client mod) | [zerozshare/OpenFriendMod](https://github.com/zerozshare/OpenFriendMod) | Brings the snapshot 26.2 Friends List UI to **older Minecraft versions** (1.16.5 – 1.21.11, 30 Fabric builds). Embeds Core as a subprocess; Mod-to-Mod joins work on online-mode servers natively. Forge / NeoForge planned for v1.0.1 |
| **OpenFriend Plugin** (Spigot / Paper / Velocity) | [zerozshare/OpenFriendPlugin](https://github.com/zerozshare/OpenFriendPlugin) | Drops the Core binary into your server, starts it as a managed subprocess, surfaces status in chat for OPs |
| **OpenFriend Bypass** (Spigot / Paper) | [zerozshare/OpenFriendBypass](https://github.com/zerozshare/OpenFriendBypass) | Optional. Skips encryption auth on **online-mode** servers for Friends-List-routed connections (Floodgate-style) |
| **OpenMix** (UI toolkit library) | [zerozshare/OpenMix](https://github.com/zerozshare/OpenMix) | Renderer-agnostic Java UI toolkit extracted from the mod. Powers the mod's overlay across all 30 MC versions; reusable in other projects |

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

### Mod (Fabric, 1.16.5 – 1.21.11)

1. Install [Fabric Loader](https://fabricmc.net/use/installer/) for your target Minecraft version.
2. Drop the matching `OpenFriend-fabric-<MCver>.jar` from the [OpenFriendMod release](https://github.com/zerozshare/OpenFriendMod/releases) into `mods/`.
3. Launch. A new **Friends** button appears on Title / Pause / Multiplayer screens.
4. Click it → first run prompts a Microsoft device code (URL + code shown in `latest.log`). After signing in once, your full Friends overlay opens: list / requests / search / host / blocked.

No Fabric API dependency. The Core binary is bundled inside each jar and auto-launches on first use via JSON-RPC stdio. **30 builds** cover MC 1.16.5 → 1.21.11. Forge / NeoForge builds planned for v1.0.1.

## Status

OpenFriend is currently in **test mode**. Feature matrix:

| | Standalone CLI | Plugin (Spigot/Velocity) | Mod (Fabric) |
|---|:---:|:---:|:---:|
| Microsoft authentication | ✓ | ✓ | ✓ |
| Presence broadcasting | ✓ | ✓ | ✓ |
| Friends list management | ✓ | ✓ | ✓ |
| Skin upload | ✓ | ✓ | — |
| Host mode (accept joins) | ✓ | ✓ | ✓ |
| Join mode (join a friend) | ✓ | — | ✓ |
| Offline-mode backend | ✓ | ✓ | ✓ |
| Online-mode backend | ⚠ via Bypass | ⚠ via Bypass | ✓ native (Mod ↔ Mod) |
| Machine-bound credential | ✓ | ✓ | ✓ (via Core) |
| Auto-update (Core) | ✓ | notify-only | bundled |
| Forge / NeoForge build | — | — | ⏳ v1.0.1 |
| Bedrock / Java protocol bridge | ✗ | ✗ | ✗ |

✓ implemented · ⚠ implemented but not yet end-to-end verified · ⏳ planned · ✗ not planned · — not applicable

### Coming soon

- **Forge / NeoForge mod builds** — `common-mc/build.gradle.kts` rework for Architectury multiloader pattern. Tracked for v1.0.1.
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
- Mojang's protocol design — referenced for compatibility (no Mojang code is shipped here)

## License

MIT. See [LICENSE](LICENSE).

"Minecraft", "Xbox", "Xbox Live", and related marks are trademarks of Microsoft Corporation and Mojang AB. OpenFriend is **not affiliated with, endorsed by, or sponsored by** Microsoft Corporation, Mojang AB, or the Xbox brand.
