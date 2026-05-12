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
	"bytes"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	pemType   = "OPENFRIEND CREDENTIALS"
	legacyExt = ".json"
)

var pemHeader = []byte(
	"# OpenFriend credentials. Treat this file like a password.\n" +
		"# Bound to this machine — copying to another machine will fail.\n" +
		"# If compromised, run `openfriend --reset` and revoke at\n" +
		"# https://account.microsoft.com/privacy/app-access\n",
)

type Store struct {
	Path string
}

func (s *Store) Load() (*Tokens, error) {
	raw, err := os.ReadFile(s.Path)
	if err == nil {
		return s.parsePEM(raw)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if legacy := s.legacyPath(); legacy != "" {
		if raw, err := os.ReadFile(legacy); err == nil {
			t, perr := parseLegacyJSON(raw)
			if perr != nil {
				return nil, perr
			}
			if saveErr := s.Save(t); saveErr == nil {
				_ = os.Remove(legacy)
			}
			return t, nil
		}
	}
	return nil, nil
}

func (s *Store) Save(t *Tokens) error {
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o700); err != nil {
		return err
	}
	body, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}

	var block *pem.Block
	if enc, encErr := encryptMachineBound(body); encErr == nil {
		block = enc
	} else {
		slog.Default().Warn("Machine-bound encryption unavailable; saving credentials as plain PEM",
			"err", encErr)
		block = &pem.Block{Type: pemType, Bytes: body}
	}

	var buf bytes.Buffer
	buf.Write(pemHeader)
	if err := pem.Encode(&buf, block); err != nil {
		return err
	}
	return os.WriteFile(s.Path, buf.Bytes(), 0o600)
}

func (s *Store) parsePEM(raw []byte) (*Tokens, error) {
	block, rest := pem.Decode(raw)
	for block != nil && block.Type != pemType {
		block, rest = pem.Decode(rest)
	}
	if block == nil {
		return nil, fmt.Errorf("no %q block found in credentials file", pemType)
	}
	body := block.Bytes
	if c := block.Headers["Cipher"]; c == cipherMachineBound {
		plain, err := decryptMachineBound(block)
		if err != nil {
			return nil, err
		}
		body = plain
	} else if c != "" {
		return nil, fmt.Errorf("unsupported cipher %q", c)
	}
	var t Tokens
	if err := json.Unmarshal(body, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func parseLegacyJSON(raw []byte) (*Tokens, error) {
	var t Tokens
	if err := json.Unmarshal(raw, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) legacyPath() string {
	if !strings.HasSuffix(s.Path, ".pem") {
		return ""
	}
	return strings.TrimSuffix(s.Path, ".pem") + legacyExt
}
