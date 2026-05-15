/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 */
package ipc

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/google/uuid"
)

type Blocklist struct {
	path string
	mu   sync.Mutex
	ids  map[uuid.UUID]string
}

func LoadBlocklist(path string) (*Blocklist, error) {
	b := &Blocklist{path: path, ids: map[uuid.UUID]string{}}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return b, nil
		}
		return nil, err
	}
	var rows []struct {
		ID   string `json:"profileId"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	for _, r := range rows {
		if id, err := uuid.Parse(r.ID); err == nil {
			b.ids[id] = r.Name
		}
	}
	return b, nil
}

func (b *Blocklist) save() error {
	rows := make([]map[string]string, 0, len(b.ids))
	for id, name := range b.ids {
		rows = append(rows, map[string]string{"profileId": id.String(), "name": name})
	}
	buf, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(b.path, buf, 0o600)
}

func (b *Blocklist) Has(id uuid.UUID) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, ok := b.ids[id]
	return ok
}

func (b *Blocklist) Add(id uuid.UUID, name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.ids[id] = name
	return b.save()
}

func (b *Blocklist) Remove(id uuid.UUID) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.ids, id)
	return b.save()
}

func (b *Blocklist) Snapshot() []map[string]string {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]map[string]string, 0, len(b.ids))
	for id, name := range b.ids {
		out = append(out, map[string]string{"profileId": id.String(), "name": name})
	}
	return out
}
