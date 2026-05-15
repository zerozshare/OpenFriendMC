/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * Disk persistence for the Friends API cache. Without this, every Core
 * restart re-fetches /friends from Mojang, which trips the 429 rate limit
 * for users who restart frequently (dev/test cycles, crashes).
 *
 * The cache file is plain JSON in the Core data dir. It's safe to delete —
 * the next GET will repopulate.
 */
package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type diskCache struct {
	FriendsETag     string                `json:"friendsETag,omitempty"`
	FriendsResponse *FriendsListResponse  `json:"friendsResponse,omitempty"`
	SavedAtUnix     int64                 `json:"savedAtUnix"`
}

const cacheFilename = "friends-cache.json"

func (c *Client) LoadCacheFromDisk(dir string) {
	if dir == "" {
		return
	}
	c.mu.Lock()
	c.cacheDir = dir
	c.mu.Unlock()
	raw, err := os.ReadFile(filepath.Join(dir, cacheFilename))
	if err != nil {
		return
	}
	var d diskCache
	if err := json.Unmarshal(raw, &d); err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.friendsETag = d.FriendsETag
	c.friendsCache = d.FriendsResponse
	c.logger.Info("Loaded friends cache from disk",
		"saved_at", time.Unix(d.SavedAtUnix, 0).Format(time.RFC3339),
		"has_response", c.friendsCache != nil,
		"etag_set", c.friendsETag != "")
}

func (c *Client) ResetCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.friendsETag = ""
	c.friendsCache = nil
	c.presenceETag = ""
	c.presenceCache = nil
}

func (c *Client) SaveCacheToDisk(dir string) {
	if dir == "" {
		return
	}
	c.mu.Lock()
	d := diskCache{
		FriendsETag:     c.friendsETag,
		FriendsResponse: c.friendsCache,
		SavedAtUnix:     time.Now().Unix(),
	}
	c.mu.Unlock()
	buf, err := json.Marshal(d)
	if err != nil {
		c.logger.Warn("Failed to marshal friends cache", "err", err.Error())
		return
	}
	tmp := filepath.Join(dir, cacheFilename+".tmp")
	if err := os.WriteFile(tmp, buf, 0o600); err != nil {
		c.logger.Warn("Failed to write friends cache tmp", "err", err.Error())
		return
	}
	_ = os.Rename(tmp, filepath.Join(dir, cacheFilename))
}
