# OpenFriend — Source Mirror

[![Join our Discord](https://img.shields.io/badge/Discord-Join%20the%20community-5865F2?style=for-the-badge&logo=discord&logoColor=white)](https://discord.gg/YRTyXEwVsE)

Come chat, ask questions, share builds, and shape the roadmap.

> ## ⚠️ Unofficial — not affiliated with Microsoft, Mojang, or the Xbox brand
>
> OpenFriend is an **independent, community-built** project. It is **not** developed, endorsed, supported, sponsored, certified, or otherwise officially connected to Microsoft Corporation, Mojang AB, Mojang Studios, or the Xbox brand. "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks of their respective owners. Use on accounts and servers you control. You assume all risk.

> ## 🚧 Current scope: offline-mode servers only
>
> Bridges Friends-List joins to **offline-mode** Minecraft servers. Online-mode bypass is implemented but **not yet verified end-to-end** (waiting for Paper / Spigot to release a build for snapshot 26.2). Use `online-mode=false` on the backend until the bypass is certified.

---

This folder is the open-source mirror of the three OpenFriend components, organized as three independent repositories ready to publish on GitHub.

| Directory | What goes on GitHub |
|---|---|
| `OpenFriendCore/` | Go binary source (the CLI) |
| `OpenFriendPlugin/` | Spigot / Paper bridge plugin + Velocity plugin (Gradle multi-module) |
| `OpenFriendBypass/` | Online-mode auth bypass plugin (Gradle multi-module) |

Each directory builds independently. No project-wide build script is included on purpose.

## Building

### OpenFriendCore (Go)

```
cd OpenFriendCore
make build        # local binary
make dist         # cross-compile macOS / Linux / Windows × amd64/arm64
```

Requires Go 1.24+.

### OpenFriendPlugin (Java)

```
cd OpenFriendPlugin
./gradlew :commons:jar :spigot:jar :velocity:jar \
    -Pmc.version=1.21.4 \
    -Pjava.release=21 \
    -Pspigot.api=1.21.4-R0.1-SNAPSHOT \
    -Pplugin.version=0.1.0
```

For each Minecraft target version, change `mc.version`, `java.release` (8/16/17/21), and `spigot.api`. JDK 21 is the build toolchain.

### OpenFriendBypass (Java)

```
cd OpenFriendBypass
./gradlew :bypass:jar \
    -Pmc.version=1.21.4 \
    -Pjava.release=21 \
    -Pspigot.api=1.21.4-R0.1-SNAPSHOT \
    -Pplugin.version=0.1.0
```

Requires JDK 21. Output jar is in `bypass/build/libs/`.

## Layout reference

```
OpenFriendCore/
├── cmd/openfriend/        CLI entry point
├── internal/              Go packages (auth, api, signaling, bridge, ...)
├── go.mod / go.sum
├── Makefile
└── LICENSE

OpenFriendPlugin/
├── commons/               Shared Java 8 library
├── spigot/                Bridge plugin (Spigot/Paper)
├── velocity/              Velocity plugin
├── gradle/ + gradlew      Gradle wrapper
├── settings.gradle.kts
└── LICENSE

OpenFriendBypass/
├── commons/               (Same shared library as OpenFriendPlugin)
├── bypass/                Bypass plugin
├── gradle/ + gradlew
├── settings.gradle.kts
└── LICENSE
```

## License

MIT. See `LICENSE` in each directory.

"Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks of their respective owners. OpenFriend is not affiliated with, endorsed by, sponsored by, or in any way officially connected to Microsoft Corporation, Mojang AB, or the Xbox brand.
