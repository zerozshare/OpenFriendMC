/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"jp.zpw.openfriend/internal/util"
)

const base = "https://api.minecraftservices.com"

type TokenProvider interface {
	Current() string
	RefreshNow() string
}

type Client struct {
	tokens TokenProvider
	logger *slog.Logger

	mu                  sync.Mutex
	presenceETag        string
	friendsETag         string
	presenceCooldownEnd time.Time
	friendsCooldownEnd  time.Time
}

func NewClient(tokens TokenProvider, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{tokens: tokens, logger: logger}
}

func (c *Client) PresenceInCooldown() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Now().Before(c.presenceCooldownEnd)
}

func (c *Client) PresenceCooldownRemaining() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Until(c.presenceCooldownEnd)
}

func (c *Client) FriendsInCooldown() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Now().Before(c.friendsCooldownEnd)
}

func (c *Client) FriendsCooldownRemaining() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Until(c.friendsCooldownEnd)
}

type APIError struct {
	Op     string
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s failed: %d %s", e.Op, e.Status, briefErr(e.Body))
}

func briefErr(body string) string {
	if len(body) == 0 {
		return ""
	}
	var j map[string]any
	if json.Unmarshal([]byte(body), &j) == nil {
		if v, ok := j["error"].(string); ok {
			return v
		}
		if v, ok := j["errorMessage"].(string); ok {
			return v
		}
	}
	if len(body) > 200 {
		return body[:200] + "..."
	}
	return body
}

type doRequest func(etag string) *http.Request

type endpoint int

const (
	endpointPresence endpoint = iota
	endpointFriends
)

func (c *Client) doWithRetry(op string, ep endpoint, supportsEtag bool, build doRequest, etagIn string) (status int, etagOut string, body []byte, err error) {
	retried := false
	for {
		req := build(etagIfEnabled(supportsEtag, etagIn))
		resp, derr := util.Client.Do(req)
		if derr != nil {
			return 0, etagIn, nil, &APIError{Op: op, Status: 0, Body: derr.Error()}
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 304 {
			return 304, etagIn, nil, nil
		}
		if resp.StatusCode < 400 {
			return resp.StatusCode, firstNonEmpty(resp.Header.Get("ETag"), etagIn), raw, nil
		}
		if resp.StatusCode == 401 && !retried {
			c.logger.Info("API 401; forcing token refresh and retrying", "op", op)
			c.tokens.RefreshNow()
			retried = true
			continue
		}
		if resp.StatusCode == 429 {
			d := parseRetryAfter(resp.Header.Get("Retry-After"))
			if d == 0 {
				d = 60 * time.Second
			}
			c.mu.Lock()
			switch ep {
			case endpointPresence:
				c.presenceCooldownEnd = time.Now().Add(d)
			case endpointFriends:
				c.friendsCooldownEnd = time.Now().Add(d)
			}
			c.mu.Unlock()
			c.logger.Warn("API 429 rate limited", "op", op, "cooldown_s", d.Seconds())
			return resp.StatusCode, etagIn, raw, &APIError{Op: op, Status: 429, Body: string(raw)}
		}
		return resp.StatusCode, etagIn, raw, &APIError{Op: op, Status: resp.StatusCode, Body: string(raw)}
	}
}

func (c *Client) authHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.tokens.Current())
}

func etagIfEnabled(enabled bool, etag string) string {
	if !enabled {
		return ""
	}
	return etag
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func parseRetryAfter(h string) time.Duration {
	if h == "" {
		return 0
	}
	if n, err := strconv.Atoi(h); err == nil {
		return time.Duration(n) * time.Second
	}
	return 0
}
