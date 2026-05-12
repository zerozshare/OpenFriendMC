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
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"jp.zpw.openfriend/internal/api"
	"jp.zpw.openfriend/internal/auth"
	"jp.zpw.openfriend/internal/bridge"
	"jp.zpw.openfriend/internal/presence"
	"jp.zpw.openfriend/internal/signaling"
)

func runJoinMode(session *auth.Session, apiClient *api.Client, joinTarget, listenAddr string, stop <-chan os.Signal, logger *slog.Logger) {
	peer, err := api.ResolvePeer(apiClient, joinTarget)
	if err != nil {
		fmt.Fprintln(os.Stderr, "resolve peer:", err)
		os.Exit(1)
	}
	label := peer.Name
	if label == "" {
		label = peer.PMID.String()
	}
	fmt.Printf("Resolved %s -> pmid %s\n", label, peer.PMID)

	var jm *bridge.JoinManager
	sig := signaling.NewClient(session, func(fromPmid uuid.UUID, payload map[string]any) {
		if jm != nil {
			jm.OnFriendJoin(fromPmid, payload)
		}
	}, logger)
	jm = bridge.NewJoinManager(sig, peer.PMID, logger)

	bc := presence.NewBroadcaster(apiClient, api.StatusOnline, 30*time.Second, nil, logger)

	bc.Start()
	sig.Connect()
	if err := jm.Listen(listenAddr); err != nil {
		fmt.Fprintln(os.Stderr, "listen:", err)
		os.Exit(1)
	}

	fmt.Printf("Open Minecraft and connect to %s to join %s's world.\n", listenAddr, label)

	<-stop
	fmt.Println("Shutting down...")
	jm.Close()
	sig.Close()
	bc.Close()
}
