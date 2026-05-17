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
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
	"jp.zpw.openfriend/internal/signaling"
)

type JoinManager struct {
	sig      *signaling.Client
	peerPmid uuid.UUID
	logger   *slog.Logger
	api      *webrtc.API

	listener net.Listener
	stopCh   chan struct{}
	doneCh   chan struct{}

	mu      sync.Mutex
	current *joinSession
}

func NewJoinManager(sig *signaling.Client, peerPmid uuid.UUID, logger *slog.Logger) *JoinManager {
	if logger == nil {
		logger = slog.Default()
	}
	se := webrtc.SettingEngine{}
	se.SetNetworkTypes([]webrtc.NetworkType{
		webrtc.NetworkTypeUDP4,
		webrtc.NetworkTypeUDP6,
	})
	// ICE timeouts: keep the keep-alive tight, but give failed/disconnected
	// transitions more room — heavy modpack handshakes (Forge mod registry
	// sync) can take 30+ seconds, during which transient ICE state churn
	// must not abort the session.
	se.SetICETimeouts(10*time.Second, 30*time.Second, 2*time.Second)
	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	jm := &JoinManager{
		sig:      sig,
		peerPmid: peerPmid,
		logger:   logger,
		api:      api,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
	sig.SetWebRtcListener(jm.onWebRtc)
	return jm
}

func (j *JoinManager) Listen(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	j.listener = ln
	j.logger.Info("Join listener up", "addr", ln.Addr().String(), "peer", j.peerPmid)
	go j.acceptLoop()
	return nil
}

func (j *JoinManager) ListenAddr() string {
	if j.listener == nil {
		return ""
	}
	return j.listener.Addr().String()
}

func (j *JoinManager) acceptLoop() {
	defer close(j.doneCh)
	for {
		conn, err := j.listener.Accept()
		if err != nil {
			select {
			case <-j.stopCh:
				return
			default:
			}
			j.logger.Warn("Listener accept failed", "err", err)
			return
		}
		go j.handleIncoming(conn)
	}
}

func (j *JoinManager) handleIncoming(local net.Conn) {
	j.mu.Lock()
	if j.current != nil {
		j.mu.Unlock()
		j.logger.Warn("Rejecting local TCP — a join session is already in progress")
		_ = local.Close()
		return
	}
	sid := uuid.New().String()
	sess := &joinSession{
		jm:        j,
		sessionID: sid,
		local:     local,
		acceptCh:  make(chan struct{}),
		rejectCh:  make(chan struct{}),
	}
	j.current = sess
	j.mu.Unlock()

	defer func() {
		j.mu.Lock()
		if j.current == sess {
			j.current = nil
		}
		j.mu.Unlock()
	}()

	j.logger.Info("[join] local client connected", "sid", sid, "addr", local.RemoteAddr())

	waitCtx, waitCancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := j.sig.WaitReady(waitCtx); err != nil {
		waitCancel()
		j.logger.Warn("Signaling not ready for JOIN_REQUEST", "err", err.Error())
		_ = local.Close()
		return
	}
	waitCancel()

	if err := j.sig.SendClientMessage(j.peerPmid, signaling.JoinRequest(sid)); err != nil {
		j.logger.Warn("Failed to send JOIN_REQUEST", "err", err.Error())
		_ = local.Close()
		return
	}
	j.logger.Info("[join] JOIN_REQUEST sent; awaiting host accept", "sid", sid)

	select {
	case <-sess.acceptCh:
	case <-sess.rejectCh:
		j.logger.Warn("[join] host rejected JOIN_REQUEST", "sid", sid)
		_ = local.Close()
		return
	case <-time.After(60 * time.Second):
		j.logger.Warn("[join] host did not respond within 60s", "sid", sid)
		_ = local.Close()
		return
	}

	if err := sess.startInitiator(); err != nil {
		j.logger.Warn("[join] initiator setup failed", "err", err)
		sess.close()
		return
	}

	dc, err := sess.rtc.WaitDataChannelOpen(context.Background(), 15*time.Second)
	if err != nil {
		j.logger.Warn("[join] DataChannel did not open", "err", err)
		sess.close()
		return
	}
	_ = dc
	j.logger.Info("[join] DataChannel open; bridging", "sid", sid)
	sess.startTcpBridge()
}

func (j *JoinManager) onWebRtc(fromPmid uuid.UUID, payload map[string]any) {
	if fromPmid != j.peerPmid {
		return
	}
	typ, _ := payload["type"].(string)
	switch typ {
	case "ANSWER":
		j.handleAnswer(payload)
	case "ICE_CANDIDATE":
		j.handleICE(payload)
	case "OFFER":
		j.logger.Debug("[join] unexpected OFFER (we're initiator)")
	}
}

func (j *JoinManager) OnFriendJoin(fromPmid uuid.UUID, payload map[string]any) {
	if fromPmid != j.peerPmid {
		return
	}
	typ, _ := payload["type"].(string)
	sid := signaling.GetSessionID(payload)

	j.mu.Lock()
	sess := j.current
	j.mu.Unlock()
	if sess == nil || sess.sessionID != sid {
		return
	}

	switch typ {
	case "JOIN_ACCEPTED":
		j.logger.Info("[join] host accepted", "sid", sid)
		close(sess.acceptCh)
	case "JOIN_REJECTED":
		close(sess.rejectCh)
	}
}

func (j *JoinManager) handleAnswer(payload map[string]any) {
	sid := signaling.GetSessionID(payload)
	sdp := signaling.GetSDP(payload)
	j.mu.Lock()
	sess := j.current
	j.mu.Unlock()
	if sess == nil || sess.sessionID != sid || sess.rtc == nil {
		return
	}
	if err := sess.rtc.HandleAnswer(sdp); err != nil {
		j.logger.Warn("[join] HandleAnswer failed", "err", err)
		sess.close()
	}
}

func (j *JoinManager) handleICE(payload map[string]any) {
	sid := signaling.GetSessionID(payload)
	ic, ok := signaling.ParseIceCandidate(payload)
	if !ok {
		return
	}
	j.mu.Lock()
	sess := j.current
	j.mu.Unlock()
	if sess == nil || sess.sessionID != sid || sess.rtc == nil {
		return
	}
	mid := ic.SdpMid
	idx := uint16(ic.SdpMLineIndex)
	sess.rtc.AddRemoteICE(webrtc.ICECandidateInit{
		Candidate:     ic.Candidate,
		SDPMid:        &mid,
		SDPMLineIndex: &idx,
	})
}

func (j *JoinManager) Close() {
	close(j.stopCh)
	if j.listener != nil {
		_ = j.listener.Close()
	}
	<-j.doneCh
	j.mu.Lock()
	sess := j.current
	j.current = nil
	j.mu.Unlock()
	if sess != nil {
		sess.close()
	}
}

type joinSession struct {
	jm        *JoinManager
	sessionID string
	local     net.Conn
	rtc       *Session
	acceptCh  chan struct{}
	rejectCh  chan struct{}

	once   sync.Once
	closed bool
	mu     sync.Mutex
}

func (s *joinSession) startInitiator() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	turn, err := s.jm.sig.RequestTurnAuth(ctx)
	if err != nil {
		return fmt.Errorf("turn auth: %w", err)
	}
	cfg := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
			{
				URLs:       turn.URLs,
				Username:   turn.Username,
				Credential: turn.Password,
			},
		},
	}

	rtc, err := NewSession(s.jm.api, cfg, RoleInitiator, s.sessionID, s.jm.peerPmid,
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
			_ = s.jm.sig.SendClientMessage(s.jm.peerPmid, signaling.IceCandidate(s.sessionID, init.Candidate, mid, idx))
		},
		func(data []byte) {
			s.mu.Lock()
			local := s.local
			s.mu.Unlock()
			if local != nil {
				_, _ = local.Write(data)
			}
		},
		func() {
			s.close()
		},
		s.jm.logger,
	)
	if err != nil {
		return err
	}
	s.rtc = rtc

	offerSDP, err := rtc.CreateOffer()
	if err != nil {
		return fmt.Errorf("create offer: %w", err)
	}
	if err := s.jm.sig.SendClientMessage(s.jm.peerPmid, signaling.Offer(s.sessionID, offerSDP)); err != nil {
		return fmt.Errorf("send offer: %w", err)
	}
	s.jm.logger.Info("[join] OFFER sent", "sid", s.sessionID)
	return nil
}

func (s *joinSession) startTcpBridge() {
	const maxBuffered uint64 = 1 * 1024 * 1024 // 1 MiB SCTP send queue cap
	buf := make([]byte, 16*1024)
	for {
		// Backpressure: wait for SCTP queue to drain before reading more from
		// MC. Without this, a fast modpack login handshake (Forge mod registry
		// for 90+ mods) floods the queue and Pion may silently drop messages —
		// MC handshake then fails halfway through with no error log.
		if err := s.rtc.WaitForBufferDrain(maxBuffered); err != nil {
			s.jm.logger.Warn("[join] backpressure wait failed", "err", err)
			s.close()
			return
		}
		n, err := s.local.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			if err := s.rtc.Send(chunk); err != nil {
				s.jm.logger.Warn("[join] DataChannel send failed", "err", err)
				s.close()
				return
			}
		}
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				s.jm.logger.Debug("[join] local TCP read ended", "err", err)
			}
			s.close()
			return
		}
	}
}

func (s *joinSession) close() {
	s.once.Do(func() {
		s.mu.Lock()
		s.closed = true
		local := s.local
		s.local = nil
		s.mu.Unlock()
		if local != nil {
			_ = local.Close()
		}
		if s.rtc != nil {
			s.rtc.Close()
		}
	})
}
