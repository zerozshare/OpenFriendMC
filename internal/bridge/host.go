/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package bridge

import (
	"context"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
	"jp.zpw.openfriend/internal/signaling"
)

const (
	handshakeTimeout  = 10 * time.Second
	targetProbeTimeout = 1 * time.Second
)

type HostManager struct {
	sig       *signaling.Client
	target    string
	logger    *slog.Logger
	api       *webrtc.API
	bypassKey []byte

	mu       sync.Mutex
	sessions map[uuid.UUID]*hostSession
}

func NewHostManager(sig *signaling.Client, target string, bypassKey []byte, logger *slog.Logger) *HostManager {
	if logger == nil {
		logger = slog.Default()
	}
	se := webrtc.SettingEngine{}
	se.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeUDP4,
		webrtc.NetworkTypeUDP6,
	})
	se.SetICETimeouts(3*time.Second, 8*time.Second, 1*time.Second)
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	pm := &HostManager{
		sig:       sig,
		target:    target,
		logger:    logger,
		api:       api,
		bypassKey: bypassKey,
		sessions:  map[uuid.UUID]*hostSession{},
	}
	sig.SetWebRtcListener(pm.onWebRtc)
	return pm
}

func (p *HostManager) OnFriendJoin(fromPmid uuid.UUID, payload map[string]any) {
	typ, _ := payload["type"].(string)
	sid := signaling.GetSessionID(payload)
	switch typ {
	case "JOIN_REQUEST":
		p.logger.Info("[proxy] JOIN_REQUEST", "pmid", fromPmid, "sid", sid)
		go func() {
			if ProbeTCP(p.target, targetProbeTimeout) {
				p.logger.Info("[proxy] target reachable; accepting", "sid", sid)
				_ = p.sig.SendClientMessage(fromPmid, signaling.JoinAccepted(sid))
			} else {
				p.logger.Warn("[proxy] target unreachable; rejecting", "target", p.target, "sid", sid)
				_ = p.sig.SendClientMessage(fromPmid, signaling.JoinRejected(sid))
			}
		}()
	case "INVITE_DECLINED":
		p.logger.Info("[proxy] INVITE_DECLINED", "pmid", fromPmid)
	default:
		p.logger.Debug("[proxy] ignoring FriendJoin", "type", typ)
	}
}

func (p *HostManager) onWebRtc(fromPmid uuid.UUID, payload map[string]any) {
	typ, _ := payload["type"].(string)
	switch typ {
	case "OFFER":
		p.handleOffer(fromPmid, payload)
	case "ICE_CANDIDATE":
		p.handleICE(fromPmid, payload)
	case "ANSWER":
		p.logger.Debug("[proxy] unexpected ANSWER")
	}
}

func (p *HostManager) handleOffer(fromPmid uuid.UUID, payload map[string]any) {
	sid := signaling.GetSessionID(payload)
	sdp := signaling.GetSDP(payload)
	if sdp == "" {
		p.logger.Warn("[proxy] OFFER without sdp")
		return
	}
	p.logger.Info("[proxy] OFFER; requesting TURN auth", "pmid", fromPmid, "sid", sid)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		turn, err := p.sig.RequestTurnAuth(ctx)
		if err != nil {
			p.logger.Warn("TURN auth failed", "err", err)
			return
		}
		p.startSession(fromPmid, sid, sdp, turn)
	}()
}

func (p *HostManager) startSession(fromPmid uuid.UUID, sid, offerSDP string, turn *signaling.TurnAuth) {
	cfg := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{{
			URLs:       turn.URLs,
			Username:   turn.Username,
			Credential: turn.Password,
		}},
	}

	ps := &hostSession{
		fromPmid: fromPmid,
		sid:      sid,
		pm:       p,
	}
	p.mu.Lock()
	if prev, ok := p.sessions[fromPmid]; ok {
		prev.close()
	}
	p.sessions[fromPmid] = ps
	p.mu.Unlock()

	rtc, err := NewSession(p.api, cfg, RoleAcceptor, sid, fromPmid,
		func(c *webrtc.ICECandidate) {
			init := c.ToJSON()
			mid := "0"
			if init.SDPMid != nil {
				mid = *init.SDPMid
			}
			idx := 0
			if init.SDPMLineIndex != nil {
				idx = int(*init.SDPMLineIndex)
			}
			_ = p.sig.SendClientMessage(fromPmid, signaling.IceCandidate(sid, init.Candidate, mid, idx))
		},
		func(data []byte) {
			ps.onPeerData(data)
		},
		func() {
			p.logger.Info("[proxy] DataChannel closed", "sid", sid)
			ps.close()
			p.removeSession(fromPmid, ps)
		},
		p.logger,
	)
	if err != nil {
		p.logger.Warn("New WebRTC session failed", "err", err)
		return
	}
	ps.rtc = rtc

	answerSDP, err := rtc.HandleOffer(offerSDP)
	if err != nil {
		p.logger.Warn("HandleOffer failed", "err", err)
		ps.close()
		return
	}
	if err := p.sig.SendClientMessage(fromPmid, signaling.Answer(sid, answerSDP)); err != nil {
		p.logger.Warn("Send ANSWER failed", "err", err)
		ps.close()
		return
	}
	p.logger.Info("[proxy] ANSWER sent", "sid", sid)

	go func() {
		dc, err := rtc.WaitDataChannelOpen(context.Background(), handshakeTimeout)
		if err != nil {
			p.logger.Warn("[proxy] handshake timeout; aborting", "sid", sid, "err", err)
			ps.close()
			p.removeSession(fromPmid, ps)
			return
		}
		_ = dc
		p.logger.Info("[proxy] DataChannel open; dialing target", "sid", sid, "target", p.target)
		ps.attachTCP(p.target)
	}()
}

func (p *HostManager) handleICE(fromPmid uuid.UUID, payload map[string]any) {
	ic, ok := signaling.ParseIceCandidate(payload)
	if !ok {
		return
	}
	p.mu.Lock()
	ps, ok := p.sessions[fromPmid]
	p.mu.Unlock()
	if !ok || ps.rtc == nil {
		return
	}
	mid := ic.SdpMid
	idx := uint16(ic.SdpMLineIndex)
	ps.rtc.AddRemoteICE(webrtc.ICECandidateInit{
		Candidate:     ic.Candidate,
		SDPMid:        &mid,
		SDPMLineIndex: &idx,
	})
}

func (p *HostManager) removeSession(fromPmid uuid.UUID, ps *hostSession) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if cur, ok := p.sessions[fromPmid]; ok && cur == ps {
		delete(p.sessions, fromPmid)
	}
}

func (p *HostManager) Close() {
	p.mu.Lock()
	all := make([]*hostSession, 0, len(p.sessions))
	for _, ps := range p.sessions {
		all = append(all, ps)
	}
	p.sessions = map[uuid.UUID]*hostSession{}
	p.mu.Unlock()
	for _, ps := range all {
		ps.close()
	}
}

type hostSession struct {
	fromPmid uuid.UUID
	sid      string
	pm       *HostManager

	mu             sync.Mutex
	rtc            *Session
	tcp            *TCPBridge
	closed         bool
	handshakeBuf   []byte
	handshakeDone  bool
}

func (ps *hostSession) attachTCP(target string) {
	tcp, err := DialTCP(target,
		func(downstream []byte) {
			ps.mu.Lock()
			rtc := ps.rtc
			ps.mu.Unlock()
			if rtc != nil {
				_ = rtc.Send(downstream)
			}
		},
		func() {
			ps.close()
			ps.pm.removeSession(ps.fromPmid, ps)
		},
		ps.pm.logger,
	)
	if err != nil {
		ps.pm.logger.Warn("Failed to dial target", "target", target, "err", err)
		ps.close()
		return
	}
	ps.mu.Lock()
	ps.tcp = tcp
	ps.mu.Unlock()
}

func (ps *hostSession) onPeerData(data []byte) {
	ps.mu.Lock()
	tcp := ps.tcp
	done := ps.handshakeDone || ps.pm.bypassKey == nil
	ps.mu.Unlock()
	if tcp == nil {
		return
	}

	if done {
		if err := tcp.Feed(data); err != nil {
			ps.pm.logger.Warn("TCP write failed", "err", err)
			ps.close()
		}
		return
	}

	ps.feedHandshake(tcp, data)
}

func (ps *hostSession) feedHandshake(tcp *TCPBridge, data []byte) {
	ps.mu.Lock()
	ps.handshakeBuf = append(ps.handshakeBuf, data...)
	buf := ps.handshakeBuf
	ps.mu.Unlock()

	payload, consumed, ok, err := readFramedPacket(buf)
	if err != nil {
		ps.pm.logger.Warn("Handshake frame decode failed; passing through", "err", err)
		ps.markHandshakeDone()
		if writeErr := tcp.Feed(buf); writeErr != nil {
			ps.close()
		}
		return
	}
	if !ok {
		return
	}

	h, err := decodeHandshake(payload)
	if err != nil {
		ps.pm.logger.Warn("Handshake packet decode failed; passing through", "err", err)
		ps.markHandshakeDone()
		if writeErr := tcp.Feed(buf); writeErr != nil {
			ps.close()
		}
		return
	}

	newAddr, err := InjectBypassMarker(h.ServerAddress, ps.pm.bypassKey)
	if err != nil {
		ps.pm.logger.Warn("Bypass marker injection failed; passing through", "err", err)
		ps.markHandshakeDone()
		if writeErr := tcp.Feed(buf); writeErr != nil {
			ps.close()
		}
		return
	}
	h.ServerAddress = newAddr
	modified := encodeHandshake(h)
	rest := buf[consumed:]

	ps.markHandshakeDone()
	if err := tcp.Feed(modified); err != nil {
		ps.close()
		return
	}
	if len(rest) > 0 {
		if err := tcp.Feed(rest); err != nil {
			ps.close()
		}
	}
	ps.pm.logger.Info("[bypass] handshake marker injected", "sid", ps.sid,
		"proto", h.ProtocolVersion, "addr_len", len(newAddr))
}

func (ps *hostSession) markHandshakeDone() {
	ps.mu.Lock()
	ps.handshakeDone = true
	ps.handshakeBuf = nil
	ps.mu.Unlock()
}

func (ps *hostSession) close() {
	ps.mu.Lock()
	if ps.closed {
		ps.mu.Unlock()
		return
	}
	ps.closed = true
	tcp := ps.tcp
	rtc := ps.rtc
	ps.tcp = nil
	ps.rtc = nil
	ps.mu.Unlock()
	if tcp != nil {
		tcp.Close()
	}
	if rtc != nil {
		rtc.Close()
	}
}

func ParseTarget(s string) (string, error) {
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return "", err
	}
	if _, err := strconv.Atoi(port); err != nil {
		return "", err
	}
	return net.JoinHostPort(host, port), nil
}
