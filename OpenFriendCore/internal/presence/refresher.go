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

type Refresher struct {
	client      *api.Client
	interval    time.Duration
	autoAccept  bool
	logger      *slog.Logger

	mu             sync.Mutex
	primed         bool
	knownFriends   map[uuid.UUID]struct{}
	knownIncoming  map[uuid.UUID]struct{}
	knownOutgoing  map[uuid.UUID]struct{}

	stop chan struct{}
	done chan struct{}
}

func NewRefresher(client *api.Client, interval time.Duration, autoAccept bool, logger *slog.Logger) *Refresher {
	if logger == nil {
		logger = slog.Default()
	}
	return &Refresher{
		client:        client,
		interval:      interval,
		autoAccept:    autoAccept,
		logger:        logger,
		knownFriends:  map[uuid.UUID]struct{}{},
		knownIncoming: map[uuid.UUID]struct{}{},
		knownOutgoing: map[uuid.UUID]struct{}{},
	}
}

func (r *Refresher) Start() {
	r.stop = make(chan struct{})
	r.done = make(chan struct{})
	go r.loop()
	r.logger.Info("Friends refresher started", "interval_s", r.interval.Seconds(), "auto_accept", r.autoAccept)
}

func (r *Refresher) loop() {
	defer close(r.done)
	r.tick()
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		select {
		case <-r.stop:
			return
		case <-t.C:
			r.tick()
		}
	}
}

func (r *Refresher) tick() {
	if r.client.FriendsInCooldown() {
		r.logger.Debug("Friends tick skipped (cooldown)")
		return
	}
	resp, err := r.client.GetFriends()
	if err != nil {
		r.logger.Warn("GET /friends failed", "err", err)
		return
	}
	if resp == nil {
		return
	}
	r.applyDiff(resp)
}

func idSet(list []api.FriendDto) map[uuid.UUID]struct{} {
	m := make(map[uuid.UUID]struct{}, len(list))
	for _, d := range list {
		m[d.ProfileID.UUID()] = struct{}{}
	}
	return m
}

func (r *Refresher) applyDiff(resp *api.FriendsListResponse) {
	r.mu.Lock()
	defer r.mu.Unlock()

	curF := idSet(resp.Friends)
	curIn := idSet(resp.IncomingRequests)
	curOut := idSet(resp.OutgoingRequests)

	if r.primed {
		r.diffOnce(resp.Friends, r.knownFriends, curF, "friend")
		r.diffOnce(resp.IncomingRequests, r.knownIncoming, curIn, "incoming-request")
		r.diffOnce(resp.OutgoingRequests, r.knownOutgoing, curOut, "outgoing-request")
	} else {
		r.logger.Info("Initial friends snapshot",
			"friends", len(curF),
			"incoming", len(curIn),
			"outgoing", len(curOut))
		r.primed = true
	}

	if r.autoAccept {
		for _, d := range resp.IncomingRequests {
			id := d.ProfileID.UUID()
			if _, seen := r.knownIncoming[id]; !seen {
				r.autoAcceptOne(d)
			}
		}
	}

	r.knownFriends = curF
	r.knownIncoming = curIn
	r.knownOutgoing = curOut
}

func (r *Refresher) autoAcceptOne(d api.FriendDto) {
	r.logger.Info("Auto-accepting friend request", "name", d.Name, "id", d.ProfileID.UUID())
	if _, err := r.client.PutFriendAction(api.AddByID(d.ProfileID.UUID())); err != nil {
		r.logger.Warn("Failed to accept friend request", "name", d.Name, "err", err)
	}
}

func (r *Refresher) diffOnce(current []api.FriendDto, known map[uuid.UUID]struct{}, currentIDs map[uuid.UUID]struct{}, label string) {
	for _, d := range current {
		if _, ok := known[d.ProfileID.UUID()]; !ok {
			r.logger.Info("[+]", "kind", label, "name", d.Name, "id", d.ProfileID.UUID())
		}
	}
	for gone := range known {
		if _, ok := currentIDs[gone]; !ok {
			r.logger.Info("[-]", "kind", label, "id", gone)
		}
	}
}

func (r *Refresher) Close() {
	if r.stop != nil {
		close(r.stop)
		<-r.done
	}
}
