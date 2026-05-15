/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 */
package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"jp.zpw.openfriend/internal/api"
	"jp.zpw.openfriend/internal/auth"
	"jp.zpw.openfriend/internal/ipc"
	"jp.zpw.openfriend/internal/presence"
)

func runIpcStdio(authFile string, parentStop <-chan os.Signal) int {
	writer := ipc.NewWriter(os.Stdout)
	logger := slog.New(ipc.NewLogHandler(writer, slog.LevelInfo))
	slog.SetDefault(logger)

	dir := filepath.Dir(authFile)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		_ = writer.Notify("log", map[string]any{"level": "ERROR", "msg": "data dir create failed", "attrs": map[string]any{"err": err.Error()}})
	}

	blocklist, _ := ipc.LoadBlocklist(filepath.Join(dir, "blocklist.json"))

	store := &auth.Store{Path: authFile}
	session := auth.NewSession(store, func(p *auth.DeviceCodePrompt) {
		_ = writer.Notify("auth.deviceCode", map[string]any{
			"verificationUri":  p.VerificationURI,
			"userCode":         p.UserCode,
			"message":          p.Message,
			"expiresInSeconds": int(p.ExpiresIn.Seconds()),
		})
	}, logger)

	apiClient := api.NewClient(session, logger)
	apiClient.LoadCacheFromDisk(dir)
	refresher := presence.NewRefresher(apiClient, 20*time.Second, false, logger)

	deps := ipc.Deps{
		Version:   version,
		Session:   session,
		APIClient: apiClient,
		Refresher: refresher,
		AuthPath:  authFile,
		Blocklist: blocklist,
		OnReset: func() error {
			return os.Remove(authFile)
		},
	}

	server := ipc.NewServer(os.Stdin, writer, logger)
	handlers := ipc.NewHandlers(deps, writer)
	handlers.Register(server)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-parentStop:
			server.Quit()
		case <-ctx.Done():
		}
	}()

	if err := server.Run(ctx); err != nil {
		logger.Error("ipc server exited with error", "err", err)
		return 1
	}
	return 0
}
