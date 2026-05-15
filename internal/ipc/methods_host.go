/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 */
package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"jp.zpw.openfriend/internal/api"
	"jp.zpw.openfriend/internal/bridge"
	"jp.zpw.openfriend/internal/bypass"
	"jp.zpw.openfriend/internal/presence"
	"jp.zpw.openfriend/internal/signaling"
)

type hostState struct {
	mu sync.Mutex

	running    bool
	target     string
	bypassUsed bool

	sig *signaling.Client
	pm  *bridge.HostManager
	bc  *presence.Broadcaster

	joinRunning bool
	joinPeer    string
	joinListen  string
	jm          *bridge.JoinManager
	joinSig     *signaling.Client
}

func newHostState() *hostState { return &hostState{} }

func (h *Handlers) registerHostMethods(s *Server) {
	s.Register("host.start", h.hostStart)
	s.Register("host.stop", h.hostStop)
	s.Register("host.status", h.hostStatus)
	s.Register("join.start", h.joinStart)
	s.Register("join.stop", h.joinStop)
}

type hostStartParams struct {
	Target        string `json:"target"`
	BypassKeyPath string `json:"bypassKeyPath,omitempty"`
	UseBypass     bool   `json:"useBypass"`
}

func (h *Handlers) hostStart(_ context.Context, raw json.RawMessage) (any, *RPCError) {
	if err := h.requireAuth(); err != nil {
		return nil, err
	}
	var p hostStartParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, newErrorf(CodeInvalidParams, "invalid params", err.Error())
		}
	}
	p.Target = strings.TrimSpace(p.Target)
	if p.Target == "" {
		return nil, newError(CodeInvalidParams, "target required (host:port)")
	}
	if _, err := bridge.ParseTarget(p.Target); err != nil {
		return nil, newErrorf(CodeInvalidParams, "invalid target", err.Error())
	}

	h.host.mu.Lock()
	if h.host.running {
		current := h.host.target
		h.host.mu.Unlock()
		return nil, newErrorf(CodeAlreadyRunning, "host already running", map[string]any{"target": current})
	}
	h.host.mu.Unlock()

	var bypassBytes []byte
	if p.UseBypass {
		path := p.BypassKeyPath
		if path == "" {
			path = filepath.Join(filepath.Dir(h.deps.AuthPath), "bypass.pem")
		}
		key, err := bypass.LoadOrAbsent(path)
		if err != nil {
			return nil, newErrorf(CodeAPIError, "failed to load bypass key", err.Error())
		}
		if key == nil {
			return nil, newError(CodeNotFound, "bypass key not found")
		}
		bypassBytes = key.Bytes
	}

	writer := h.writer

	var sig *signaling.Client
	var pm *bridge.HostManager
	sig = signaling.NewClient(h.deps.Session, func(fromPmid uuid.UUID, payload map[string]any) {
		_ = writer.Notify("friend.joined", map[string]any{"pmid": fromPmid.String()})
		if pm != nil {
			pm.OnFriendJoin(fromPmid, payload)
		}
	}, slog.Default())
	pm = bridge.NewHostManager(sig, p.Target, bypassBytes, slog.Default())

	bc := presence.NewBroadcaster(h.deps.APIClient, api.StatusPlayingHostedServer, 30*time.Second, nil, slog.Default())

	h.host.mu.Lock()
	h.host.sig = sig
	h.host.pm = pm
	h.host.bc = bc
	h.host.target = p.Target
	h.host.bypassUsed = bypassBytes != nil
	h.host.running = true
	h.host.mu.Unlock()

	sig.Connect()
	bc.Start()

	h.watch.mu.Lock()
	h.watch.currentStatus = api.StatusPlayingHostedServer
	h.watch.mu.Unlock()

	_ = writer.Notify("host.started", map[string]any{
		"target":    p.Target,
		"useBypass": bypassBytes != nil,
	})

	return map[string]any{
		"running":   true,
		"target":    p.Target,
		"useBypass": bypassBytes != nil,
	}, nil
}

func (h *Handlers) hostStop(_ context.Context, _ json.RawMessage) (any, *RPCError) {
	h.host.mu.Lock()
	if !h.host.running {
		h.host.mu.Unlock()
		return map[string]any{"running": false}, nil
	}
	sig := h.host.sig
	pm := h.host.pm
	bc := h.host.bc
	h.host.sig = nil
	h.host.pm = nil
	h.host.bc = nil
	h.host.running = false
	target := h.host.target
	h.host.target = ""
	h.host.bypassUsed = false
	h.host.mu.Unlock()

	if bc != nil {
		bc.Close()
	}
	if pm != nil {
		pm.Close()
	}
	if sig != nil {
		sig.Close()
	}

	h.watch.mu.Lock()
	h.watch.currentStatus = api.StatusOnline
	h.watch.mu.Unlock()

	_, _ = h.deps.APIClient.PostPresence(api.PresenceRequest{Status: api.StatusOnline})

	_ = h.writer.Notify("host.stopped", map[string]any{"target": target})
	return map[string]any{"running": false}, nil
}

func (h *Handlers) hostStatus(_ context.Context, _ json.RawMessage) (any, *RPCError) {
	h.host.mu.Lock()
	defer h.host.mu.Unlock()
	return map[string]any{
		"running":   h.host.running,
		"target":    h.host.target,
		"useBypass": h.host.bypassUsed,
	}, nil
}

type joinStartParams struct {
	Name      string `json:"name"`
	ProfileID string `json:"profileId"`
	PMID      string `json:"pmid"`
	Listen    string `json:"listen"`
}

func (h *Handlers) joinStart(_ context.Context, raw json.RawMessage) (any, *RPCError) {
	if err := h.requireAuth(); err != nil {
		return nil, err
	}
	var p joinStartParams
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, newErrorf(CodeInvalidParams, "invalid params", err.Error())
		}
	}
	if p.Listen == "" {
		p.Listen = "127.0.0.1:25577"
	}
	if p.Name == "" && p.ProfileID == "" && p.PMID == "" {
		return nil, newError(CodeInvalidParams, "name, profileId, or pmid required")
	}

	h.host.mu.Lock()
	prevJm := h.host.jm
	prevSig := h.host.joinSig
	prevListen := h.host.joinListen
	prevRunning := h.host.joinRunning
	prevPeer := h.host.joinPeer
	h.host.jm = nil
	h.host.joinSig = nil
	h.host.joinRunning = false
	h.host.joinPeer = ""
	h.host.joinListen = ""
	h.host.mu.Unlock()

	if prevRunning && prevPeer == strings.TrimSpace(p.Name) && prevListen == p.Listen {
		h.host.mu.Lock()
		h.host.jm = prevJm
		h.host.joinSig = prevSig
		h.host.joinPeer = prevPeer
		h.host.joinListen = prevListen
		h.host.joinRunning = true
		h.host.mu.Unlock()
		return map[string]any{
			"running": true,
			"peer":    prevPeer,
			"pmid":    "",
			"listen":  prevListen,
		}, nil
	}

	if prevJm != nil {
		prevJm.Close()
	}
	if prevSig != nil {
		prevSig.Close()
	}

	peer, rerr := h.resolveJoinPeer(p)
	if rerr != nil {
		return nil, rerr
	}

	writer := h.writer
	var sig *signaling.Client
	var jm *bridge.JoinManager
	sig = signaling.NewClient(h.deps.Session, func(fromPmid uuid.UUID, payload map[string]any) {
		if jm != nil {
			jm.OnFriendJoin(fromPmid, payload)
		}
	}, slog.Default())
	jm = bridge.NewJoinManager(sig, peer.PMID, slog.Default())

	actualListen := p.Listen
	if err := jm.Listen(p.Listen); err != nil {
		slog.Default().Warn("join.start primary listen failed; retrying on random port", "addr", p.Listen, "err", err.Error())
		if err2 := jm.Listen("127.0.0.1:0"); err2 != nil {
			return nil, newError(CodeAPIError, "listen failed: "+err.Error())
		}
		if jm.ListenAddr() != "" {
			actualListen = jm.ListenAddr()
		}
	}
	sig.Connect()

	h.host.mu.Lock()
	h.host.joinSig = sig
	h.host.jm = jm
	h.host.joinPeer = peer.Name
	h.host.joinListen = actualListen
	h.host.joinRunning = true
	h.host.mu.Unlock()

	_ = writer.Notify("join.started", map[string]any{
		"peer":   peer.Name,
		"pmid":   peer.PMID.String(),
		"listen": actualListen,
	})
	return map[string]any{
		"running": true,
		"peer":    peer.Name,
		"pmid":    peer.PMID.String(),
		"listen":  actualListen,
	}, nil
}

func (h *Handlers) joinStop(_ context.Context, _ json.RawMessage) (any, *RPCError) {
	h.host.mu.Lock()
	if !h.host.joinRunning {
		h.host.mu.Unlock()
		return map[string]any{"running": false}, nil
	}
	jm := h.host.jm
	sig := h.host.joinSig
	h.host.jm = nil
	h.host.joinSig = nil
	h.host.joinRunning = false
	peer := h.host.joinPeer
	h.host.joinPeer = ""
	h.host.joinListen = ""
	h.host.mu.Unlock()

	if jm != nil {
		jm.Close()
	}
	if sig != nil {
		sig.Close()
	}
	_ = h.writer.Notify("join.stopped", map[string]any{"peer": peer})
	return map[string]any{"running": false}, nil
}

func (h *Handlers) resolveJoinPeer(p joinStartParams) (*api.Peer, *RPCError) {
	if p.PMID != "" {
		id, err := uuid.Parse(p.PMID)
		if err != nil {
			return nil, newError(CodeInvalidParams, "invalid pmid")
		}
		return &api.Peer{PMID: id}, nil
	}
	query := p.Name
	if query == "" {
		query = p.ProfileID
	}
	peer, err := api.ResolvePeer(h.deps.APIClient, query)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, newError(CodeNotFound, err.Error())
		}
		return nil, newErrorf(CodeAPIError, "resolve peer failed", err.Error())
	}
	if peer.PMID == uuid.Nil {
		return nil, newError(CodeNotFound, "peer pmid unknown (presence may be unavailable)")
	}
	return peer, nil
}
