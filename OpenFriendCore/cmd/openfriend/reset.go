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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"jp.zpw.openfriend/internal/api"
	"jp.zpw.openfriend/internal/auth"
	"jp.zpw.openfriend/internal/presence"
)

func doReset(authFile string, logger *slog.Logger) {
	dir := filepath.Dir(authFile)
	skinLock := filepath.Join(dir, "skin.lock")

	store := &auth.Store{Path: authFile}
	tokens, err := store.Load()
	if err != nil {
		logger.Warn("Failed to load auth file", "err", err)
	}
	if tokens != nil && tokens.MsRefreshToken != "" {
		announceOfflineBeforeReset(store, logger)
	}

	removeIfExists(authFile, "auth.pem", logger)
	legacyJSON := strings.TrimSuffix(authFile, ".pem") + ".json"
	if legacyJSON != authFile {
		removeIfExists(legacyJSON, "auth.json (legacy)", logger)
	}
	removeIfExists(skinLock, "skin.lock", logger)
	fmt.Println("Reset.")
}

func announceOfflineBeforeReset(store *auth.Store, logger *slog.Logger) {
	session := auth.NewSession(store, func(*auth.DeviceCodePrompt) {}, logger)
	if _, err := session.Authenticate(); err != nil {
		logger.Debug("Skipping OFFLINE announce (auth failed)", "err", err)
		return
	}
	client := api.NewClient(session, logger)
	bc := presence.NewBroadcaster(client, api.StatusOffline, time.Minute, nil, logger)
	bc.AnnounceOffline()
}

func removeIfExists(path, label string, logger *slog.Logger) {
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		logger.Warn("Remove failed", "file", label, "path", path, "err", err)
		return
	}
	logger.Info("Removed", "file", label, "path", path)
}
