/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package bypass

import (
	"bytes"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	PEMType  = "OPENFRIEND BYPASS KEY"
	KeyBytes = 32
)

var pemHeader = []byte(
	"# OpenFriend bypass shared secret. The same file must be present on the\n" +
		"# Minecraft server (next to OpenFriendBypass plugin) and the Go bridge.\n" +
		"# Treat this like a password.\n",
)

type Key struct {
	Bytes []byte
}

func Load(path string) (*Key, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(raw)
	if block == nil || block.Type != PEMType {
		return nil, fmt.Errorf("not a %q PEM file", PEMType)
	}
	if len(block.Bytes) != KeyBytes {
		return nil, fmt.Errorf("bypass key must be %d bytes, got %d", KeyBytes, len(block.Bytes))
	}
	return &Key{Bytes: block.Bytes}, nil
}

func Generate(path string) (*Key, error) {
	key := make([]byte, KeyBytes)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.Write(pemHeader)
	if err := pem.Encode(&buf, &pem.Block{Type: PEMType, Bytes: key}); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return nil, err
	}
	return &Key{Bytes: key}, nil
}

func LoadOrAbsent(path string) (*Key, error) {
	k, err := Load(path)
	if err == nil {
		return k, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return nil, err
}
