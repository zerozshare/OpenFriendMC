/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const fileName = "status.json"

type Snapshot struct {
	Authenticated      bool      `json:"authenticated"`
	ProfileID          string    `json:"profile_id"`
	ProfileName        string    `json:"profile_name"`
	PresenceStatus     string    `json:"presence_status"`
	PresenceRunning    bool      `json:"presence_running"`
	SignalingConnected bool      `json:"signaling_connected"`
	BypassEnabled      bool      `json:"bypass_enabled"`
	Version            string    `json:"version"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type Writer struct {
	path string

	mu   sync.Mutex
	last Snapshot
}

func NewWriter(dataDir string) *Writer {
	return &Writer{path: filepath.Join(dataDir, fileName)}
}

func (w *Writer) Update(mutator func(*Snapshot)) {
	w.mu.Lock()
	mutator(&w.last)
	w.last.UpdatedAt = time.Now().UTC()
	snap := w.last
	w.mu.Unlock()
	w.write(snap)
}

func (w *Writer) write(s Snapshot) {
	if w.path == "" {
		return
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	tmp := w.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return
	}
	_ = os.Rename(tmp, w.path)
}
