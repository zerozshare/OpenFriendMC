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

	dcOpenCh   chan struct{}
	dcOpenOnce sync.Once
}

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
		s.logger.Debug("PeerConnection state", "state", state)
		if state == webrtc.PeerConnectionStateFailed || state == webrtc.PeerConnectionStateClosed {
			if s.onClose != nil {
				s.onClose()
			}
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
	s.mu.Unlock()

	dc.OnOpen(func() {
		s.dcOpenOnce.Do(func() { close(s.dcOpenCh) })
	})
	dc.OnClose(func() {
		if s.onClose != nil {
			s.onClose()
		}
	})
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		if s.onData != nil {
			s.onData(msg.Data)
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
