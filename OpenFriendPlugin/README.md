> ## ⚠️ Unofficial — not affiliated with Microsoft, Mojang, or the Xbox brand
>
> OpenFriend is an **independent, community-built** project. It is **not** developed, endorsed, supported, sponsored, certified, or otherwise officially connected to Microsoft Corporation, Mojang AB, Mojang Studios, or the Xbox brand. "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks of their respective owners. Use OpenFriend on accounts you control, on servers you operate or have permission to operate on. You assume all risk associated with running this software.

> ## 🚧 Current scope: offline-mode servers only
>
> OpenFriend bridges Friends-List joins **only to offline-mode Minecraft servers** at this time. The online-mode bypass (Floodgate-style auth skip) is **implemented but not yet verified end-to-end** because Paper / Spigot have not released a build matching snapshot 26.2. Set `online-mode=false` on the backend server you bridge to until the bypass is certified.

---

# OpenFriend Plugin

Bridge plugin for Paper / Spigot / Velocity servers. Drops the OpenFriend Core binary into the server, manages it as a subprocess, surfaces status to OPs in chat.

## Install

Pick the jar matching your Minecraft version and drop it into `plugins/`:

| Server type | File |
|---|---|
| Spigot / Paper | `OpenFriend-spigot-<MCver>.jar` |
| Velocity | `OpenFriend-velocity-0.1.0.jar` |

Start the server. On first run the plugin:

1. Extracts the matching `openfriend-<os>-<arch>` binary to `plugins/OpenFriend/bin/`
2. Generates `plugins/OpenFriend/auth.pem` after the operator completes a Microsoft device-code login (one-time)
3. Begins broadcasting presence and accepting Friends-List joins

## Configuration

`plugins/OpenFriend/config.yml`:

```yaml
enabled: true
target: 127.0.0.1:25565        # backend Minecraft server (use the server's own port)
interval-s: 30                 # presence interval
no-auto-accept: false          # auto-accept incoming friend requests
verbose: false
skin:
  file: ""                     # path relative to plugins/OpenFriend/
  variant: classic             # classic or slim
```

## OP status

OPs see a status report on join:

```
[OpenFriend] status:
  account: HIKA2021 (28dc6a27-...)
  presence: PLAYING_HOSTED_SERVER ✓
  signaling: connected
  bypass: enabled
```

## Online-mode support

This plugin alone works with **offline-mode** servers. To accept online-mode joins via Friends List, also install the **OpenFriendBypass** plugin.

## License

MIT. See `LICENSE` in this directory.
