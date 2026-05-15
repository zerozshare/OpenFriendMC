/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
//go:build !windows

package update

import (
	"fmt"
	"os"
	"syscall"
)

func replaceSelf(currentPath, newPath string) error {
	if err := os.Rename(newPath, currentPath); err != nil {
		return fmt.Errorf("rename %s -> %s: %w", newPath, currentPath, err)
	}
	return nil
}

func ReExec(path string, args []string, env []string) error {
	return syscall.Exec(path, args, env)
}
