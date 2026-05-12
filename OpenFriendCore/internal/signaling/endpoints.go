/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package signaling

import "time"

const (
	base           = "https://signaling-afd.franchise.minecraft-services.net"
	configPath     = "/api/v1.0/configuration/java"
	wsPath         = "/ws/v1.0/messaging/connect/java"
	pingInterval   = 50 * time.Second
	initialBackoff = 1 * time.Second
	maxBackoff     = 30 * time.Second
)
