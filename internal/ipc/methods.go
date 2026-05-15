/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 */
package ipc

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"jp.zpw.openfriend/internal/api"
	"jp.zpw.openfriend/internal/auth"
	"jp.zpw.openfriend/internal/presence"
)

type Deps struct {
	Version   string
	Session   *auth.Session
	APIClient *api.Client
	Refresher *presence.Refresher
	AuthPath  string
	OnReset   func() error
	Blocklist *Blocklist
}

type Handlers struct {
	deps   Deps
	writer *Writer
	watch  *watchState
	host   *hostState

	authMu    sync.Mutex
	authRun   bool
	cachedTok *auth.Tokens
}

func NewHandlers(deps Deps, w *Writer) *Handlers {
	return &Handlers{deps: deps, writer: w, watch: newWatchState(), host: newHostState()}
}

func (h *Handlers) Register(s *Server) {
	s.Register("version", h.version)
	s.Register("auth.status", h.authStatus)
	s.Register("auth.signIn", h.authSignIn)
	s.Register("auth.signOut", h.authSignOut)
	s.Register("auth.useMojangSession", h.authUseMojangSession)
	s.Register("friends.list", h.friendsList)
	s.Register("friends.add", h.friendsAdd)
	s.Register("friends.remove", h.friendsRemove)
	s.Register("friends.accept", h.friendsAccept)
	s.Register("friends.decline", h.friendsDecline)
	s.Register("blocks.list", h.blocksList)
	s.Register("blocks.add", h.blocksAdd)
	s.Register("blocks.remove", h.blocksRemove)
	s.Register("quit", h.quit(s))
	h.registerWatchMethods(s)
	h.registerHostMethods(s)
}

func (h *Handlers) version(_ context.Context, _ json.RawMessage) (any, *RPCError) {
	return map[string]any{
		"version":        h.deps.Version,
		"protocolVersion": ProtocolVersion,
	}, nil
}

func (h *Handlers) authStatus(_ context.Context, _ json.RawMessage) (any, *RPCError) {
	t := h.deps.Session.CurrentTokens()
	if t == nil || t.ProfileID == uuid.Nil {
		return map[string]any{"authenticated": false}, nil
	}
	return map[string]any{
		"authenticated": true,
		"profileId":     t.ProfileID.String(),
		"name":          t.Name,
		"expiresAt":     t.MojangExpiresAt.UTC().Format(time.RFC3339),
	}, nil
}

func (h *Handlers) authSignIn(_ context.Context, raw json.RawMessage) (any, *RPCError) {
	h.authMu.Lock()
	if h.authRun {
		h.authMu.Unlock()
		return nil, newError(CodeAlreadyRunning, "sign-in already in progress")
	}
	h.authRun = true
	h.authMu.Unlock()
	defer func() {
		h.authMu.Lock()
		h.authRun = false
		h.authMu.Unlock()
	}()

	if len(raw) > 0 {
		var p struct {
			ExpectedProfileID string `json:"expectedProfileId"`
		}
		if err := json.Unmarshal(raw, &p); err == nil && p.ExpectedProfileID != "" {
			if expected, perr := uuid.Parse(p.ExpectedProfileID); perr == nil {
				cur := h.deps.Session.CurrentProfileID()
				if cur != uuid.Nil && cur != expected {
					slog.Default().Info("auth.signIn: launcher hint differs from cached profileId, clearing cache",
						"cached", cur.String(), "expected", expected.String())
					if h.deps.AuthPath != "" {
						_ = os.Remove(h.deps.AuthPath)
						_ = os.Remove(filepath.Join(filepath.Dir(h.deps.AuthPath), "friends-cache.json"))
					}
					h.deps.Session.ResetCache()
					if h.deps.APIClient != nil {
						h.deps.APIClient.ResetCache()
					}
				}
			}
		}
	}

	tokens, err := h.deps.Session.Authenticate()
	if err != nil {
		return nil, newErrorf(CodeAPIError, "authentication failed", err.Error())
	}
	h.writer.Notify("auth.signedIn", map[string]any{
		"profileId": tokens.ProfileID.String(),
		"name":      tokens.Name,
	})
	return map[string]any{
		"profileId": tokens.ProfileID.String(),
		"name":      tokens.Name,
	}, nil
}

func (h *Handlers) authUseMojangSession(_ context.Context, raw json.RawMessage) (any, *RPCError) {
	var p struct {
		AccessToken string `json:"accessToken"`
		ProfileID   string `json:"profileId"`
		Name        string `json:"name"`
		TTLSeconds  int    `json:"ttlSeconds"`
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, newErrorf(CodeInvalidParams, "invalid params", err.Error())
		}
	}
	if p.AccessToken == "" || p.ProfileID == "" {
		return nil, newError(CodeInvalidParams, "accessToken and profileId required")
	}
	id, err := uuid.Parse(p.ProfileID)
	if err != nil {
		return nil, newError(CodeInvalidParams, "invalid profileId")
	}
	ttl := p.TTLSeconds
	if ttl <= 0 {
		ttl = 23 * 3600
	}
	currentID := h.deps.Session.CurrentProfileID()
	if currentID != uuid.Nil && currentID != id {
		slog.Default().Info("auth.useMojangSession: launcher account changed, invalidating cache",
			"previous", currentID.String(), "incoming", id.String())
		if h.deps.AuthPath != "" {
			_ = os.Remove(h.deps.AuthPath)
			_ = os.Remove(filepath.Join(filepath.Dir(h.deps.AuthPath), "friends-cache.json"))
		}
		h.deps.Session.ResetCache()
		if h.deps.APIClient != nil {
			h.deps.APIClient.ResetCache()
		}
	}
	h.deps.Session.SetMojangSession(p.AccessToken, id, p.Name, time.Now().Add(time.Duration(ttl)*time.Second))
	_ = h.writer.Notify("auth.signedIn", map[string]any{
		"profileId": id.String(),
		"name":      p.Name,
	})
	return map[string]any{
		"authenticated": true,
		"profileId":     id.String(),
		"name":          p.Name,
	}, nil
}

func (h *Handlers) authSignOut(_ context.Context, _ json.RawMessage) (any, *RPCError) {
	if h.deps.OnReset != nil {
		if err := h.deps.OnReset(); err != nil {
			return nil, newErrorf(CodeInternalError, "reset failed", err.Error())
		}
	}
	return map[string]any{"ok": true}, nil
}

type idParams struct {
	ProfileID string `json:"profileId"`
	Name      string `json:"name"`
}

func parseIDParams(raw json.RawMessage) (*idParams, *RPCError) {
	if len(raw) == 0 {
		return &idParams{}, nil
	}
	var p idParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, newErrorf(CodeInvalidParams, "invalid params", err.Error())
	}
	return &p, nil
}

func (h *Handlers) requireAuth() *RPCError {
	if _, err := h.deps.Session.EnsureFreshSilent(); err != nil {
		return newErrorf(CodeNotAuthenticated, "not authenticated", err.Error())
	}
	return nil
}

func (h *Handlers) friendsList(_ context.Context, _ json.RawMessage) (any, *RPCError) {
	if err := h.requireAuth(); err != nil {
		return nil, err
	}
	resp, err := h.deps.APIClient.GetFriends()
	if err != nil {
		return nil, newErrorf(CodeAPIError, "GET /friends failed", err.Error())
	}
	if resp == nil {
		return map[string]any{
			"friends": []any{}, "incoming": []any{}, "outgoing": []any{},
		}, nil
	}
	blocked := func(id uuid.UUID) bool {
		if h.deps.Blocklist == nil {
			return false
		}
		return h.deps.Blocklist.Has(id)
	}
	dtoMap := func(list []api.FriendDto) []map[string]any {
		out := make([]map[string]any, 0, len(list))
		for _, d := range list {
			id := d.ProfileID.UUID()
			out = append(out, map[string]any{
				"profileId": id.String(),
				"name":      d.Name,
				"blocked":   blocked(id),
			})
		}
		return out
	}
	return map[string]any{
		"friends":  dtoMap(resp.Friends),
		"incoming": dtoMap(resp.IncomingRequests),
		"outgoing": dtoMap(resp.OutgoingRequests),
	}, nil
}

func (h *Handlers) friendsAdd(_ context.Context, raw json.RawMessage) (any, *RPCError) {
	if err := h.requireAuth(); err != nil {
		return nil, err
	}
	p, rerr := parseIDParams(raw)
	if rerr != nil {
		return nil, rerr
	}
	var action api.FriendActionRequest
	var debugTarget string
	switch {
	case p.ProfileID != "":
		id, err := uuid.Parse(p.ProfileID)
		if err != nil {
			return nil, newError(CodeInvalidParams, "invalid profileId")
		}
		action = api.AddByID(id)
		debugTarget = "profileId=" + id.String()
	case p.Name != "":
		action = api.AddByName(p.Name)
		debugTarget = "name=" + p.Name
	default:
		return nil, newError(CodeInvalidParams, "name or profileId required")
	}
	slog.Default().Info("friends.add request", "target", debugTarget)
	resp, err := h.deps.APIClient.PutFriendAction(action)
	if err != nil {
		slog.Default().Warn("friends.add failed", "target", debugTarget, "err", err.Error())
		return nil, friendActionRPCError(err)
	}
	if resp == nil {
		slog.Default().Info("friends.add ok (empty body)", "target", debugTarget)
	} else {
		slog.Default().Info("friends.add ok",
			"target", debugTarget,
			"friends", len(resp.Friends),
			"incoming", len(resp.IncomingRequests),
			"outgoing", len(resp.OutgoingRequests))
	}
	return map[string]any{"ok": true}, nil
}

func (h *Handlers) friendsRemove(_ context.Context, raw json.RawMessage) (any, *RPCError) {
	if err := h.requireAuth(); err != nil {
		return nil, err
	}
	p, rerr := parseIDParams(raw)
	if rerr != nil {
		return nil, rerr
	}
	if p.ProfileID == "" {
		return nil, newError(CodeInvalidParams, "profileId required")
	}
	id, err := uuid.Parse(p.ProfileID)
	if err != nil {
		return nil, newError(CodeInvalidParams, "invalid profileId")
	}
	if _, err := h.deps.APIClient.PutFriendAction(api.RemoveByID(id)); err != nil {
		return nil, friendActionRPCError(err)
	}
	return map[string]any{"ok": true}, nil
}

func friendActionRPCError(err error) *RPCError {
	if apiErr, ok := err.(*api.APIError); ok {
		body := strings.ToLower(apiErr.Body)
		switch apiErr.Status {
		case 403:
			switch {
			case strings.Contains(body, "does not have friends enabled") ||
				strings.Contains(body, "accept invites"):
				return newError(CodeAPIError,
					"Xbox account privacy settings may need to be adjusted on the other side.")
			case strings.Contains(body, "privacy"):
				return newError(CodeAPIError,
					"Blocked by Xbox Live privacy settings.")
			case strings.Contains(body, "blocked"):
				return newError(CodeAPIError,
					"Blocked by Xbox block list.")
			default:
				return newError(CodeAPIError,
					"Mojang refused the request (403).")
			}
		case 404:
			return newError(CodeAPIError, "Player not found. Check the gamertag.")
		case 409:
			return newError(CodeAPIError, "Already friends or pending.")
		case 429:
			return newError(CodeAPIError, "Rate limited by Mojang. Please wait a moment.")
		}
	}
	return newError(CodeAPIError, err.Error())
}

func (h *Handlers) friendsAccept(_ context.Context, raw json.RawMessage) (any, *RPCError) {
	return h.friendsAdd(nil, raw)
}

func (h *Handlers) friendsDecline(_ context.Context, raw json.RawMessage) (any, *RPCError) {
	return h.friendsRemove(nil, raw)
}

func (h *Handlers) blocksList(_ context.Context, _ json.RawMessage) (any, *RPCError) {
	if h.deps.Blocklist == nil {
		return map[string]any{"blocks": []any{}}, nil
	}
	return map[string]any{"blocks": h.deps.Blocklist.Snapshot()}, nil
}

func (h *Handlers) blocksAdd(_ context.Context, raw json.RawMessage) (any, *RPCError) {
	if h.deps.Blocklist == nil {
		return nil, newError(CodeInternalError, "blocklist not configured")
	}
	p, rerr := parseIDParams(raw)
	if rerr != nil {
		return nil, rerr
	}
	if p.ProfileID == "" {
		return nil, newError(CodeInvalidParams, "profileId required")
	}
	id, err := uuid.Parse(p.ProfileID)
	if err != nil {
		return nil, newError(CodeInvalidParams, "invalid profileId")
	}
	if err := h.deps.Blocklist.Add(id, p.Name); err != nil {
		return nil, newErrorf(CodeInternalError, "blocklist save failed", err.Error())
	}
	return map[string]any{"ok": true}, nil
}

func (h *Handlers) blocksRemove(_ context.Context, raw json.RawMessage) (any, *RPCError) {
	if h.deps.Blocklist == nil {
		return nil, newError(CodeInternalError, "blocklist not configured")
	}
	p, rerr := parseIDParams(raw)
	if rerr != nil {
		return nil, rerr
	}
	if p.ProfileID == "" {
		return nil, newError(CodeInvalidParams, "profileId required")
	}
	id, err := uuid.Parse(p.ProfileID)
	if err != nil {
		return nil, newError(CodeInvalidParams, "invalid profileId")
	}
	if err := h.deps.Blocklist.Remove(id); err != nil {
		return nil, newErrorf(CodeInternalError, "blocklist save failed", err.Error())
	}
	return map[string]any{"ok": true}, nil
}

func (h *Handlers) quit(s *Server) MethodFunc {
	return func(_ context.Context, _ json.RawMessage) (any, *RPCError) {
		go func() {
			time.Sleep(50 * time.Millisecond)
			s.Quit()
		}()
		return map[string]any{"ok": true}, nil
	}
}

