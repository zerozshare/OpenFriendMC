# OpenFriend

Bridge Minecraft Java Edition's Friends List (snapshot 26.2+) to any TCP Minecraft server, online-mode included.

Official site: **https://openfriend.net/**
Team: **ZSHARE** ([zpw.jp](https://zpw.jp))

## What it does

OpenFriend lets your friends join your Minecraft server through the in-game **Friends List** (introduced in Java Edition snapshot 26.2). Three components, mix and match:

| Component | What it does |
|---|---|
| **OpenFriend Core** (CLI / Go binary) | Authenticates with your Microsoft account, broadcasts presence, accepts incoming Friends-List joins, and bridges the WebRTC data channel to a real Minecraft server |
| **OpenFriend Plugin** (Spigot / Paper / Velocity) | Drops the Core binary into your server, starts it as a managed subprocess, surfaces status in chat for OPs |
| **OpenFriend Bypass** (Spigot / Paper) | Optional. Skips encryption auth on **online-mode** servers for Friends-List-routed connections (Floodgate-style) |

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

## Status

OpenFriend is currently in **test mode**. Feature matrix:

| | Standalone CLI | Plugin (Spigot/Velocity) | Mod (Forge/Fabric/NeoForge) |
|---|:---:|:---:|:---:|
| Microsoft authentication | ✓ | ✓ | — |
| Presence broadcasting | ✓ | ✓ | — |
| Friends list management | ✓ | ✓ | — |
| Skin upload | ✓ | ✓ | — |
| Host mode (accept joins) | ✓ | ✓ | — |
| Join mode (join a friend) | ✓ | — | — |
| Offline-mode backend | ✓ | ✓ | — |
| Online-mode backend (Bypass) | ✓ | ✓ | — |
| Machine-bound credential | ✓ | ✓ | — |
| Auto-update (Core) | ✓ | notify-only | — |
| Bedrock / Java protocol bridge | ✗ | ✗ | — |

✓ implemented · ✗ not planned · — not yet built

### Coming soon

- **Mod** (Forge / Fabric / NeoForge) — bring the same bridging into modded servers and **older Minecraft versions** that don't have the Friends List built in. The idea is to make older Minecraft join newer friends-aware servers (via protocol translation with ViaVersion/ViaBackwards).
- **Geyser-style protocol translation** so a 1.20.x server can accept a 26.2-snapshot client through OpenFriend.
- **In-game `/openfriend` commands** for OPs.
- **Web dashboard** for managing your accounts and server bindings from a browser.

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
