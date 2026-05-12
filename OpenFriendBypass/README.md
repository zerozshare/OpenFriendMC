> ## ⚠️ Unofficial — not affiliated with Microsoft, Mojang, or the Xbox brand
>
> OpenFriend is an **independent, community-built** project. It is **not** developed, endorsed, supported, sponsored, certified, or otherwise officially connected to Microsoft Corporation, Mojang AB, Mojang Studios, or the Xbox brand. "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks of their respective owners. Use OpenFriend on accounts you control, on servers you operate or have permission to operate on. You assume all risk associated with running this software.

> ## 🚧 Current scope: offline-mode servers only
>
> OpenFriend bridges Friends-List joins **only to offline-mode Minecraft servers** at this time. The online-mode bypass (Floodgate-style auth skip) is **implemented but not yet verified end-to-end** because Paper / Spigot have not released a build matching snapshot 26.2. Set `online-mode=false` on the backend server you bridge to until the bypass is certified.

---

# OpenFriend Bypass

Optional companion to **OpenFriend Plugin** that lets the Friends-List bridge work on **online-mode** Minecraft servers. Skips the encryption handshake for trusted OpenFriend-routed connections by injecting a signed marker in the handshake address (Floodgate-style).

## Requirements

| | |
|---|---|
| Minecraft server | Paper or Spigot, Java 17+ (1.18+) |
| Companion library | [packetevents-spigot](https://github.com/retrooper/packetevents) (drop the jar in `plugins/`) |
| Companion plugin | OpenFriend Plugin (the bridge) |

## Install

Drop into `plugins/`:

1. `OpenFriendBypass-<MCver>.jar` (matching your server's MC version)
2. `packetevents.jar` (required, not bundled)
3. `OpenFriend-spigot-<MCver>.jar` (the bridge plugin — already installed if you set up online-mode bypass)

Start the server:

1. First boot generates `plugins/OpenFriendBypass/bypass.pem` and copies it to `plugins/OpenFriend/bypass.pem`
2. Plugin asks you to RESTART
3. After restart, the Go binary loads the same key and signs handshake markers; the plugin verifies + skips encryption for matching connections

## OP status

```
[OpenFriendBypass] status:
  key: loaded
  reflection: ready
  bypassed connections: 3
```

If `reflection: unavailable` appears, your MC server version uses a class layout the bypass introspection didn't recognize. The plugin falls back gracefully (no harm done, just no bypass).

## Compatibility

OpenFriendBypass uses runtime introspection of `ServerLoginPacketListenerImpl` rather than per-version NMS code, so the same jar works across many MC versions. Per-version jars are still provided for convenience (matching bytecode target).

| MC range | Java target | Verified |
|---|---|---|
| 1.18.x ~ 1.20.4 | 17 | reflection should match |
| 1.20.5 ~ 1.21.x | 21 | reflection verified against snapshot 26.2 layout |
| 26.1.x | 21 | should match |
| Snapshot 26.2+ | 25 | designed for this layout |

## License

MIT. See `LICENSE` in this directory.
