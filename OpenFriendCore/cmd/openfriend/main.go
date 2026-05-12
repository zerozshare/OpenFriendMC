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
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/uuid"
	"jp.zpw.openfriend/internal/api"
	"jp.zpw.openfriend/internal/auth"
	"jp.zpw.openfriend/internal/bridge"
	"jp.zpw.openfriend/internal/bypass"
	"jp.zpw.openfriend/internal/presence"
	"jp.zpw.openfriend/internal/signaling"
	"jp.zpw.openfriend/internal/status"
)

var version = "dev"

func main() {
	var (
		dataDir         string
		authFile        string
		target          string
		intervalSec     int
		noAutoAccept    bool
		announceOffline bool
		verbose         bool
		showVersion     bool
		skinPath        string
		skinVariant     string
		noUpdate        bool
		joinTarget      string
		listenAddr      string
		reset           bool
		watchParent     bool
		bypassKeyPath   string
	)
	flag.StringVar(&bypassKeyPath, "bypass-key", "", "path to bypass.pem (enables online-mode auth bypass)")
	flag.BoolVar(&reset, "reset", false, "send OFFLINE, delete saved auth and skin state, then exit")
	flag.BoolVar(&watchParent, "watch-parent", false, "exit gracefully when stdin closes (used when launched as a managed subprocess)")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.StringVar(&skinPath, "skin", "", "PNG file to upload as Minecraft skin (sets the friend-list head icon)")
	flag.StringVar(&skinVariant, "skin-variant", "classic", "skin variant: classic or slim")
	flag.BoolVar(&noUpdate, "no-update", false, "skip the self-update check on startup")
	flag.StringVar(&dataDir, "data-dir", "", "directory for auth.pem/skin.lock (default: next to binary, or ~/.openfriend/)")
	flag.StringVar(&authFile, "auth-file", "", "explicit path to auth.pem (overrides --data-dir)")
	flag.StringVar(&target, "target", "127.0.0.1:25565", "host mode: host:port of backend Minecraft server")
	flag.IntVar(&intervalSec, "interval-s", 30, "presence broadcast interval in seconds")
	flag.BoolVar(&noAutoAccept, "no-auto-accept", false, "don't auto-accept incoming friend requests")
	flag.BoolVar(&announceOffline, "announce-offline", false, "send OFFLINE presence and exit")
	flag.BoolVar(&verbose, "verbose", false, "verbose (debug) logging")
	flag.StringVar(&joinTarget, "join", "", "join mode: friend name or pmid to join")
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:25565", "join mode: local TCP address for Minecraft client to connect to")
	flag.Parse()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	if watchParent {
		go watchStdinEOF(stop)
	}

	if showVersion {
		fmt.Println("openfriend", version)
		return
	}

	if !noUpdate {
		maybeSelfUpdate(verbose)
	}

	lvl := slog.LevelInfo
	if verbose {
		lvl = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lvl}))
	slog.SetDefault(logger)

	if authFile == "" {
		if dataDir == "" {
			dataDir = resolveDataDir()
		}
		authFile = filepath.Join(dataDir, "auth.pem")
	}

	if reset {
		doReset(authFile, logger)
		return
	}

	if _, err := bridge.ParseTarget(target); err != nil {
		fmt.Fprintf(os.Stderr, "invalid --target %q: %v\n", target, err)
		os.Exit(2)
	}

	store := &auth.Store{Path: authFile}
	session := auth.NewSession(store, func(p *auth.DeviceCodePrompt) {
		fmt.Println()
		fmt.Println("==================================================")
		fmt.Println(" 1. Open: " + p.VerificationURI)
		fmt.Println(" 2. Enter the code: " + p.UserCode)
		fmt.Println("==================================================")
		fmt.Println()
	}, logger)

	tokens, err := session.Authenticate()
	if err != nil {
		fmt.Fprintln(os.Stderr, "auth failed:", err)
		os.Exit(1)
	}
	fmt.Printf("Authenticated as: %s (%s)\n", tokens.Name, tokens.ProfileID)

	statusWriter := status.NewWriter(filepath.Dir(authFile))
	statusWriter.Update(func(s *status.Snapshot) {
		s.Authenticated = true
		s.ProfileID = tokens.ProfileID.String()
		s.ProfileName = tokens.Name
		s.Version = version
	})

	apiClient := api.NewClient(session, logger)

	if skinPath != "" {
		lock := filepath.Join(filepath.Dir(authFile), "skin.lock")
		if err := ensureSkin(apiClient, skinPath, skinVariant, lock, logger); err != nil {
			fmt.Fprintln(os.Stderr, "skin upload failed:", err)
			os.Exit(1)
		}
	}

	if announceOffline {
		bc := presence.NewBroadcaster(apiClient, api.StatusOffline, time.Minute, nil, logger)
		bc.AnnounceOffline()
		fmt.Println("Done.")
		return
	}

	if joinTarget != "" {
		runJoinMode(session, apiClient, joinTarget, listenAddr, stop, logger)
		return
	}

	fmt.Println("Bridge target:", target)

	if bypassKeyPath == "" {
		bypassKeyPath = filepath.Join(filepath.Dir(authFile), "bypass.pem")
	}
	var bypassBytes []byte
	if key, err := bypass.LoadOrAbsent(bypassKeyPath); err != nil {
		logger.Warn("Failed to load bypass key; proceeding without bypass", "path", bypassKeyPath, "err", err)
	} else if key != nil {
		bypassBytes = key.Bytes
		logger.Info("Loaded bypass key; online-mode bypass enabled", "path", bypassKeyPath)
	}
	statusWriter.Update(func(s *status.Snapshot) {
		s.BypassEnabled = bypassBytes != nil
	})

	var pm *bridge.HostManager
	sig := signaling.NewClient(session, func(fromPmid uuid.UUID, payload map[string]any) {
		if pm != nil {
			pm.OnFriendJoin(fromPmid, payload)
		}
	}, logger)
	pm = bridge.NewHostManager(sig, target, bypassBytes, logger)

	bc := presence.NewBroadcaster(apiClient, api.StatusPlayingHostedServer,
		time.Duration(intervalSec)*time.Second, nil, logger)
	rf := presence.NewRefresher(apiClient, 20*time.Second, !noAutoAccept, logger)

	bc.Start()
	rf.Start()
	sig.Connect()
	statusWriter.Update(func(s *status.Snapshot) {
		s.PresenceStatus = "PLAYING_HOSTED_SERVER"
		s.PresenceRunning = true
		s.SignalingConnected = true
	})

	fmt.Printf("Broadcasting presence and accepting joins -> %s\n", target)

	<-stop
	fmt.Println("Shutting down...")
	pm.Close()
	sig.Close()
	rf.Close()
	bc.Close()
}
