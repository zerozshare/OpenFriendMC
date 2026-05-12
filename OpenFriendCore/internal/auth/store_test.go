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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestStoreSaveLoadPEM(t *testing.T) {
	dir := t.TempDir()
	store := &Store{Path: filepath.Join(dir, "auth.pem")}

	orig := &Tokens{
		ProfileID:         uuid.MustParse("28dc6a27-f082-48f3-aa67-a42683a133a7"),
		Name:              "Tester",
		MojangAccessToken: "moj-token",
		MojangExpiresAt:   time.Now().Add(time.Hour).Truncate(time.Second),
		MsRefreshToken:    "ms-refresh",
	}
	if err := store.Save(orig); err != nil {
		t.Fatalf("Save: %v", err)
	}

	raw, err := os.ReadFile(store.Path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(raw), "-----BEGIN OPENFRIEND CREDENTIALS-----") {
		t.Fatalf("expected PEM block; got:\n%s", string(raw))
	}
	if !strings.Contains(string(raw), "Treat this file like a password") {
		t.Fatalf("expected human warning header")
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Name != orig.Name || loaded.MsRefreshToken != orig.MsRefreshToken {
		t.Fatalf("mismatch: %+v vs %+v", loaded, orig)
	}
}

func TestStoreLegacyMigration(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "auth.json")
	pemPath := filepath.Join(dir, "auth.pem")
	store := &Store{Path: pemPath}

	orig := &Tokens{
		ProfileID:         uuid.MustParse("28dc6a27-f082-48f3-aa67-a42683a133a7"),
		Name:              "Legacy",
		MojangAccessToken: "moj-token",
		MojangExpiresAt:   time.Now().Add(time.Hour).Truncate(time.Second),
		MsRefreshToken:    "ms-refresh",
	}
	raw, _ := json.Marshal(orig)
	if err := os.WriteFile(legacy, raw, 0o600); err != nil {
		t.Fatalf("seed legacy: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Name != orig.Name {
		t.Fatalf("mismatch: %+v", loaded)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy auth.json should be removed after migration")
	}
	if _, err := os.Stat(pemPath); err != nil {
		t.Fatalf("auth.pem should be created: %v", err)
	}
}

func TestStoreLoadMissing(t *testing.T) {
	dir := t.TempDir()
	store := &Store{Path: filepath.Join(dir, "auth.pem")}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load on missing: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil tokens, got %+v", got)
	}
}

func TestStoreEncryptedRoundtrip(t *testing.T) {
	if _, err := machineKey(); err != nil {
		t.Skipf("machine key unavailable on this system: %v", err)
	}
	dir := t.TempDir()
	store := &Store{Path: filepath.Join(dir, "auth.pem")}

	orig := &Tokens{
		ProfileID:         uuid.MustParse("28dc6a27-f082-48f3-aa67-a42683a133a7"),
		Name:              "Crypto",
		MojangAccessToken: "moj-token-secret",
		MojangExpiresAt:   time.Now().Add(time.Hour).Truncate(time.Second),
		MsRefreshToken:    "ms-refresh-secret",
	}
	if err := store.Save(orig); err != nil {
		t.Fatalf("Save: %v", err)
	}

	raw, _ := os.ReadFile(store.Path)
	if !strings.Contains(string(raw), "Cipher: AES-256-GCM-MachineBound") {
		t.Fatalf("expected machine-bound cipher header; got:\n%s", string(raw))
	}
	if strings.Contains(string(raw), "ms-refresh-secret") {
		t.Fatalf("plaintext token leaked into encrypted file")
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.MsRefreshToken != orig.MsRefreshToken {
		t.Fatalf("roundtrip mismatch: got %q", loaded.MsRefreshToken)
	}
}

func TestStoreTamperDetection(t *testing.T) {
	if _, err := machineKey(); err != nil {
		t.Skipf("machine key unavailable on this system: %v", err)
	}
	dir := t.TempDir()
	store := &Store{Path: filepath.Join(dir, "auth.pem")}
	_ = store.Save(&Tokens{
		ProfileID:      uuid.MustParse("28dc6a27-f082-48f3-aa67-a42683a133a7"),
		Name:           "Tampered",
		MsRefreshToken: "ms-refresh",
	})

	raw, _ := os.ReadFile(store.Path)
	tampered := strings.Replace(string(raw), "GCM-MachineBound", "GCM-MachineBound", 1)
	idx := strings.Index(tampered, "\n\n")
	if idx > 0 && idx+5 < len(tampered) {
		buf := []byte(tampered)
		buf[idx+3] ^= 0xFF
		tampered = string(buf)
	}
	if err := os.WriteFile(store.Path, []byte(tampered), 0o600); err != nil {
		t.Fatalf("write tampered: %v", err)
	}

	if _, err := store.Load(); err == nil {
		t.Fatalf("expected decryption error after tamper, got nil")
	}
}

