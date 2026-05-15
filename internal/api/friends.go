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

func (c *Client) GetFriends() (*FriendsListResponse, error) {
	if c.FriendsInCooldown() {
		c.mu.Lock()
		cached := c.friendsCache
		c.mu.Unlock()
		if cached != nil {
			return cached, nil
		}
		return nil, &APIError{Op: "GET /friends", Status: 429, Body: "client cooldown"}
	}
	c.mu.Lock()
	etag := c.friendsETag
	c.mu.Unlock()

	status, etagOut, raw, err := c.doWithRetry("GET /friends", endpointFriends, true, func(et string) *http.Request {
		req, _ := http.NewRequest("GET", base+"/friends", nil)
		c.authHeader(req)
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
			c.friendsETag = etagOut
		}
		cached := c.friendsCache
		c.mu.Unlock()
		return cached, nil
	}
	var out FriendsListResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.friendsETag = etagOut
	c.friendsCache = &out
	dir := c.cacheDir
	c.mu.Unlock()
	if dir != "" {
		c.SaveCacheToDisk(dir)
	}
	return &out, nil
}

func (c *Client) PutFriendAction(body FriendActionRequest) (*FriendsListResponse, error) {
	if c.FriendsInCooldown() {
		return nil, &APIError{Op: "PUT /friends", Status: 429, Body: "client cooldown"}
	}
	_, _, raw, err := c.doWithRetry("PUT /friends", endpointFriends, false, func(_ string) *http.Request {
		buf, _ := json.Marshal(body)
		req, _ := http.NewRequest("PUT", base+"/friends", bytes.NewReader(buf))
		c.authHeader(req)
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		req.Header.Set("Accept", "application/json")
		return req
	}, "")
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.friendsETag = ""
	c.mu.Unlock()
	if len(raw) == 0 {
		return nil, nil
	}
	var out FriendsListResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
