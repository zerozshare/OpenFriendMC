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
	"bytes"
	"encoding/json"
	"net/http"
)

func (c *Client) PostPresence(body PresenceRequest) (*PresenceResponse, error) {
	if c.PresenceInCooldown() {
		c.mu.Lock()
		cached := c.presenceCache
		c.mu.Unlock()
		if cached != nil {
			return cached, nil
		}
		return nil, &APIError{Op: "POST /presence", Status: 429, Body: "client cooldown"}
	}
	c.mu.Lock()
	etag := c.presenceETag
	c.mu.Unlock()

	status, etagOut, raw, err := c.doWithRetry("POST /presence", endpointPresence, true, func(et string) *http.Request {
		buf, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", base+"/presence", bytes.NewReader(buf))
		c.authHeader(req)
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Accept", "application/json")
		if et != "" {
			req.Header.Set("If-None-Match", et)
		}
		return req
	}, etag)
	if err != nil {
		return nil, err
	}
	if status == 304 || len(raw) == 0 {
		c.mu.Lock()
		if etagOut != "" {
			c.presenceETag = etagOut
		}
		cached := c.presenceCache
		c.mu.Unlock()
		return cached, nil
	}
	var out PresenceResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.presenceETag = etagOut
	c.presenceCache = &out
	c.mu.Unlock()
	return &out, nil
}
