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

type Observer interface {
	OnFriendAdded(d api.FriendDto)
	OnFriendRemoved(id uuid.UUID)
	OnIncomingRequest(d api.FriendDto)
	OnIncomingResolved(id uuid.UUID)
	OnOutgoingRequest(d api.FriendDto)
	OnOutgoingResolved(id uuid.UUID)
	OnInitialSnapshot(resp *api.FriendsListResponse)
}

type Refresher struct {
	client      *api.Client
	interval    time.Duration
	autoAccept  bool
	logger      *slog.Logger
	observer    Observer

	mu             sync.Mutex
	primed         bool
	knownFriends   map[uuid.UUID]api.FriendDto
	knownIncoming  map[uuid.UUID]api.FriendDto
	knownOutgoing  map[uuid.UUID]api.FriendDto

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
		knownFriends:  map[uuid.UUID]api.FriendDto{},
		knownIncoming: map[uuid.UUID]api.FriendDto{},
		knownOutgoing: map[uuid.UUID]api.FriendDto{},
	}
}

func (r *Refresher) SetObserver(o Observer) {
	r.mu.Lock()
	r.observer = o
	r.mu.Unlock()
}

func (r *Refresher) SetAutoAccept(v bool) {
	r.mu.Lock()
	r.autoAccept = v
	r.mu.Unlock()
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

func (r *Refresher) TickNow() {
	r.tick()
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

func idMap(list []api.FriendDto) map[uuid.UUID]api.FriendDto {
	m := make(map[uuid.UUID]api.FriendDto, len(list))
	for _, d := range list {
		m[d.ProfileID.UUID()] = d
	}
	return m
}

func (r *Refresher) applyDiff(resp *api.FriendsListResponse) {
	r.mu.Lock()
	obs := r.observer
	primed := r.primed
	autoAccept := r.autoAccept
	prevIncoming := r.knownIncoming
	prevFriends := r.knownFriends
	prevOutgoing := r.knownOutgoing
	r.mu.Unlock()

	curF := idMap(resp.Friends)
	curIn := idMap(resp.IncomingRequests)
	curOut := idMap(resp.OutgoingRequests)

	if primed {
		r.diffWithObserver(resp.Friends, prevFriends, curF, "friend", obs,
			func(d api.FriendDto) { if obs != nil { obs.OnFriendAdded(d) } },
			func(id uuid.UUID) { if obs != nil { obs.OnFriendRemoved(id) } })
		r.diffWithObserver(resp.IncomingRequests, prevIncoming, curIn, "incoming-request", obs,
			func(d api.FriendDto) { if obs != nil { obs.OnIncomingRequest(d) } },
			func(id uuid.UUID) { if obs != nil { obs.OnIncomingResolved(id) } })
		r.diffWithObserver(resp.OutgoingRequests, prevOutgoing, curOut, "outgoing-request", obs,
			func(d api.FriendDto) { if obs != nil { obs.OnOutgoingRequest(d) } },
			func(id uuid.UUID) { if obs != nil { obs.OnOutgoingResolved(id) } })
	} else {
		r.logger.Info("Initial friends snapshot",
			"friends", len(curF),
			"incoming", len(curIn),
			"outgoing", len(curOut))
		if obs != nil {
			obs.OnInitialSnapshot(resp)
		}
	}

	if autoAccept {
		for _, d := range resp.IncomingRequests {
			id := d.ProfileID.UUID()
			if _, seen := prevIncoming[id]; !seen {
				r.autoAcceptOne(d)
			}
		}
	}

	r.mu.Lock()
	r.knownFriends = curF
	r.knownIncoming = curIn
	r.knownOutgoing = curOut
	r.primed = true
	r.mu.Unlock()
}

func (r *Refresher) autoAcceptOne(d api.FriendDto) {
	r.logger.Info("Auto-accepting friend request", "name", d.Name, "id", d.ProfileID.UUID())
	if _, err := r.client.PutFriendAction(api.AddByID(d.ProfileID.UUID())); err != nil {
		r.logger.Warn("Failed to accept friend request", "name", d.Name, "err", err)
	}
}

func (r *Refresher) diffWithObserver(
	current []api.FriendDto,
	known map[uuid.UUID]api.FriendDto,
	currentIDs map[uuid.UUID]api.FriendDto,
	label string,
	_ Observer,
	onAdd func(api.FriendDto),
	onRemove func(uuid.UUID),
) {
	for _, d := range current {
		if _, ok := known[d.ProfileID.UUID()]; !ok {
			r.logger.Info("[+]", "kind", label, "name", d.Name, "id", d.ProfileID.UUID())
			onAdd(d)
		}
	}
	for gone := range known {
		if _, ok := currentIDs[gone]; !ok {
			r.logger.Info("[-]", "kind", label, "id", gone)
			onRemove(gone)
		}
	}
}

func (r *Refresher) Close() {
	if r.stop != nil {
		close(r.stop)
		<-r.done
	}
}
