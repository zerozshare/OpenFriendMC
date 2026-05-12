/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package presence

import (
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"jp.zpw.openfriend/internal/api"
)

type Broadcaster struct {
	client      *api.Client
	status      api.PresenceStatus
	interval    time.Duration
	invitePmids []uuid.UUID
	logger      *slog.Logger

	mu      sync.Mutex
	running bool
	stop    chan struct{}
	done    chan struct{}
}

func NewBroadcaster(client *api.Client, status api.PresenceStatus, interval time.Duration, invitePmids []uuid.UUID, logger *slog.Logger) *Broadcaster {
	if logger == nil {
		logger = slog.Default()
	}
	return &Broadcaster{
		client:      client,
		status:      status,
		interval:    interval,
		invitePmids: invitePmids,
		logger:      logger,
	}
}

func (b *Broadcaster) Start() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.running {
		return
	}
	b.running = true
	b.stop = make(chan struct{})
	b.done = make(chan struct{})
	go b.loop()
	b.logger.Info("Presence broadcaster started", "status", b.status, "interval_s", b.interval.Seconds())
}

func (b *Broadcaster) loop() {
	defer close(b.done)
	b.tick()
	t := time.NewTicker(b.interval)
	defer t.Stop()
	for {
		select {
		case <-b.stop:
			return
		case <-t.C:
			b.tick()
		}
	}
}

func (b *Broadcaster) tick() {
	if b.client.PresenceInCooldown() {
		b.logger.Debug("Presence tick skipped (cooldown)", "remaining_s", b.client.PresenceCooldownRemaining().Seconds())
		return
	}
	body := api.PresenceRequest{Status: b.status}
	if b.status == api.StatusPlayingHostedServer {
		body.JoinInfo = api.NewJoinInfoUpdate(b.invitePmids)
	}
	resp, err := b.client.PostPresence(body)
	if err != nil {
		b.logger.Warn("Presence POST failed", "err", err)
		return
	}
	n := 0
	if resp != nil {
		n = len(resp.Presence)
	}
	b.logger.Debug("Presence ack", "peers", n, "status", b.status)
}

func (b *Broadcaster) AnnounceOffline() {
	body := api.PresenceRequest{Status: api.StatusOffline, JoinInfo: nil}
	if _, err := b.client.PostPresence(body); err != nil {
		b.logger.Warn("Failed to announce OFFLINE", "err", err)
		return
	}
	b.logger.Info("Announced OFFLINE presence")
}

func (b *Broadcaster) Close() {
	b.mu.Lock()
	if !b.running {
		b.mu.Unlock()
		return
	}
	b.running = false
	close(b.stop)
	b.mu.Unlock()
	<-b.done
	b.AnnounceOffline()
}
