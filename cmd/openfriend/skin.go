/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 *
 * "Minecraft", "Xbox", "Xbox Live", "Microsoft", and "Mojang" are trademarks
 * of their respective owners. OpenFriend is not affiliated with, endorsed by,
 * sponsored by, or otherwise officially connected to Microsoft Corporation,
 * Mojang AB, or the Xbox brand. See LICENSE for the full notice.
 */
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"jp.zpw.openfriend/internal/api"
	"jp.zpw.openfriend/internal/skinimg"
)

func ensureSkin(client *api.Client, skinPath, variantStr, lockPath string, logger *slog.Logger) error {
	variant, err := parseSkinVariant(variantStr)
	if err != nil {
		return err
	}

	abs, err := filepath.Abs(skinPath)
	if err != nil {
		return err
	}
	res, err := skinimg.Prepare(abs)
	if err != nil {
		return err
	}
	if res.Processed {
		logger.Info("Composed skin from face image", "src", abs,
			"src_size", fmt.Sprintf("%dx%d", res.OriginalW, res.OriginalH))
	}

	sum := sha256.Sum256(res.PNG)
	hash := hex.EncodeToString(sum[:]) + ":" + string(variant)

	if old, err := os.ReadFile(lockPath); err == nil {
		if strings.TrimSpace(string(old)) == hash {
			logger.Info("Skin unchanged, skipping upload", "path", abs)
			return nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		logger.Warn("Read skin lockfile failed", "err", err)
	}

	logger.Info("Uploading skin", "path", abs, "variant", variant, "bytes", len(res.PNG))
	if err := client.UploadSkinBytes(res.PNG, filepath.Base(abs), variant); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(lockPath, []byte(hash), 0o600); err != nil {
		return fmt.Errorf("write skin lockfile: %w", err)
	}
	logger.Info("Skin upload complete")
	return nil
}

func parseSkinVariant(s string) (api.SkinVariant, error) {
	switch strings.ToLower(s) {
	case "", "classic":
		return api.SkinClassic, nil
	case "slim":
		return api.SkinSlim, nil
	default:
		return "", fmt.Errorf("invalid skin variant %q (want classic or slim)", s)
	}
}
