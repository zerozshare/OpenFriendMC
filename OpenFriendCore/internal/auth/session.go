/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package auth

import (
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

const refreshMargin = 5 * time.Minute

type Session struct {
	store    *Store
	onPrompt func(*DeviceCodePrompt)
	logger   *slog.Logger

	mu     sync.Mutex
	tokens *Tokens
}

func NewSession(store *Store, onPrompt func(*DeviceCodePrompt), logger *slog.Logger) *Session {
	if logger == nil {
		logger = slog.Default()
	}
	return &Session{store: store, onPrompt: onPrompt, logger: logger}
}

func (s *Session) Authenticate() (*Tokens, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	saved, err := s.store.Load()
	if err != nil {
		s.logger.Warn("token store load failed", "err", err)
	}
	if saved != nil && saved.MsRefreshToken != "" {
		s.logger.Info("Using saved refresh token, performing silent refresh")
		t, err := SilentRefresh(saved.MsRefreshToken)
		if err == nil {
			s.tokens = t
			_ = s.store.Save(t)
			return t, nil
		}
		s.logger.Warn("silent refresh failed", "err", err)
	}
	return s.deviceCode()
}

func (s *Session) deviceCode() (*Tokens, error) {
	prompt, err := StartDeviceCode()
	if err != nil {
		return nil, err
	}
	if s.onPrompt != nil {
		s.onPrompt(prompt)
	}
	ms, err := PollForMsToken(prompt, func(msg string) { s.logger.Debug(msg) })
	if err != nil {
		return nil, err
	}
	_ = s.store.Save(&Tokens{
		ProfileID:      uuid.Nil,
		MsRefreshToken: ms.RefreshToken,
	})
	t, err := ExchangeForMojang(ms)
	if err != nil {
		return nil, err
	}
	s.tokens = t
	_ = s.store.Save(t)
	return t, nil
}

func (s *Session) EnsureFresh() (*Tokens, error) {
	s.mu.Lock()
	t := s.tokens
	s.mu.Unlock()
	if t == nil {
		return s.Authenticate()
	}
	if time.Now().Before(t.MojangExpiresAt.Add(-refreshMargin)) {
		return t, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Now().Before(s.tokens.MojangExpiresAt.Add(-refreshMargin)) {
		return s.tokens, nil
	}
	s.logger.Info("Mojang token near expiry, refreshing")
	nt, err := SilentRefresh(s.tokens.MsRefreshToken)
	if err != nil {
		return nil, err
	}
	s.tokens = nt
	_ = s.store.Save(nt)
	return nt, nil
}

func (s *Session) Current() string {
	t, err := s.EnsureFresh()
	if err != nil {
		s.logger.Warn("EnsureFresh failed", "err", err)
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.tokens != nil {
			return s.tokens.MojangAccessToken
		}
		return ""
	}
	return t.MojangAccessToken
}

func (s *Session) RefreshNow() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tokens == nil {
		return ""
	}
	s.logger.Info("Forcing token refresh")
	nt, err := SilentRefresh(s.tokens.MsRefreshToken)
	if err != nil {
		s.logger.Warn("force refresh failed", "err", err)
		return s.tokens.MojangAccessToken
	}
	s.tokens = nt
	_ = s.store.Save(nt)
	return nt.MojangAccessToken
}

func (s *Session) CurrentTokens() *Tokens {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tokens
}
