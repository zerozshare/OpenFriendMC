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
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
)

type Role int

const (
	RoleAcceptor Role = iota
	RoleInitiator
)

type Session struct {
	SessionID string
	PeerPmid  uuid.UUID
	Role      Role

	pc     *webrtc.PeerConnection
	dc     *webrtc.DataChannel
	logger *slog.Logger

	mu        sync.Mutex
	queuedICE []webrtc.ICECandidateInit
	remoteSet bool

	onLocalICE func(*webrtc.ICECandidate)
	onData     func([]byte)
	onClose    func()

	dcOpenCh    chan struct{}
	dcOpenOnce  sync.Once
	closeOnce   sync.Once

	warmupRemaining int
}

const warmupMagic = "OFW0"

func NewSession(api *webrtc.API, cfg webrtc.Configuration, role Role, sessionID string, peerPmid uuid.UUID,
	onLocalICE func(*webrtc.ICECandidate),
	onData func([]byte),
	onClose func(),
	logger *slog.Logger) (*Session, error) {
	if logger == nil {
		logger = slog.Default()
	}
	pc, err := api.NewPeerConnection(cfg)
	if err != nil {
		return nil, err
	}
	s := &Session{
		SessionID:  sessionID,
		PeerPmid:   peerPmid,
		Role:       role,
		pc:         pc,
		logger:     logger,
		onLocalICE: onLocalICE,
		onData:     onData,
		onClose:    onClose,
		dcOpenCh:   make(chan struct{}),
	}
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil || s.onLocalICE == nil {
			return
		}
		s.onLocalICE(c)
	})
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		s.logger.Info("[rtc] PeerConnection state", "sid", s.SessionID, "state", state.String())
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			s.fireClose()
		}
	})
	pc.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		s.logger.Info("[rtc] ICE state", "sid", s.SessionID, "state", state.String())
		if state == webrtc.ICEConnectionStateConnected || state == webrtc.ICEConnectionStateCompleted {
			s.logSelectedCandidatePair()
		}
	})

	if role == RoleAcceptor {
		pc.OnDataChannel(s.attachDataChannel)
	}
	return s, nil
}

func (s *Session) attachDataChannel(dc *webrtc.DataChannel) {
	s.logger.Info("[rtc] DataChannel attached", "label", dc.Label(), "role", s.Role)
	s.mu.Lock()
	s.dc = dc
	s.warmupRemaining = len(warmupMagic)
	s.mu.Unlock()

	dc.OnOpen(func() {
		if err := dc.Send([]byte(warmupMagic)); err != nil {
			s.logger.Debug("[rtc] warmup send failed", "err", err)
		} else {
			s.logger.Debug("[rtc] warmup magic sent")
		}
		s.dcOpenOnce.Do(func() { close(s.dcOpenCh) })
	})
	dc.OnClose(func() {
		s.fireClose()
	})
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		data := msg.Data
		s.mu.Lock()
		remaining := s.warmupRemaining
		s.mu.Unlock()
		if remaining > 0 {
			drop := remaining
			if drop > len(data) {
				drop = len(data)
			}
			s.mu.Lock()
			s.warmupRemaining -= drop
			s.mu.Unlock()
			data = data[drop:]
			if len(data) == 0 {
				return
			}
		}
		if s.onData != nil {
			s.onData(data)
		}
	})
	if dc.ReadyState() == webrtc.DataChannelStateOpen {
		s.dcOpenOnce.Do(func() { close(s.dcOpenCh) })
	}
}

func (s *Session) HandleOffer(offerSDP string) (string, error) {
	err := s.pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  offerSDP,
	})
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.remoteSet = true
	queued := s.queuedICE
	s.queuedICE = nil
	s.mu.Unlock()
	for _, c := range queued {
		if err := s.pc.AddICECandidate(c); err != nil {
			s.logger.Warn("Failed to add queued ICE", "err", err)
		}
	}

	answer, err := s.pc.CreateAnswer(nil)
	if err != nil {
		return "", err
	}
	if err := s.pc.SetLocalDescription(answer); err != nil {
		return "", err
	}
	return answer.SDP, nil
}

func (s *Session) CreateOffer() (string, error) {
	if s.Role != RoleInitiator {
		return "", errors.New("CreateOffer only valid for initiator")
	}
	ordered := true
	dc, err := s.pc.CreateDataChannel("minecraft", &webrtc.DataChannelInit{
		Ordered: &ordered,
	})
	if err != nil {
		return "", err
	}
	s.attachDataChannel(dc)

	offer, err := s.pc.CreateOffer(nil)
	if err != nil {
		return "", err
	}
	if err := s.pc.SetLocalDescription(offer); err != nil {
		return "", err
	}
	return offer.SDP, nil
}

func (s *Session) HandleAnswer(answerSDP string) error {
	if s.Role != RoleInitiator {
		return errors.New("HandleAnswer only valid for initiator")
	}
	err := s.pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  answerSDP,
	})
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.remoteSet = true
	queued := s.queuedICE
	s.queuedICE = nil
	s.mu.Unlock()
	for _, c := range queued {
		if err := s.pc.AddICECandidate(c); err != nil {
			s.logger.Warn("Failed to add queued ICE", "err", err)
		}
	}
	return nil
}

func (s *Session) AddRemoteICE(init webrtc.ICECandidateInit) {
	s.mu.Lock()
	if !s.remoteSet {
		s.queuedICE = append(s.queuedICE, init)
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	if err := s.pc.AddICECandidate(init); err != nil {
		s.logger.Warn("Failed to add ICE", "err", err)
	}
}

func (s *Session) WaitDataChannelOpen(ctx context.Context, timeout time.Duration) (*webrtc.DataChannel, error) {
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	select {
	case <-s.dcOpenCh:
		s.mu.Lock()
		dc := s.dc
		s.mu.Unlock()
		if dc == nil {
			return nil, errors.New("data channel was reset")
		}
		return dc, nil
	case <-tctx.Done():
		return nil, tctx.Err()
	}
}

func (s *Session) Send(data []byte) error {
	s.mu.Lock()
	dc := s.dc
	s.mu.Unlock()
	if dc == nil || dc.ReadyState() != webrtc.DataChannelStateOpen {
		return errors.New("data channel not open")
	}
	return dc.Send(data)
}

// WaitForBufferDrain blocks until the SCTP send queue drops below maxBuffered,
// or returns an error if the channel closes / is not open. Callers should use
// this BEFORE Send() to apply TCP backpressure on the upstream socket — without
// it, a fast local TCP feed will overflow SCTP and Pion may silently drop.
func (s *Session) WaitForBufferDrain(maxBuffered uint64) error {
	s.mu.Lock()
	dc := s.dc
	s.mu.Unlock()
	if dc == nil {
		return errors.New("data channel not attached")
	}
	for {
		if dc.ReadyState() != webrtc.DataChannelStateOpen {
			return errors.New("data channel not open")
		}
		if dc.BufferedAmount() <= maxBuffered {
			return nil
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// logSelectedCandidatePair logs which ICE candidate pair won — host (LAN),
// srflx (P2P through NAT), prflx, or relay (TURN). The latter implies all
// bytes go through a TURN relay and bandwidth is capped by the relay server.
func (s *Session) logSelectedCandidatePair() {
	sctp := s.pc.SCTP()
	if sctp == nil {
		return
	}
	dtls := sctp.Transport()
	if dtls == nil {
		return
	}
	ice := dtls.ICETransport()
	if ice == nil {
		return
	}
	pair, err := ice.GetSelectedCandidatePair()
	if err != nil || pair == nil {
		s.logger.Info("[rtc] selected pair unavailable", "sid", s.SessionID, "err", errString(err))
		return
	}
	s.logger.Info("[rtc] selected pair",
		"sid", s.SessionID,
		"local", pair.Local.Typ.String(),
		"remote", pair.Remote.Typ.String(),
		"localAddr", pair.Local.Address+":"+itoa(int(pair.Local.Port)),
		"remoteAddr", pair.Remote.Address+":"+itoa(int(pair.Remote.Port)),
		"path", describePath(pair.Local.Typ, pair.Remote.Typ),
	)
}

func describePath(local, remote webrtc.ICECandidateType) string {
	if local == webrtc.ICECandidateTypeRelay || remote == webrtc.ICECandidateTypeRelay {
		return "TURN-relay (bandwidth limited by relay)"
	}
	if local == webrtc.ICECandidateTypeHost && remote == webrtc.ICECandidateTypeHost {
		return "LAN-direct"
	}
	return "P2P (NAT-traversed, full bandwidth)"
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func itoa(i int) string {
	return strconv.Itoa(i)
}

func (s *Session) fireClose() {
	s.closeOnce.Do(func() {
		if s.onClose != nil {
			s.onClose()
		}
	})
}

func (s *Session) Close() {
	s.mu.Lock()
	dc := s.dc
	pc := s.pc
	s.dc = nil
	s.pc = nil
	s.mu.Unlock()
	if dc != nil {
		_ = dc.Close()
	}
	if pc != nil {
		_ = pc.Close()
	}
}
