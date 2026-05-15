/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
//go:build windows

package update

import (
	"fmt"
	"os"
	"os/exec"
)

func replaceSelf(currentPath, newPath string) error {
	oldBackup := currentPath + ".old"
	_ = os.Remove(oldBackup)
	if err := os.Rename(currentPath, oldBackup); err != nil {
		return fmt.Errorf("rename old: %w", err)
	}
	if err := os.Rename(newPath, currentPath); err != nil {
		_ = os.Rename(oldBackup, currentPath)
		return fmt.Errorf("rename new: %w", err)
	}
	return nil
}

func ReExec(path string, args []string, env []string) error {
	cmd := exec.Command(path, args[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
