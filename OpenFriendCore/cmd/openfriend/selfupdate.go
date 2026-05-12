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
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"jp.zpw.openfriend/internal/update"
)

func maybeSelfUpdate(verbose bool) {
	lvl := slog.LevelInfo
	if verbose {
		lvl = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))

	checker := &update.Checker{
		Current: version,
		Logger:  logger,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := checker.Check(ctx)
	if err != nil {
		logger.Warn("Update check failed; continuing", "err", err)
		return
	}
	if !out.Available {
		return
	}

	applyCtx, applyCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer applyCancel()
	newPath, err := checker.Apply(applyCtx, out.Latest)
	if err != nil {
		logger.Warn("Self-update apply failed; continuing", "err", err)
		return
	}
	fmt.Fprintf(os.Stderr, "Updated to openfriend %s; re-executing...\n", out.Latest)
	args := append([]string{newPath, "--no-update"}, os.Args[1:]...)
	if err := update.ReExec(newPath, args, os.Environ()); err != nil {
		logger.Warn("Re-exec failed", "err", err)
	}
}
