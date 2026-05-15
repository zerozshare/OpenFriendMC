/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package main

import (
	"io"
	"os"
	"syscall"
)

func watchStdinEOF(stop chan<- os.Signal) {
	_, _ = io.Copy(io.Discard, os.Stdin)
	select {
	case stop <- syscall.SIGTERM:
	default:
	}
}
