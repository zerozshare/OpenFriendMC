/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 */
package ipc

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"jp.zpw.openfriend/internal/api"
)

type watchState struct {
	mu sync.Mutex

	friendsRunning bool
	friendsStop    chan struct{}
	friendsDone    chan struct{}

	presenceRunning bool
	presenceStop    chan struct{}
	presenceDone    chan struct{}

	currentStatus  api.PresenceStatus
	currentJoinVal *string
	currentInvites []uuid.UUID

	lastPostedStatus   api.PresenceStatus
	lastPostedJoinVal  *string
	lastPostedInvites  []uuid.UUID
	lastPostedAt       time.Time

	lastPresence map[uuid.UUID]string
}

const (
	presenceMinInterval = 10 * time.Second
	presenceMaxInterval = 60 * time.Second
)

func newWatchState() *watchState {
	return &watchState{
		currentStatus: api.StatusOnline,
		lastPresence:  map[uuid.UUID]string{},
	}
}

func (h *Handlers) registerWatchMethods(s *Server) {
	s.Register("friends.search", h.friendsSearch)
	s.Register("friends.watch", h.friendsWatch)
	s.Register("friends.unwatch", h.friendsUnwatch)
	s.Register("friends.setAutoAccept", h.friendsSetAutoAccept)
	s.Register("presence.set", h.presenceSet)
	s.Register("presence.watch", h.presenceWatch)
	s.Register("presence.unwatch", h.presenceUnwatch)
}

func (h *Handlers) friendsSearch(_ context.Context, raw json.RawMessage) (any, *RPCError) {
	if err := h.requireAuth(); err != nil {
		return nil, err
	}
	p, rerr := parseIDParams(raw)
	if rerr != nil {
		return nil, rerr
	}
	if p.Name == "" {
		return nil, newError(CodeInvalidParams, "name required")
	}
	resp, err := h.deps.APIClient.GetFriends()
	if err != nil {
		return nil, newErrorf(CodeAPIError, "GET /friends failed", err.Error())
	}
	if resp == nil {
		return map[string]any{"found": false}, nil
	}
	for _, d := range resp.Friends {
		if strings.EqualFold(d.Name, p.Name) {
			return map[string]any{
				"found":     true,
				"profileId": d.ProfileID.UUID().String(),
				"name":      d.Name,
				"isFriend":  true,
			}, nil
		}
	}
	for _, d := range resp.OutgoingRequests {
		if strings.EqualFold(d.Name, p.Name) {
			return map[string]any{
				"found":     true,
				"profileId": d.ProfileID.UUID().String(),
				"name":      d.Name,
				"isOutgoing": true,
			}, nil
		}
	}
	for _, d := range resp.IncomingRequests {
		if strings.EqualFold(d.Name, p.Name) {
			return map[string]any{
				"found":     true,
				"profileId": d.ProfileID.UUID().String(),
				"name":      d.Name,
				"isIncoming": true,
			}, nil
		}
	}
	return map[string]any{"found": false}, nil
}

type watchParams struct {
	IntervalSeconds int `json:"intervalSeconds"`
}

func (h *Handlers) friendsWatch(_ context.Context, raw json.RawMessage) (any, *RPCError) {
	if err := h.requireAuth(); err != nil {
		return nil, err
	}
	p := watchParams{IntervalSeconds: 20}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &p)
	}
	if p.IntervalSeconds < 5 {
		p.IntervalSeconds = 5
	}

	h.watch.mu.Lock()
	if h.watch.friendsRunning {
		h.watch.mu.Unlock()
		return map[string]any{"running": true}, nil
	}
	h.watch.friendsRunning = true
	h.watch.friendsStop = make(chan struct{})
	h.watch.friendsDone = make(chan struct{})
	stop := h.watch.friendsStop
	done := h.watch.friendsDone
	h.watch.mu.Unlock()

	interval := time.Duration(p.IntervalSeconds) * time.Second
	h.deps.Refresher.SetObserver(newWriterObserver(h.writer))
	h.deps.Refresher.SetAutoAccept(false)

	go h.runFriendsWatch(interval, stop, done)
	return map[string]any{"running": true, "intervalSeconds": p.IntervalSeconds}, nil
}

func (h *Handlers) runFriendsWatch(interval time.Duration, stop, done chan struct{}) {
	defer close(done)
	h.deps.Refresher.TickNow()
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			h.deps.Refresher.TickNow()
		}
	}
}

func (h *Handlers) friendsUnwatch(_ context.Context, _ json.RawMessage) (any, *RPCError) {
	h.watch.mu.Lock()
	if !h.watch.friendsRunning {
		h.watch.mu.Unlock()
		return map[string]any{"running": false}, nil
	}
	close(h.watch.friendsStop)
	done := h.watch.friendsDone
	h.watch.friendsRunning = false
	h.watch.mu.Unlock()
	<-done
	return map[string]any{"running": false}, nil
}

type autoAcceptParams struct {
	Enabled bool `json:"enabled"`
}

func (h *Handlers) friendsSetAutoAccept(_ context.Context, raw json.RawMessage) (any, *RPCError) {
	var p autoAcceptParams
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &p)
	}
	h.deps.Refresher.SetAutoAccept(p.Enabled)
	return map[string]any{"enabled": p.Enabled}, nil
}

type presenceSetParams struct {
	Status   string   `json:"status"`
	JoinVal  *string  `json:"joinValue,omitempty"`
	Invites  []string `json:"invites,omitempty"`
}

func (h *Handlers) presenceSet(_ context.Context, raw json.RawMessage) (any, *RPCError) {
	if err := h.requireAuth(); err != nil {
		return nil, err
	}
	var p presenceSetParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, newErrorf(CodeInvalidParams, "invalid params", err.Error())
		}
	}
	status := api.PresenceStatus(strings.ToUpper(strings.TrimSpace(p.Status)))
	if status == "" {
		status = api.StatusOnline
	}
	invites := make([]uuid.UUID, 0, len(p.Invites))
	for _, s := range p.Invites {
		if id, err := uuid.Parse(s); err == nil {
			invites = append(invites, id)
		}
	}
	req := api.PresenceRequest{Status: status}
	if status == api.StatusPlayingHostedServer {
		req.JoinInfo = api.NewJoinInfoUpdate(invites)
		if p.JoinVal != nil {
			req.JoinInfo.Value = p.JoinVal
		}
	} else if status != api.StatusOffline && len(invites) > 0 {
		req.JoinInfo = api.NewJoinInfoUpdate(invites)
	}

	resp, err := h.deps.APIClient.PostPresence(req)
	if err != nil {
		return nil, newErrorf(CodeAPIError, "POST /presence failed", err.Error())
	}

	h.watch.mu.Lock()
	h.watch.currentStatus = status
	h.watch.currentJoinVal = p.JoinVal
	h.watch.currentInvites = invites
	h.watch.mu.Unlock()

	if resp != nil {
		h.emitPresenceChanges(resp)
	}
	return map[string]any{"ok": true, "status": string(status)}, nil
}

func (h *Handlers) emitPresenceChanges(resp *api.PresenceResponse) {
	h.watch.mu.Lock()
	for _, p := range resp.Presence {
		id := p.ProfileID.UUID()
		next := string(p.Status)
		if prev := h.watch.lastPresence[id]; prev == next {
			continue
		}
		h.watch.lastPresence[id] = next
		payload := map[string]any{
			"profileId":   id.String(),
			"pmid":        p.PMID.UUID().String(),
			"status":      next,
			"lastUpdated": p.LastUpdated.UTC().Format(time.RFC3339),
		}
		if p.JoinInfo != nil {
			payload["invited"] = p.JoinInfo.Invited
			if p.JoinInfo.Value != nil {
				payload["joinValue"] = *p.JoinInfo.Value
			}
		}
		_ = h.writer.Notify("presence.changed", payload)
	}
	h.watch.mu.Unlock()
}

func (h *Handlers) presenceWatch(_ context.Context, raw json.RawMessage) (any, *RPCError) {
	if err := h.requireAuth(); err != nil {
		return nil, err
	}
	p := watchParams{IntervalSeconds: 30}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &p)
	}
	if p.IntervalSeconds < 10 {
		p.IntervalSeconds = 10
	}

	h.watch.mu.Lock()
	if h.watch.presenceRunning {
		h.watch.mu.Unlock()
		return map[string]any{"running": true}, nil
	}
	h.watch.presenceRunning = true
	h.watch.presenceStop = make(chan struct{})
	h.watch.presenceDone = make(chan struct{})
	stop := h.watch.presenceStop
	done := h.watch.presenceDone
	h.watch.mu.Unlock()

	interval := time.Duration(p.IntervalSeconds) * time.Second
	go h.runPresenceWatch(interval, stop, done)
	return map[string]any{"running": true, "intervalSeconds": p.IntervalSeconds}, nil
}

func (h *Handlers) runPresenceWatch(_ time.Duration, stop, done chan struct{}) {
	defer close(done)
	h.presenceTickNow()
	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()
	for {
		var coolTimer <-chan time.Time
		if remain := h.deps.APIClient.PresenceCooldownRemaining(); remain > 0 {
			coolTimer = time.After(remain + 2*time.Second)
		}
		select {
		case <-stop:
			return
		case <-tick.C:
			h.presenceTickIfDue()
		case <-coolTimer:
			h.presenceTickIfDue()
		}
	}
}

func (h *Handlers) presenceTickIfDue() {
	h.watch.mu.Lock()
	status := h.watch.currentStatus
	joinVal := h.watch.currentJoinVal
	invites := append([]uuid.UUID{}, h.watch.currentInvites...)
	lastPostedAt := h.watch.lastPostedAt
	dirty := presenceStateDiffers(status, joinVal, invites,
		h.watch.lastPostedStatus, h.watch.lastPostedJoinVal, h.watch.lastPostedInvites)
	h.watch.mu.Unlock()

	elapsed := time.Since(lastPostedAt)
	if lastPostedAt.IsZero() {
		elapsed = presenceMaxInterval + time.Second
	}
	if elapsed < presenceMinInterval {
		return
	}
	if !dirty && elapsed < presenceMaxInterval {
		return
	}
	h.presenceTickNow()
}

func (h *Handlers) presenceTickNow() {
	h.watch.mu.Lock()
	status := h.watch.currentStatus
	joinVal := h.watch.currentJoinVal
	invites := append([]uuid.UUID{}, h.watch.currentInvites...)
	h.watch.mu.Unlock()

	req := api.PresenceRequest{Status: status}
	if status == api.StatusPlayingHostedServer {
		req.JoinInfo = api.NewJoinInfoUpdate(invites)
		if joinVal != nil {
			req.JoinInfo.Value = joinVal
		}
	}
	resp, err := h.deps.APIClient.PostPresence(req)
	if err != nil {
		return
	}

	h.watch.mu.Lock()
	h.watch.lastPostedStatus = status
	h.watch.lastPostedJoinVal = joinVal
	h.watch.lastPostedInvites = invites
	h.watch.lastPostedAt = time.Now()
	h.watch.mu.Unlock()

	if resp != nil {
		h.emitPresenceChanges(resp)
	}
}

func presenceStateDiffers(curStatus api.PresenceStatus, curJoinVal *string, curInvites []uuid.UUID,
	prevStatus api.PresenceStatus, prevJoinVal *string, prevInvites []uuid.UUID) bool {
	if curStatus != prevStatus {
		return true
	}
	if (curJoinVal == nil) != (prevJoinVal == nil) {
		return true
	}
	if curJoinVal != nil && prevJoinVal != nil && *curJoinVal != *prevJoinVal {
		return true
	}
	if len(curInvites) != len(prevInvites) {
		return true
	}
	prevSet := make(map[uuid.UUID]struct{}, len(prevInvites))
	for _, id := range prevInvites {
		prevSet[id] = struct{}{}
	}
	for _, id := range curInvites {
		if _, ok := prevSet[id]; !ok {
			return true
		}
	}
	return false
}

func (h *Handlers) presenceUnwatch(_ context.Context, _ json.RawMessage) (any, *RPCError) {
	h.watch.mu.Lock()
	if !h.watch.presenceRunning {
		h.watch.mu.Unlock()
		return map[string]any{"running": false}, nil
	}
	close(h.watch.presenceStop)
	done := h.watch.presenceDone
	h.watch.presenceRunning = false
	h.watch.mu.Unlock()
	<-done
	return map[string]any{"running": false}, nil
}
